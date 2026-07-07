//go:build goexperiment.simd && amd64

package traceql

import (
	"math"
	"simd/archsimd"
)

// vecEnabled gates the vector paths on AVX2, required by Merge (used for the division-by-zero blend).
var vecEnabled = archsimd.X86.AVX2()

func vecSupportedOp(op Operator) bool {
	switch op {
	case OpAdd, OpSub, OpMult, OpDiv:
		return true
	}
	return false
}

func vecArithmeticOp(op Operator, a, b archsimd.Float64x4) archsimd.Float64x4 {
	switch op {
	case OpAdd:
		return a.Add(b)
	case OpSub:
		return a.Sub(b)
	case OpMult:
		return a.Mul(b)
	default: // OpDiv; callers filter with vecSupportedOp
		// Division by zero yields NaN, not ±Inf, matching applyArithmeticOp.
		nan := archsimd.BroadcastFloat64x4(math.NaN())
		zero := archsimd.BroadcastFloat64x4(0)
		return nan.Merge(a.Div(b), b.Equal(zero))
	}
}

// applyArithmeticOpVec computes out[i] = l[i] op r[i] with SIMD. l and r must
// be at least len(out) long. It returns false when the op has no vector form
// or the CPU lacks support, in which case the caller runs the scalar loop.
func applyArithmeticOpVec(op Operator, l, r, out []float64) bool {
	if !vecEnabled || !vecSupportedOp(op) {
		return false
	}

	i := 0
	for ; i+4 <= len(out); i += 4 {
		a := archsimd.LoadFloat64x4Slice(l[i:])
		b := archsimd.LoadFloat64x4Slice(r[i:])
		vecArithmeticOp(op, a, b).StoreSlice(out[i:])
	}
	if i < len(out) {
		a := archsimd.LoadFloat64x4SlicePart(l[i:])
		b := archsimd.LoadFloat64x4SlicePart(r[i:])
		vecArithmeticOp(op, a, b).StoreSlicePart(out[i:])
	}
	return true
}

// applyArithmeticOpScalarVec computes out[i] = scalar op in[i] (scalarOnLeft)
// or out[i] = in[i] op scalar with SIMD. in must be at least len(out) long.
// It returns false when the op has no vector form or the CPU lacks support.
func applyArithmeticOpScalarVec(op Operator, scalar float64, scalarOnLeft bool, in, out []float64) bool {
	if !vecEnabled || !vecSupportedOp(op) {
		return false
	}

	s := archsimd.BroadcastFloat64x4(scalar)
	i := 0
	for ; i+4 <= len(out); i += 4 {
		v := archsimd.LoadFloat64x4Slice(in[i:])
		a, b := v, s
		if scalarOnLeft {
			a, b = s, v
		}
		vecArithmeticOp(op, a, b).StoreSlice(out[i:])
	}
	if i < len(out) {
		v := archsimd.LoadFloat64x4SlicePart(in[i:])
		a, b := v, s
		if scalarOnLeft {
			a, b = s, v
		}
		vecArithmeticOp(op, a, b).StoreSlicePart(out[i:])
	}
	return true
}
