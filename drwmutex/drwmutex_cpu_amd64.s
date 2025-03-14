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
TEXT ·getCurrentCPUViaRDTSCP(SB), NOSPLIT, $8-4 // Note: changed stack size to 8
        // Save registers since RDTSCP modifies EDX and EAX too
        MOVQ CX, 0(SP)
        RDTSCP                  // Returns TSC in EDX:EAX, CPU ID in ECX
        MOVL CX, AX            // Move CPU ID from CX to AX
        ANDL $0xff, AX         // Mask to get just the CPU ID
        MOVL AX, ret+0(FP)     // Store result
        // Restore saved register
        MOVQ 0(SP), CX
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

// func debugRDTSCP() (cpu, eax, edx uint32)
TEXT ·debugRDTSCP(SB), NOSPLIT, $8-12
    MOVQ CX, 0(SP)
    RDTSCP
    MOVL CX, cpu+0(FP)     // Save raw CPU ID
    MOVL AX, eax+4(FP)     // Save low 32 bits of TSC
    MOVL DX, edx+8(FP)     // Save high 32 bits of TSC
    MOVQ 0(SP), CX
    RET

// func asmRdtscpAsm() (eax, ebx, ecx, edx uint32)
TEXT ·asmRdtscpAsm(SB), 7, $0
	BYTE $0x0F; BYTE $0x01; BYTE $0xF9 // RDTSCP
	MOVL AX, eax+0(FP)
	MOVL BX, ebx+4(FP)
	MOVL CX, ecx+8(FP)
	MOVL DX, edx+12(FP)
	RET

