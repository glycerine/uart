//go:build darwin
// +build darwin

package drwmutex

import (
	"runtime"
)

func map_cpus() (cpus map[int]int) {
	cpus = make(map[int]int)
	nCPU := runtime.NumCPU()
	for i := range nCPU {
		cpus[i] = i // darwin already sequential
	}
	return
}
