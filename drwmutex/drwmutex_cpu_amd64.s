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

// func tryRDPID() (cpu uint32, ok bool)
TEXT ·tryRDPID(SB), NOSPLIT, $0-8
    // Try RDPID
    BYTE $0xF3        // REP prefix
    BYTE $0x0F        // 2-byte opcode
    BYTE $0xC7        // 2-byte opcode
    BYTE $0xF8        // ModR/M byte for RDPID
    MOVL AX, cpu+0(FP)
    MOVB $1, ok+4(FP)
    RET

NOP_RDPID:
    MOVL $0, cpu+0(FP)
    MOVB $0, ok+4(FP)
    RET
