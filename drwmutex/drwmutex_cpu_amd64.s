#include "textflag.h"

// func cpu() uint64
TEXT ·cpu(SB),NOSPLIT,$0-8
	MOVL	$0x01, AX // version information
	MOVL	$0x00, BX // any leaf will do
	MOVL	$0x00, CX // any subleaf will do

	// call CPUID
	BYTE $0x0f
	BYTE $0xa2

	SHRQ	$24, BX // logical cpu id is put in EBX[31-24]
	MOVQ	BX, ret+0(FP)
	RET

// func getCurrentCPUViaRDTSCP() uint32
TEXT ·getCurrentCPUViaRDTSCP(SB), NOSPLIT, $0-4
RDTSCP
MOVL CX, AX    // CPU ID is in ECX
ANDL $0xff, AX // Mask to get just the CPU ID
MOVL AX, ret+0(FP)
RET
