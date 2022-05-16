//go:build !purego

#include "textflag.h"

DATA ones+0(SB)/4, $1
DATA ones+4(SB)/4, $1
DATA ones+8(SB)/4, $1
DATA ones+12(SB)/4, $1
DATA ones+16(SB)/4, $1
DATA ones+20(SB)/4, $1
DATA ones+24(SB)/4, $1
DATA ones+28(SB)/4, $1
GLOBL ones(SB), RODATA|NOPTR, $32

DATA salt+0(SB)/4, $0x47b6137b
DATA salt+4(SB)/4, $0x44974d91
DATA salt+8(SB)/4, $0x8824ad5b
DATA salt+12(SB)/4, $0xa2b7289d
DATA salt+16(SB)/4, $0x705495c7
DATA salt+20(SB)/4, $0x2df1424b
DATA salt+24(SB)/4, $0x9efc4947
DATA salt+28(SB)/4, $0x5c6bfb31
GLOBL salt(SB), RODATA|NOPTR, $32

// This initial block is a SIMD implementation of the mask function declared in
// block_default.go and block_optimized.go. For each of the 8 x 32 bits words of
// the bloom filter block, the operation performed is:
//
//      block[i] = 1 << ((x * salt[i]) >> 27)
//
// Arguments
// ---------
//
// * src is a memory location where the value to use when computing the mask is
//   located. The memory location is not modified.
//
// * tmp is a YMM register used as scratch space to hold intermediary results in
//   the algorithm.
//
// * dst is a YMM register where the final mask is written.
//
#define generateMask(src, tmp, dst) \
    VMOVDQA ones(SB), dst \
    VPBROADCASTD src, tmp \
    VPMULLD salt(SB), tmp, tmp \
    VPSRLD $27, tmp, tmp \
    VPSLLVD tmp, dst, dst

// func blockInsert(b *Block, x uint32)
TEXT ·blockInsert(SB), NOSPLIT, $0-16
    MOVQ b+0(FP), AX
    generateMask(x+8(FP), Y1, Y0)
    // Set all 1 bits of the mask in the bloom filter block.
    VPOR (AX), Y0, Y0
    VMOVDQU Y0, (AX)
    VZEROUPPER
    RET

// func blockCheck(b *Block, x uint32) bool
TEXT ·blockCheck(SB), NOSPLIT, $0-17
    MOVQ b+0(FP), AX
    generateMask(x+8(FP), Y1, Y0)
    // Compare the 1 bits of the mask with the bloom filter block, then compare
    // the result with the mask, expecting equality if the value `x` was present
    // in the block.
    VPAND (AX), Y0, Y1 // Y0 = block & mask
    VPTEST Y0, Y1      // if (Y0 & ^Y1) != 0 { CF = 1 }
    SETCS ret+16(FP)   // return CF == 1
    VZEROUPPER
    RET
