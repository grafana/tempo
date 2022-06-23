//go:build !purego

#include "funcdata.h"
#include "textflag.h"

// func validateLengthValuesAVX2(lengths []int32) (totalLength int, ok bool)
TEXT ·validateLengthValuesAVX2(SB), NOSPLIT, $0-33
    MOVQ lengths_base+0(FP), AX
    MOVQ lengths_len+8(FP), CX

    XORQ BX, BX // totalLength
    XORQ DX, DX // err
    XORQ SI, SI
    XORQ DI, DI
    XORQ R8, R8

    CMPQ CX, $16
    JB test

    MOVQ CX, DI
    SHRQ $4, DI
    SHLQ $4, DI

    VPXOR X0, X0, X0 // totalLengths
    VPXOR X1, X1, X1 // negative test
loopAVX2:
    VMOVDQU (AX)(SI*4), Y2
    VMOVDQU 32(AX)(SI*4), Y3
    VPADDD Y2, Y0, Y0
    VPADDD Y3, Y0, Y0
    VPOR Y2, Y1, Y1
    VPOR Y3, Y1, Y1
    ADDQ $16, SI
    CMPQ SI, DI
    JNE loopAVX2

    // If any of the 32 bit words has its most significant bit set to 1,
    // then at least one of the values was negative, which must be reported as
    // an error.
    VMOVMSKPS Y1, R8
    CMPQ R8, $0
    JNE done

    VPSRLDQ $4, Y0, Y1
    VPSRLDQ $8, Y0, Y2
    VPSRLDQ $12, Y0, Y3

    VPADDD Y1, Y0, Y0
    VPADDD Y3, Y2, Y2
    VPADDD Y2, Y0, Y0

    VPERM2I128 $1, Y0, Y0, Y1
    VPADDD Y1, Y0, Y0
    VZEROUPPER
    MOVQ X0, BX
    ANDQ $0x7FFFFFFF, BX

    JMP test
loop:
    MOVL (AX)(SI*4), DI
    ADDL DI, BX
    ORL DI, R8
    INCQ SI
test:
    CMPQ SI, CX
    JNE loop
    CMPL R8, $0
    JL done
    MOVB $1, DX
done:
    MOVQ BX, totalLength+24(FP)
    MOVB DX, ok+32(FP)
    RET

// This function is an optimization of the decodeLengthByteArray using AVX2
// instructions to implement an opportunistic copy strategy which improves
// throughput compared to using runtime.memmove (via Go's copy).
//
// Parquet columns of type BYTE_ARRAY will often hold short strings, rarely
// exceeding a couple hundred bytes in size. Making a function call to
// runtime.memmove for each value results in spending most of the CPU time
// on branching rather than actually copying bytes to the output buffer.
//
// This function works by always assuming it can copy 16 bytes of data between
// the input and outputs, even in the event where a value is shorter than this.
//
// The pointers to the current positions for input and output pointers are
// always adjusted by the right number of bytes so that the next writes
// overwrite any extra bytes that were written in the previous iteration of the
// copy loop.
//
// The throughput of this function is not as good as runtime.memmove for large
// buffers, but it ends up being close to an order of magnitude higher for the
// common case of working with short strings.
//
// func decodeLengthByteArrayAVX2(dst, src []byte, lengths []int32) int
TEXT ·decodeLengthByteArrayAVX2(SB), NOSPLIT, $0-80
    MOVQ dst_base+0(FP), AX
    MOVQ src_base+24(FP), BX
    MOVQ lengths_base+48(FP), DX
    MOVQ lengths_len+56(FP), DI

    LEAQ (DX)(DI*4), DI
    LEAQ 4(AX), AX
    XORQ CX, CX
    JMP test
loop:
    MOVL (DX), CX
    MOVL CX, -4(AX)
    // First pass moves 16 bytes, this makes it a very fast path for short
    // strings.
    VMOVDQU (BX), X0
    VMOVDQU X0, (AX)
    CMPQ CX, $16
    JA copy
next:
    LEAQ 4(AX)(CX*1), AX
    LEAQ 0(BX)(CX*1), BX
    LEAQ 4(DX), DX
test:
    CMPQ DX, DI
    JNE loop
    MOVQ dst_base+0(FP), BX
    SUBQ BX, AX
    SUBQ $4, AX
    MOVQ AX, ret+72(FP)
    VZEROUPPER
    RET
copy:
    // Values longer than 16 bytes enter this loop and move 32 byte chunks
    // which helps improve throughput on larger chunks.
    MOVQ $16, SI
copyLoop32:
    VMOVDQU (BX)(SI*1), Y0
    VMOVDQU Y0, (AX)(SI*1)
    ADDQ $32, SI
    CMPQ SI, CX
    JAE next
    JMP copyLoop32
