//go:build darwin
// +build darwin

package drwmutex

import (
	"runtime"
)

func map_cpus() (cpus map[uint64]int) {
	cpus = make(map[uint64]int)
	nCPU := runtime.NumCPU()
	for i := range nCPU {
		cpus[uint64(i)] = i // darwin already sequential
	}
	return
}
