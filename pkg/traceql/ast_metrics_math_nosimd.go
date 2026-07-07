//go:build !(goexperiment.simd && amd64)

package traceql

// SIMD fast paths exist only on amd64 with GOEXPERIMENT=simd. These stubs
// make the callers fall back to the scalar loops.

func applyArithmeticOpVec(Operator, []float64, []float64, []float64) bool { return false }

func applyArithmeticOpScalarVec(Operator, float64, bool, []float64, []float64) bool { return false }
