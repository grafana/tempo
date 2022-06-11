//go:build !purego

#include "textflag.h"

// func broadcastRangeInt32AVX2(dst []int32, base int32)
TEXT ·broadcastRangeInt32AVX2(SB), NOSPLIT, $0-28
    MOVQ dst_base+0(FP), AX
    MOVQ dst_len+8(FP), BX
    MOVL base+24(FP), CX
    XORQ SI, SI

    CMPQ BX, $8
    JB test1x4

    VMOVDQU ·range0n8(SB), Y0         // [0,1,2,3,4,5,6,7]
    VPBROADCASTD ·range0n8+32(SB), Y1 // [8,8,8,8,8,8,8,8]
    VPBROADCASTD base+24(FP), Y2      // [base...]
    VPADDD Y2, Y0, Y0                 // [base,base+1,...]

    MOVQ BX, DI
    SHRQ $3, DI
    SHLQ $3, DI
    JMP test8x4
loop8x4:
    VMOVDQU Y0, (AX)(SI*4)
    VPADDD Y1, Y0, Y0
    ADDQ $8, SI
test8x4:
    CMPQ SI, DI
    JNE loop8x4
    VZEROUPPER
    JMP test1x4

loop1x4:
    INCQ SI
    MOVL CX, DX
    IMULL SI, DX
    MOVL DX, -4(AX)(SI*4)
test1x4:
    CMPQ SI, BX
    JNE loop1x4
    RET

// func writeValuesBitpackAVX2(values unsafe.Pointer, rows array, size, offset uintptr)
TEXT ·writeValuesBitpackAVX2(SB), NOSPLIT, $0-40
    MOVQ values+0(FP), AX
    MOVQ rows_ptr+8(FP), BX
    MOVQ rows_len+16(FP), CX
    MOVQ size+24(FP), DX
    MOVQ offset+32(FP), DI

    CMPQ CX, $0
    JNE init
    RET
init:
    ADDQ DI, BX
    SHRQ $3, CX
    XORQ SI, SI

    // Make sure `size - offset` is at least 4 bytes, otherwise VPGATHERDD
    // may read data beyond the end of the program memory and trigger a fault.
    //
    // If the boolean values do not have enough padding we must fallback to the
    // scalar algorithm to be able to load single bytes from memory.
    MOVQ DX, R8
    SUBQ DI, R8
    CMPQ R8, $4
    JB loop

    VPBROADCASTD size+24(FP), Y0
    VPMULLD ·range0n8(SB), Y0, Y0
    VPCMPEQD Y1, Y1, Y1
    VPCMPEQD Y2, Y2, Y2
    VPCMPEQD Y3, Y3, Y3
    VPSRLD $31, Y3, Y3
avx2loop:
    VPGATHERDD Y1, (BX)(Y0*1), Y4
    VMOVDQU Y2, Y1
    VPAND Y3, Y4, Y4
    VPSLLD $31, Y4, Y4
    VMOVMSKPS Y4, DI

    MOVB DI, (AX)(SI*1)

    LEAQ (BX)(DX*8), BX
    INCQ SI
    CMPQ SI, CX
    JNE avx2loop
    VZEROUPPER
    RET
loop:
    LEAQ (BX)(DX*2), DI
    MOVBQZX (BX), R8
    MOVBQZX (BX)(DX*1), R9
    MOVBQZX (DI), R10
    MOVBQZX (DI)(DX*1), R11
    LEAQ (BX)(DX*4), BX
    LEAQ (DI)(DX*4), DI
    MOVBQZX (BX), R12
    MOVBQZX (BX)(DX*1), R13
    MOVBQZX (DI), R14
    MOVBQZX (DI)(DX*1), R15
    LEAQ (BX)(DX*4), BX

    ANDQ $1, R8
    ANDQ $1, R9
    ANDQ $1, R10
    ANDQ $1, R11
    ANDQ $1, R12
    ANDQ $1, R13
    ANDQ $1, R14
    ANDQ $1, R15

    SHLQ $1, R9
    SHLQ $2, R10
    SHLQ $3, R11
    SHLQ $4, R12
    SHLQ $5, R13
    SHLQ $6, R14
    SHLQ $7, R15

    ORQ R9, R8
    ORQ R11, R10
    ORQ R13, R12
    ORQ R15, R14
    ORQ R10, R8
    ORQ R12, R8
    ORQ R14, R8

    MOVB R8, (AX)(SI*1)

    INCQ SI
    CMPQ SI, CX
    JNE loop
    RET

// func writeValues32bitsAVX2(values unsafe.Pointer, rows array, size, offset uintptr)
TEXT ·writeValues32bitsAVX2(SB), NOSPLIT, $0-40
    MOVQ values+0(FP), AX
    MOVQ rows_ptr+8(FP), BX
    MOVQ rows_len+16(FP), CX
    MOVQ size+24(FP), DX

    XORQ SI, SI
    ADDQ offset+32(FP), BX

    CMPQ CX, $0
    JE done

    CMPQ CX, $8
    JB loop1x4

    MOVQ CX, DI
    SHRQ $3, DI
    SHLQ $3, DI

    VPBROADCASTD size+24(FP), Y0
    VPMULLD ·range0n8(SB), Y0, Y0
    VPCMPEQD Y1, Y1, Y1
    VPCMPEQD Y2, Y2, Y2
loop8x4:
    VPGATHERDD Y1, (BX)(Y0*1), Y3
    VMOVDQU Y3, (AX)(SI*4)
    VMOVDQU Y2, Y1

    LEAQ (BX)(DX*8), BX
    ADDQ $8, SI
    CMPQ SI, DI
    JNE loop8x4
    VZEROUPPER

    CMPQ SI, CX
    JE done

loop1x4:
    MOVL (BX), R8
    MOVL R8, (AX)(SI*4)

    ADDQ DX, BX
    INCQ SI
    CMPQ SI, CX
    JNE loop1x4
done:
    RET

// func writeValues64bitsAVX2(values unsafe.Pointer, rows array, size, offset uintptr)
TEXT ·writeValues64bitsAVX2(SB), NOSPLIT, $0-40
    MOVQ values+0(FP), AX
    MOVQ rows_ptr+8(FP), BX
    MOVQ rows_len+16(FP), CX
    MOVQ size+24(FP), DX

    XORQ SI, SI
    ADDQ offset+32(FP), BX

    CMPQ CX, $0
    JE done

    CMPQ CX, $4
    JB loop1x8

    MOVQ CX, DI
    SHRQ $2, DI
    SHLQ $2, DI

    VPBROADCASTQ size+24(FP), Y0
    VPMULLD ·scale4x8(SB), Y0, Y0
    VPCMPEQD Y1, Y1, Y1
    VPCMPEQD Y2, Y2, Y2
loop4x8:
    VPGATHERQQ Y1, (BX)(Y0*1), Y3
    VMOVDQU Y3, (AX)(SI*8)
    VMOVDQU Y2, Y1

    LEAQ (BX)(DX*4), BX
    ADDQ $4, SI
    CMPQ SI, DI
    JNE loop4x8
    VZEROUPPER

    CMPQ SI, CX
    JE done
loop1x8:
    MOVQ (BX), R8
    MOVQ R8, (AX)(SI*8)

    ADDQ DX, BX
    INCQ SI
    CMPQ SI, CX
    JNE loop1x8
done:
    RET

// func writeValues128bits(values unsafe.Pointer, rows array, size, offset uintptr)
TEXT ·writeValues128bits(SB), NOSPLIT, $0-40
    MOVQ values+0(FP), AX
    MOVQ rows_ptr+8(FP), BX
    MOVQ rows_len+16(FP), CX
    MOVQ size+24(FP), DX
    ADDQ offset+32(FP), BX

    CMPQ CX, $0
    JE done

    CMPQ CX, $1
    JE tail

    XORQ SI, SI
    MOVQ CX, DI
    SHRQ $1, DI
    SHLQ $1, DI
loop:
    MOVOU (BX), X0
    MOVOU (BX)(DX*1), X1

    MOVOU X0, (AX)
    MOVOU X1, 16(AX)

    LEAQ (BX)(DX*2), BX
    ADDQ $32, AX
    ADDQ $2, SI
    CMPQ SI, DI
    JNE loop

    CMPQ SI, CX
    JE done
tail:
    MOVOU (BX), X0
    MOVOU X0, (AX)
done:
    RET

GLOBL ·scale4x8(SB), RODATA|NOPTR, $32
DATA ·scale4x8+0(SB)/8,  $0
DATA ·scale4x8+8(SB)/8,  $1
DATA ·scale4x8+16(SB)/8, $2
DATA ·scale4x8+24(SB)/8, $3
