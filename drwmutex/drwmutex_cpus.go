//go:build !linux && !darwin
// +build !linux,!darwin

package drwmutex

func map_cpus() (cpus map[int]int) {
	cpus = make(map[int]int)
	return
}
