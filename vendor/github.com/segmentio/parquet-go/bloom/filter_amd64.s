//go:build !purego

#include "textflag.h"

// See block_amd64.s for a description of this algorithm.
#define generateMask(src, dst) \
    VMOVDQA ones(SB), dst \
    VPMULLD salt(SB), src, src \
    VPSRLD $27, src, src \
    VPSLLVD src, dst, dst

#define applyMask(src, dst) \
    VPOR dst, src, src \
    VMOVDQU src, dst

#define fasthash1x64(scale, value) \
    SHRQ $32, value \
    IMULQ scale, value \
    SHRQ $32, value \
    SHLQ $5, value

#define fasthash4x64(scale, value) \
    VPSRLQ $32, value, value \
    VPMULUDQ scale, value, value \
    VPSRLQ $32, value, value \
    VPSLLQ $5, value, value

#define extract4x64(srcYMM, srcXMM, tmpXMM, r0, r1, r2, r3) \
    VEXTRACTI128 $1, srcYMM, tmpXMM \
    MOVQ srcXMM, r0 \
    VPEXTRQ $1, srcXMM, r1 \
    MOVQ tmpXMM, r2 \
    VPEXTRQ $1, tmpXMM, r3

// func filterInsertBulk(f []Block, x []uint64)
TEXT ·filterInsertBulk(SB), NOSPLIT, $0-48
    MOVQ f_base+0(FP), AX
    MOVQ f_len+8(FP), CX
    MOVQ x_base+24(FP), BX
    MOVQ x_len+32(FP), DX
    VPBROADCASTQ f_base+8(FP), Y0

    // Loop initialization, SI holds the current index in `x`, DI is the number
    // of elements in `x` rounded down to the nearest multiple of 4.
    XORQ SI, SI
    MOVQ DX, DI
    SHRQ $2, DI
    SHLQ $2, DI
loop4x64:
    CMPQ SI, DI
    JAE loop

    // The masks and indexes for 4 input hashes are computed in each loop
    // iteration. The hashes are loaded in Y1 so we can use vector instructions
    // to compute all 4 indexes in parallel. The lower 32 bits of the hashes are
    // also broadcasted in 4 YMM registers to compute the 4 masks that will then
    // be applied to the filter.
    VMOVDQU (BX)(SI*8), Y1
    VPBROADCASTD 0(BX)(SI*8), Y2
    VPBROADCASTD 8(BX)(SI*8), Y3
    VPBROADCASTD 16(BX)(SI*8), Y4
    VPBROADCASTD 24(BX)(SI*8), Y5

    fasthash4x64(Y0, Y1)
    generateMask(Y2, Y6)
    generateMask(Y3, Y7)
    generateMask(Y4, Y8)
    generateMask(Y5, Y9)

    // The next block of instructions move indexes from the vector to general
    // purpose registers in order to use them as offsets when applying the mask
    // to the filter.
    extract4x64(Y1, X1, X10, R8, R9, R10, R11)

    // Apply masks to the filter; this operation is sensitive to aliasing, when
    // blocks overlap the, CPU has to serialize the reads and writes, which has
    // a measurable impact on throughput. This would be frequent for small bloom
    // filters which may have only a few blocks, the probability of seeing
    // overlapping blocks on large filters should be small enough to make this
    // a non-issue though.
    applyMask(Y6, (AX)(R8*1))
    applyMask(Y7, (AX)(R9*1))
    applyMask(Y8, (AX)(R10*1))
    applyMask(Y9, (AX)(R11*1))

    ADDQ $4, SI
    JMP loop4x64
loop:
    // Compute trailing elements in `x` if the length was not a multiple of 4.
    // This is the same algorithm as the one in the loop4x64 section, working
    // on a single mask/block pair at a time.
    CMPQ SI, DX
    JE done
    MOVQ (BX)(SI*8), R8
    VPBROADCASTD (BX)(SI*8), Y0
    fasthash1x64(CX, R8)
    generateMask(Y0, Y1)
    applyMask(Y1, (AX)(R8*1))
    INCQ SI
    JMP loop
done:
    VZEROUPPER
    RET

// func filterInsert(f []Block, x uint64)
TEXT ·filterInsert(SB), NOSPLIT, $0-32
    MOVQ f_base+0(FP), AX
    MOVQ f_len+8(FP), BX
    MOVQ x+24(FP), CX
    VPBROADCASTD x+24(FP), Y1
    fasthash1x64(BX, CX)
    generateMask(Y1, Y0)
    applyMask(Y0, (AX)(CX*1))
    VZEROUPPER
    RET

// func filterCheck(f []Block, x uint64) bool
TEXT ·filterCheck(SB), NOSPLIT, $0-33
    MOVQ f_base+0(FP), AX
    MOVQ f_len+8(FP), BX
    MOVQ x+24(FP), CX
    VPBROADCASTD x+24(FP), Y1
    fasthash1x64(BX, CX)
    generateMask(Y1, Y0)
    VPAND (AX)(CX*1), Y0, Y1
    VPTEST Y0, Y1
    SETCS ret+32(FP)
    VZEROUPPER
    RET
