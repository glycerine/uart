package drwmutex

/* for the drwmutex.go code
MIT License

Copyright (c) 2019 Jon Gjengset

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

// package comment:

// package drwmutex provides a DRWMutex, a distributed RWMutex for use when
// there are many readers spread across many cores, and relatively few cores.
// DRWMutex is meant as an almost drop-in replacement for sync.RWMutex.

# Distributed Read-Write Mutex in Go

The default Go implementation of
[sync.RWMutex](https://golang.org/pkg/sync/#RWMutex) does not scale well
to multiple cores, as all readers contend on the same memory location
when they all try to atomically increment it. This repository provides
an `n`-way RWMutex, also known as a "big reader" lock, which gives each
CPU core its own RWMutex. Readers take only a read lock local to their
core, whereas writers must take all locks in order.

**Note that the current implementation only supports x86 processors on
Linux; other combinations will revert (automatically) to the old
sync.RWMutex behaviour. To support other architectures and OSes, the
appropriate `cpu_GOARCH.go` and `cpus_GOOS.go` files need to be written.
If you have a different setup available, and have the time to write one
of these, I'll happily accept patches.**

## Finding the current CPU

To determine which lock to take, readers use the CPUID instruction,
which gives the APICID of the currently active CPU without having to
issue a system call or modify the runtime. This instruction is supported
on both Intel and AMD processors; ARM CPUs should use the [CPU ID
register](http://infocenter.arm.com/help/index.jsp?topic=/com.arm.doc.ddi0360e/CACEDHJG.html)
instead. For systems with more than 256 processors, x2APIC must be used,
and the EDX register after CPUID with EAX=0xb should be used instead. A
mapping from APICID to CPU index is constructed (using CPU affinity
syscalls) when the program is started, as it is static for the lifetime
of a process.  Since the CPUID instruction can be fairly expensive,
goroutines will also only periodically update their estimate of what
core they are running on.  More frequent updates lead to less inter-core
lock traffic, but also increases the time spent on CPUID instructions
relative to the actual locking.

**Stale CPU information.**
The information of which CPU a goroutine is running on *might* be stale
when we take the lock (the goroutine could have been moved to another
core), but this will only affect performance, not correctness, as long
as the reader remembers which lock it took. Such moves are also
unlikely, as the OS kernel tries to keep threads on the same core to
improve cache hits.

## Performance

There are many parameters that affect the performance characteristics of
this scheme. In particular, the frequency of CPUID checking, the number
of readers, the ratio of readers to writers, and the time readers hold
their locks, are all important. Since only a single writer is active at
the time, the duration a writer holds a lock for does not affect the
difference in performance between sync.RWMutex and DRWMutex.

Experiments show that DRWMutex performs better the more cores the system
has, and in particular when the fraction of writers is <1%, and CPUID is
called at most every 10 locks (this changes depending on the duration a
lock is held for). Even on few cores, DRWMutex outperforms sync.RWMutex
under these conditions, which are common for applications that elect to
use sync.RWMutex over sync.Mutex.

The plot below shows mean performance across 30 runs (using
[experiment](https://github.com/jonhoo/experiment)) as the number of
cores increases using:

    drwmutex-bench -i 5000 -p 0.0001 -n 500 -w 1 -r 100 -c 100

![DRWMutex and sync.RWMutex performance comparison](benchmarks/perf.png)

Error bars denote 25th and 75th percentile.
Note the drops every 10th core; this is because 10 cores constitute a
NUMA node on the machine the benchmarks were run on, so once a NUMA node
is added, cross-core traffic becomes more expensive. Performance
increases for DRWMutex as more readers can work in parallel compared to
sync.RWMutex.

See the [go-nuts
thread](https://groups.google.com/d/msg/golang-nuts/zt_CQssHw4M/TteNG44geaEJ)
for further discussion.

*/

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/klauspost/cpuid"
)

/*
Linux / Zen.
go test -v
48/48 cpus found in 20.866431ms: map[0:0 1:24 2:1 3:25 4:2 5:26 8:3 9:27 10:4 11:28 12:5 13:29 16:6 17:30 18:7 19:31 20:8 21:32 24:9 25:33 26:10 27:34 28:11 29:35 32:12 33:36 34:13 35:37 36:14 37:38 40:15 41:39 42:16 43:40 44:17 45:41 48:18 49:42 50:19 51:43 52:20 53:44 56:21 57:45 58:22 59:46 60:23 61:47]

=== RUN   TestDrwmutex
runtime.NumCPU() = 48
logcpu = 45; rdtscp = 41; PDPID = 41
i=0 LogicalCPUID from klauspost/cpuid: 45
logcpu = 29; rdtscp = 35; PDPID = 35
i=1 LogicalCPUID from klauspost/cpuid: 29
logcpu = 45; rdtscp = 41; PDPID = 41
i=1 LogicalCPUID from klauspost/cpuid: 45
logcpu = 29; rdtscp = 35; PDPID = 35
i=0 LogicalCPUID from klauspost/cpuid: 29
logcpu = 45; rdtscp = 41; PDPID = 41
i=1 LogicalCPUID from klauspost/cpuid: 45
logcpu = 34; rdtscp = 13; PDPID = 13
i=0 LogicalCPUID from klauspost/cpuid: 34
logcpu = 45; rdtscp = 41; PDPID = 41
i=1 LogicalCPUID from klauspost/cpuid: 45

logcpu = 24; rdtscp = 9; PDPID = 9
logcpu = 33; rdtscp = 36; PDPID = 36

*/

func Cpu2() (cpu int) {
	// dynamically detects current core, supports many architecture/OS.
	//return uint64(cpuid.CPU.LogicalCPU())

	//t0 := time.Now()
	rdpid, ok := tryRDPID()
	//e0 := time.Since(t0)

	if ok {
		//fmt.Printf("tryRDPID got rdpid = %v in %v\n", rdpid, e0)
		//return int(cpu)
	}

	rdtscp := int(getCurrentCPUViaRDTSCP())
	logcpu := cpuid.CPU.LogicalCPU()
	if rdtscp != 0 {
		//fmt.Printf("RDTSCP was non-zero! logcpu = %v; rdtscp = %v\n", logcpu, rdtscp)
	}

	//mac := MacOSOnlySysctlGetLogicalCPU()

	//if logcpu != rdtscp {
	fmt.Printf("logcpu = %v; rdtscp = %v; RDPID = %v;\n", logcpu, rdtscp, rdpid) // , mac)
	//}
	return logcpu
}

// cpus maps (non-consecutive) CPUID values to integer indices.
var cpus map[int]int

// init will construct the cpus map so that CPUIDs can be looked up to
// determine a particular core's lock index.
func init() {
	start := time.Now()
	cpus = map_cpus() // from from APICID -> "processor"
	fmt.Fprintf(os.Stderr, "%d/%d cpus found in %v: %v\n", len(cpus), runtime.NumCPU(), time.Now().Sub(start), cpus)
}

type paddedRWMutex struct {
	_ [5]uint64 // assuming alignment. Pad by cache-line size to prevent false sharing.
	//_  [8]uint64 // Pad by cache-line size to prevent false sharing.

	mu sync.RWMutex
}

type DRWMutex struct {
	slc  []paddedRWMutex
	last RLocker
}

// New returns a new, unlocked, distributed RWMutex.
func NewDRWMutex() *DRWMutex {
	return &DRWMutex{
		slc: make([]paddedRWMutex, len(cpus)),
	}
}

// Lock takes out an exclusive writer lock similar to sync.Mutex.Lock.
// A writer lock also excludes all readers.
func (mx DRWMutex) Lock() {
	for core := range mx.slc {
		mx.slc[core].mu.Lock()
	}
}

// Unlock releases an exclusive writer lock similar to sync.Mutex.Unlock.
func (mx DRWMutex) Unlock() {
	for core := range mx.slc {
		mx.slc[core].mu.Unlock()
	}
}

type RLocker interface {
	RLock()
	RUnlock()
	Lock()
	Unlock()
}

// RLocker returns a sync.Locker presenting Lock() and Unlock() methods that
// take and release a non-exclusive *reader* lock. Note that this call may be
// relatively slow, depending on the underlying system architechture, and so
// its result should be cached if possible.
func (mx DRWMutex) RLocker() *sync.RWMutex { // sync.Locker {
	return &(mx.slc[cpus[Cpu2()]].mu) // .RLocker()
}

// RLock takes out a non-exclusive reader lock, and returns the lock that was
// taken so that it can later be released.
func (mx DRWMutex) RLock() (l *sync.RWMutex) { // RLocker) { // (l sync.Locker) {
	l = &(mx.slc[cpus[Cpu2()]].mu) // .RLocker()
	mx.last = l
	l.Lock()
	return
}
