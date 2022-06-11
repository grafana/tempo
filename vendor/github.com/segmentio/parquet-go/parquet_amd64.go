//go:build !purego

package parquet

import "golang.org/x/sys/cpu"

const (
	// For very short inputs we are better off staying in Go code; since the
	// functions get inlined the abstractions have zero overhead then.
	minLenAVX2 = 8
)

var (
	// This variable is used in x86 assembly source files to gate the use of
	// AVX2 instructions depending on whether the CPU supports it.
	hasAVX2     = cpu.X86.HasAVX2
	hasAVX512F  = cpu.X86.HasAVX512F
	hasAVX512VL = cpu.X86.HasAVX512F && cpu.X86.HasAVX512VL
	// For min/max functions over big-endian 128 bits values, we need the
	// follwing instructions from the DQ set:
	// * VPBROADCASTQ (with 64 bits source register)
	// * VBROADCASTI64X2
	hasAVX512MinMaxBE128 = hasAVX512F && cpu.X86.HasAVX512DQ
)

func optimize(n int) bool {
	return n >= minLenAVX2 && hasAVX2
}
