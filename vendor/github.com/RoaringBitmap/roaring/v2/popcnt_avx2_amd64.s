// +build amd64,!appengine

//go:build amd64 && !appengine

#include "textflag.h"

// AVX2 population-count routines for amd64. They count the set bits across a
// []uint64 (the backing storage of a bitmap container), optionally combining
// each pair of words with a boolean op first: And, Or, Xor, and Mask (s &^ m).
//
// Algorithm (Mula/Lemire VPSHUFB nibble lookup)
// ---------------------------------------------
// AVX2 has no single "popcount a whole vector" instruction, so each byte's
// popcount is taken from a 16-entry lookup table indexed by a 4-bit nibble:
// a byte is split into its low and high nibble, each nibble is looked up (one
// VPSHUFB performs all 32 lookups in a 256-bit register at once), the two
// results are added to give a per-byte popcount, and VPSADBW then sums each
// group of 8 byte-counts into a 64-bit lane total that is accumulated. After
// the loop the four lane totals are summed (HSUM) into a scalar register.
// Each iteration handles 256 bits (4 uint64); a scalar POPCNTQ tail handles
// the trailing len%4 words, so any slice length is counted correctly.
//
// Go assembler conventions used below
// -----------------------------------
//   - Operands are written source(s) first, destination LAST. So
//     "VPAND Ymask, Ydata, Ylo" means Ylo = Ydata AND Ymask.
//   - Yn are the 256-bit AVX registers; Xn aliases the low 128 bits of Yn.
//   - Arguments/results are read from the frame pointer (FP). A Go slice is a
//     3-word header {ptr,len,cap}: s_base+0(FP), s_len+8(FP); a second slice
//     argument starts at +24(FP). The uint64 result slot follows the args
//     (e.g. ret+24(FP) for one slice arg, ret+48(FP) for two).
//   - Every routine is a leaf (makes no calls): NOSPLIT with a $0 local frame.
//   - Loads/stores use VMOVDQU (unaligned): container slices are only 8-byte
//     aligned, not 32. VZEROUPPER precedes every RET to avoid the AVX<->SSE
//     transition penalty in any non-VEX SSE code that runs afterwards.

// lutmask is a 64-byte read-only blob holding two constants used by every
// routine:
//   bytes  0..31 - the nibble popcount table, i.e. table[i] = number of set
//                  bits in the 4-bit value i. VPSHUFB indexes within each
//                  128-bit lane independently, so the 16-entry table is stored
//                  twice (once per lane). Read low-byte-first, the first qword
//                  0x0302020102010100 is the bytes {0,1,1,2,1,2,2,3} for
//                  nibbles 0..7, and 0x0403030203020201 is {1,2,2,3,2,3,3,4}
//                  for nibbles 8..15.
//   bytes 32..63 - 0x0F in every byte: a mask that isolates the low nibble of
//                  each byte.
// RODATA|NOPTR marks it read-only and pointer-free (so the GC ignores it).
DATA lutmask<>+0(SB)/8, $0x0302020102010100
DATA lutmask<>+8(SB)/8, $0x0403030203020201
DATA lutmask<>+16(SB)/8, $0x0302020102010100
DATA lutmask<>+24(SB)/8, $0x0403030203020201
DATA lutmask<>+32(SB)/8, $0x0f0f0f0f0f0f0f0f
DATA lutmask<>+40(SB)/8, $0x0f0f0f0f0f0f0f0f
DATA lutmask<>+48(SB)/8, $0x0f0f0f0f0f0f0f0f
DATA lutmask<>+56(SB)/8, $0x0f0f0f0f0f0f0f0f
GLOBL lutmask<>(SB), RODATA|NOPTR, $64

// Register aliases. Ylut/Ymask/Yzero are constants set up once per call (see
// SETUP); Yacc is the running accumulator of lane totals; Ydata/Yb hold the
// current input vector(s); Ylo/Yhi/Yc1/Yc2 are scratch used by COUNTBLOCK.
#define Ylut Y0
#define Ymask Y1
#define Yzero Y2
#define Yacc Y3
#define Ydata Y4
#define Yb Y5
#define Ylo Y6
#define Yhi Y7
#define Yc1 Y8
#define Yc2 Y9

// COUNTBLOCK folds the popcount of the 32 bytes currently in Ydata into the
// accumulator Yacc. Line by line:
//   VPAND  Ymask,Ydata,Ylo : Ylo = low nibble of every byte
//   VPSRLW $4,Ydata,Yhi    : shift each 16-bit lane right by 4...
//   VPAND  Ymask,Yhi,Yhi   : ...then mask, leaving the high nibble of each byte
//   VPSHUFB Ylo,Ylut,Yc1   : Yc1[b] = popcount(low nibble of byte b)
//   VPSHUFB Yhi,Ylut,Yc2   : Yc2[b] = popcount(high nibble of byte b)
//   VPADDB  Yc2,Yc1,Yc1    : Yc1[b] = popcount(byte b)             (0..8 each)
//   VPSADBW Yzero,Yc1,Yc1  : sum each group of 8 bytes -> 4 lane totals (0..512)
//   VPADDQ  Yc1,Yacc,Yacc  : add the 4 lane totals into the accumulator
// Per-byte counts max at 8 and lane totals at 512, so accumulating across the
// whole loop never overflows the 64-bit lanes.
#define COUNTBLOCK \
	VPAND Ymask, Ydata, Ylo \
	VPSRLW $4, Ydata, Yhi \
	VPAND Ymask, Yhi, Yhi \
	VPSHUFB Ylo, Ylut, Yc1 \
	VPSHUFB Yhi, Ylut, Yc2 \
	VPADDB Yc2, Yc1, Yc1 \
	VPSADBW Yzero, Yc1, Yc1 \
	VPADDQ Yc1, Yacc, Yacc

// SETUP loads the lookup table and nibble mask and zeroes Yzero (the VPSADBW
// addend) and Yacc (the accumulator). Run once at the top of each routine.
#define SETUP \
	VMOVDQU lutmask<>+0(SB), Ylut \
	VMOVDQU lutmask<>+32(SB), Ymask \
	VPXOR Yzero, Yzero, Yzero \
	VPXOR Yacc, Yacc, Yacc

// HSUM reduces Yacc's four 64-bit lane totals to a single sum in AX. X3 is the
// low 128 bits of Yacc (Y3); VEXTRACTI128 pulls the high 128 bits into X5, the
// two halves are added (giving two qwords in X3), and those two qwords are then
// added into AX.
#define HSUM \
	VEXTRACTI128 $1, Yacc, X5 \
	VPADDQ X5, X3, X3 \
	VPEXTRQ $1, X3, DX \
	MOVQ X3, R9 \
	ADDQ R9, AX \
	ADDQ DX, AX

// func _popcntSliceAVX2(s []uint64) uint64
// Returns the total number of set bits in s. This is the canonical routine;
// the And/Or/Xor/Mask variants below share its structure and differ only by
// the boolean op applied before counting.
TEXT ·_popcntSliceAVX2(SB), NOSPLIT, $0-32
	MOVQ s_base+0(FP), SI   // SI = &s[0]
	MOVQ s_len+8(FP), CX    // CX = len(s), in 64-bit words
	XORQ AX, AX             // AX = running result
	SETUP                   // load table/mask; zero Yzero and Yacc
	MOVQ CX, R8
	SHRQ $2, R8             // R8 = len/4 = number of full 256-bit blocks
	TESTQ R8, R8
	JZ slicetail            // fewer than 4 words: skip the vector loop
sliceloop:
	VMOVDQU (SI), Ydata     // load 4 words (32 bytes)
	COUNTBLOCK              // Yacc += popcount(those 32 bytes)
	ADDQ $32, SI            // advance to the next block
	DECQ R8
	JNZ sliceloop
	HSUM                    // AX += sum of Yacc's lane totals
slicetail:
	ANDQ $3, CX             // CX = len % 4 = leftover words (0..3)
	TESTQ CX, CX
	JZ slicedone
slicetailloop:
	MOVQ (SI), DX
	POPCNTQ DX, DX          // scalar popcount of one word
	ADDQ DX, AX
	ADDQ $8, SI
	DECQ CX
	JNZ slicetailloop
slicedone:
	VZEROUPPER              // clear upper YMM state before returning
	MOVQ AX, ret+24(FP)     // return AX
	RET

// func _popcntAndSliceAVX2(s, m []uint64) uint64
// Returns the sum of popcount(s[i] & m[i]). Mirrors _popcntSliceAVX2 but loads
// a vector from each of s and m and ANDs them before counting. s and m are
// assumed to have equal length.
TEXT ·_popcntAndSliceAVX2(SB), NOSPLIT, $0-56
	MOVQ s_base+0(FP), SI    // SI = &s[0]
	MOVQ m_base+24(FP), DI   // DI = &m[0]
	MOVQ s_len+8(FP), CX     // CX = len
	XORQ AX, AX
	SETUP
	MOVQ CX, R8
	SHRQ $2, R8
	TESTQ R8, R8
	JZ andtail
andloop:
	VMOVDQU (SI), Ydata
	VMOVDQU (DI), Yb
	VPAND Yb, Ydata, Ydata   // Ydata = s & m
	COUNTBLOCK
	ADDQ $32, SI
	ADDQ $32, DI
	DECQ R8
	JNZ andloop
	HSUM
andtail:
	ANDQ $3, CX
	TESTQ CX, CX
	JZ anddone
andtailloop:
	MOVQ (SI), DX
	ANDQ (DI), DX            // s & m, one word
	POPCNTQ DX, DX
	ADDQ DX, AX
	ADDQ $8, SI
	ADDQ $8, DI
	DECQ CX
	JNZ andtailloop
anddone:
	VZEROUPPER
	MOVQ AX, ret+48(FP)      // +48: result follows two 24-byte slice headers
	RET

// func _popcntOrSliceAVX2(s, m []uint64) uint64
// Returns the sum of popcount(s[i] | m[i]); see _popcntAndSliceAVX2 for the
// shared structure.
TEXT ·_popcntOrSliceAVX2(SB), NOSPLIT, $0-56
	MOVQ s_base+0(FP), SI
	MOVQ m_base+24(FP), DI
	MOVQ s_len+8(FP), CX
	XORQ AX, AX
	SETUP
	MOVQ CX, R8
	SHRQ $2, R8
	TESTQ R8, R8
	JZ ortail
orloop:
	VMOVDQU (SI), Ydata
	VMOVDQU (DI), Yb
	VPOR Yb, Ydata, Ydata    // Ydata = s | m
	COUNTBLOCK
	ADDQ $32, SI
	ADDQ $32, DI
	DECQ R8
	JNZ orloop
	HSUM
ortail:
	ANDQ $3, CX
	TESTQ CX, CX
	JZ ordone
ortailloop:
	MOVQ (SI), DX
	ORQ (DI), DX             // s | m, one word
	POPCNTQ DX, DX
	ADDQ DX, AX
	ADDQ $8, SI
	ADDQ $8, DI
	DECQ CX
	JNZ ortailloop
ordone:
	VZEROUPPER
	MOVQ AX, ret+48(FP)
	RET

// func _popcntXorSliceAVX2(s, m []uint64) uint64
// Returns the sum of popcount(s[i] ^ m[i]); see _popcntAndSliceAVX2 for the
// shared structure.
TEXT ·_popcntXorSliceAVX2(SB), NOSPLIT, $0-56
	MOVQ s_base+0(FP), SI
	MOVQ m_base+24(FP), DI
	MOVQ s_len+8(FP), CX
	XORQ AX, AX
	SETUP
	MOVQ CX, R8
	SHRQ $2, R8
	TESTQ R8, R8
	JZ xortail
xorloop:
	VMOVDQU (SI), Ydata
	VMOVDQU (DI), Yb
	VPXOR Yb, Ydata, Ydata   // Ydata = s ^ m
	COUNTBLOCK
	ADDQ $32, SI
	ADDQ $32, DI
	DECQ R8
	JNZ xorloop
	HSUM
xortail:
	ANDQ $3, CX
	TESTQ CX, CX
	JZ xordone
xortailloop:
	MOVQ (SI), DX
	XORQ (DI), DX            // s ^ m, one word
	POPCNTQ DX, DX
	ADDQ DX, AX
	ADDQ $8, SI
	ADDQ $8, DI
	DECQ CX
	JNZ xortailloop
xordone:
	VZEROUPPER
	MOVQ AX, ret+48(FP)
	RET

// func _popcntMaskSliceAVX2(s, m []uint64) uint64
// Returns the sum of popcount(s[i] &^ m[i]) == popcount(s & ~m). Same structure
// as _popcntAndSliceAVX2; the combine is VPANDN, which computes (NOT first) AND
// second, i.e. VPANDN Ydata, Yb, Ydata -> Ydata = (NOT Yb) AND Ydata = s &^ m.
TEXT ·_popcntMaskSliceAVX2(SB), NOSPLIT, $0-56
	MOVQ s_base+0(FP), SI
	MOVQ m_base+24(FP), DI
	MOVQ s_len+8(FP), CX
	XORQ AX, AX
	SETUP
	MOVQ CX, R8
	SHRQ $2, R8
	TESTQ R8, R8
	JZ masktail
maskloop:
	VMOVDQU (SI), Ydata
	VMOVDQU (DI), Yb
	VPANDN Ydata, Yb, Ydata  // Ydata = s &^ m  (= (NOT m) AND s)
	COUNTBLOCK
	ADDQ $32, SI
	ADDQ $32, DI
	DECQ R8
	JNZ maskloop
	HSUM
masktail:
	ANDQ $3, CX
	TESTQ CX, CX
	JZ maskdone
masktailloop:
	MOVQ (DI), R10
	NOTQ R10                 // ~m
	MOVQ (SI), DX
	ANDQ R10, DX             // s &^ m = s & ~m, one word
	POPCNTQ DX, DX
	ADDQ DX, AX
	ADDQ $8, SI
	ADDQ $8, DI
	DECQ CX
	JNZ masktailloop
maskdone:
	VZEROUPPER
	MOVQ AX, ret+48(FP)
	RET

// func _hasAVX2() bool
// Reports whether the CPU supports AVX2 and the OS has enabled the wide (YMM)
// register state. All three checks must pass; otherwise the Go wrappers fall
// back to the scalar implementation. Note CPUID clobbers AX/BX/CX/DX.
TEXT ·_hasAVX2(SB), NOSPLIT, $0-1
	// CPUID leaf 1: require OSXSAVE (ECX bit 27) and AVX (ECX bit 28). Both must
	// be set, so mask and compare against the combined bit pattern.
	MOVL $1, AX
	XORL CX, CX
	CPUID
	ANDL $0x18000000, CX
	CMPL CX, $0x18000000
	JNE noavx2
	// XGETBV(0): the OS must have enabled saving of SSE and AVX/YMM state, i.e.
	// XCR0 bits 1 and 2. Without this the YMM registers would be corrupted
	// across a context switch even though the CPU supports the instructions.
	XORL CX, CX
	XGETBV
	ANDL $0x6, AX
	CMPL AX, $0x6
	JNE noavx2
	// CPUID leaf 7, sub-leaf 0: require AVX2 itself (EBX bit 5). The sub-leaf is
	// selected via ECX, which must be 0.
	MOVL $7, AX
	XORL CX, CX
	CPUID
	ANDL $0x20, BX
	CMPL BX, $0x20
	JNE noavx2
	MOVB $1, ret+0(FP)
	RET
noavx2:
	MOVB $0, ret+0(FP)
	RET
