//go:build darwin
// +build darwin

package drwmutex

import (
	"runtime"
	//"syscall"
	//"unsafe"
)

func map_cpus() (cpus map[int]int) {
	cpus = make(map[int]int)
	nCPU := runtime.NumCPU()
	for i := range nCPU {
		cpus[i] = i // darwin already sequential
	}
	return
}

/* always returns 0
func MacOSOnlySysctlGetLogicalCPU() int {
	// syscall numbers for macOS/darwin amd64
	const sys_sysctl = 202

	// Define the MIB (Management Information Base) array
	// These values are from xnu source: osfmk/kern/processor.h
	// CTL_HW (6) and HW_AVAILCPU (25)
	mib := []uint32{6, 25}

	// Prepare the arguments for the syscall
	_, _, errno := syscall.RawSyscall6(
		sys_sysctl,
		uintptr(unsafe.Pointer(&mib[0])),
		uintptr(len(mib)),
		0,
		0,
		0,
		0,
	)

	if errno != 0 {
		return -1
	}

	return int(errno) // The CPU ID is returned in the errno value
}
*/
