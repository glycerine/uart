//go:build !amd64
// +build !amd64

package drwmutex

// cpu returns a unique identifier for the core the current goroutine is
// executing on. This function is platform dependent, and is implemented in
// cpu_*.s.
func cpu() uint64 {
	// this reverts the behaviour to that of a regular DRWMutex
	return 0
}

func getCurrentCPUViaRDTSCP() uint32 {
	return 0
}

func tryRDPID() (cpu uint32, ok bool) {
	return
}
