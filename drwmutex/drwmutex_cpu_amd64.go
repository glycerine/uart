package drwmutex

func cpu() uint64

func getCurrentCPUViaRDTSCP() uint32

func tryRDPID() (cpu uint32, ok bool)

func debugRDTSCP() (cpu, eax, edx uint32)
