package traceql

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// vecAvailable reports whether this build has a usable SIMD path
// (GOEXPERIMENT=simd, amd64, AVX2). A zero-length call does no work but
// still returns the availability check.
func vecAvailable() bool {
	return applyArithmeticOpVec(OpAdd, nil, nil, nil)
}

func TestApplyArithmeticOpVecMatchesScalar(t *testing.T) {
	if !vecAvailable() {
		t.Skip("SIMD path not available in this build")
	}

	// Values chosen to hit the special cases: zero divisors, NaN and ±Inf
	// propagation, negative zero.
	specials := []float64{0, math.Copysign(0, -1), 1, -1, 2.5, -3.75, math.NaN(), math.Inf(1), math.Inf(-1), 1e308, 5e-324}
	val := func(i int) float64 { return specials[i%len(specials)] }

	requireSameFloat := func(t *testing.T, want, got float64, i int) {
		t.Helper()
		if math.IsNaN(want) {
			require.True(t, math.IsNaN(got), "index %d: want NaN, got %v", i, got)
			return
		}
		// IEEE 754 basic ops are exact, so vector and scalar must agree bit-for-bit.
		require.Equal(t, want, got, "index %d", i)
	}

	ops := []Operator{OpAdd, OpSub, OpMult, OpDiv}
	sizes := []int{0, 1, 3, 4, 5, 7, 8, 33, 100}

	for _, op := range ops {
		for _, n := range sizes {
			l := make([]float64, n)
			r := make([]float64, n)
			for i := range l {
				l[i] = val(i)
				r[i] = val(i + 3)
			}

			out := make([]float64, n)
			require.True(t, applyArithmeticOpVec(op, l, r, out))
			for i := range out {
				requireSameFloat(t, applyArithmeticOp(op, l[i], r[i]), out[i], i)
			}

			for _, scalarOnLeft := range []bool{true, false} {
				for _, scalar := range []float64{0, 100, -2.5} {
					out := make([]float64, n)
					require.True(t, applyArithmeticOpScalarVec(op, scalar, scalarOnLeft, l, out))
					for i := range out {
						want := applyArithmeticOp(op, l[i], scalar)
						if scalarOnLeft {
							want = applyArithmeticOp(op, scalar, l[i])
						}
						requireSameFloat(t, want, out[i], i)
					}
				}
			}
		}
	}

	// Unsupported ops must decline so callers run the scalar loop.
	require.False(t, applyArithmeticOpVec(OpMod, []float64{1}, []float64{1}, []float64{0}))
	require.False(t, applyArithmeticOpScalarVec(OpPower, 2, false, []float64{1}, []float64{0}))
}

func BenchmarkArithmeticOpKernel(b *testing.B) {
	// 1440 = one day of 1-minute steps, the typical per-series size (cache
	// resident). 100k exceeds L1/L2 to show the memory-bound ceiling.
	for _, n := range []int{100, 1440, 100_000} {
		l := make([]float64, n)
		r := make([]float64, n)
		out := make([]float64, n)
		for i := range l {
			l[i] = float64(i) * 1.5
			r[i] = float64(n - i) // includes one zero divisor
		}

		for _, op := range []Operator{OpAdd, OpDiv} {
			b.Run(fmt.Sprintf("scalar/%s/n=%d", op, n), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					for j := range n {
						out[j] = applyArithmeticOp(op, l[j], r[j])
					}
				}
			})
			b.Run(fmt.Sprintf("vec/%s/n=%d", op, n), func(b *testing.B) {
				if !vecAvailable() {
					b.Skip("SIMD path not available in this build")
				}
				for i := 0; i < b.N; i++ {
					applyArithmeticOpVec(op, l, r, out)
				}
			})
		}
	}
}

// BenchmarkApplyBinaryOp measures the full series ⊙ series path — map
// iteration, label merging, and allocation included — to show what a real
// ratio query gains end to end. Compare runs of a GOEXPERIMENT=simd build
// against a default build; there is no in-binary scalar variant here.
func BenchmarkApplyBinaryOp(b *testing.B) {
	const (
		numSeries = 1000
		numSteps  = 1440
	)

	makeSet := func(seed float64) SeriesSet {
		set := make(SeriesSet, numSeries)
		for s := range numSeries {
			lbls := Labels{{Name: "service.name", Value: NewStaticString(fmt.Sprintf("svc-%d", s))}}
			values := make([]float64, numSteps)
			for i := range values {
				values[i] = seed * float64(s+i+1)
			}
			set[lbls.MapKey()] = TimeSeries{Labels: lbls, Values: values}
		}
		return set
	}
	lhs := makeSet(1.5)
	rhs := makeSet(2.0)
	// sprinkle zero divisors so OpDiv exercises the NaN rule
	for _, ts := range rhs {
		for i := 0; i < len(ts.Values); i += 97 {
			ts.Values[i] = 0
		}
	}

	for _, op := range []Operator{OpAdd, OpDiv} {
		b.Run(op.String(), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				applyBinaryOp(op, lhs, rhs)
			}
		})
	}
}

func TestApplyBinaryOp_Exemplars(t *testing.T) {
	exemplar := func(id string) Exemplar {
		return Exemplar{
			Labels:      Labels{{Name: "trace.id", Value: NewStaticString(id)}},
			Value:       1,
			TimestampMs: 100,
		}
	}
	eEvenL := exemplar("evL")
	eOddL := exemplar("odL")
	eEvenR := exemplar("evR")
	eOddR := exemplar("odR")
	eAll1 := exemplar("a1")
	eAll2 := exemplar("a2")

	svcEven := Labels{{Name: "service.name", Value: NewStaticString("even")}}
	svcOdd := Labels{{Name: "service.name", Value: NewStaticString("odd")}}

	series := func(lbls Labels, ex []Exemplar) TimeSeries {
		return TimeSeries{Labels: lbls, Values: []float64{2}, Exemplars: ex}
	}

	tests := []struct {
		name string
		lhs  SeriesSet
		rhs  SeriesSet
		want map[SeriesMapKey][]Exemplar
	}{
		{
			name: "both grouped same -> merge per key",
			lhs: SeriesSet{
				svcEven.MapKey(): series(svcEven, []Exemplar{eEvenL}),
				svcOdd.MapKey():  series(svcOdd, []Exemplar{eOddL}),
			},
			rhs: SeriesSet{
				svcEven.MapKey(): series(svcEven, []Exemplar{eEvenR}),
				svcOdd.MapKey():  series(svcOdd, []Exemplar{eOddR}),
			},
			want: map[SeriesMapKey][]Exemplar{
				svcEven.MapKey(): {eEvenL, eEvenR},
				svcOdd.MapKey():  {eOddL, eOddR},
			},
		},
		{
			name: "RHS broadcast -> drop RHS exemplars",
			lhs: SeriesSet{
				svcEven.MapKey(): series(svcEven, []Exemplar{eEvenL}),
				svcOdd.MapKey():  series(svcOdd, []Exemplar{eOddL}),
			},
			rhs: SeriesSet{
				noLabelsSeriesMapKey: series(Labels{}, []Exemplar{eAll1, eAll2}),
			},
			want: map[SeriesMapKey][]Exemplar{
				svcEven.MapKey(): {eEvenL},
				svcOdd.MapKey():  {eOddL},
			},
		},
		{
			name: "LHS broadcast -> drop LHS exemplars",
			lhs: SeriesSet{
				noLabelsSeriesMapKey: series(Labels{}, []Exemplar{eAll1, eAll2}),
			},
			rhs: SeriesSet{
				svcEven.MapKey(): series(svcEven, []Exemplar{eEvenR}),
				svcOdd.MapKey():  series(svcOdd, []Exemplar{eOddR}),
			},
			want: map[SeriesMapKey][]Exemplar{
				svcEven.MapKey(): {eEvenR},
				svcOdd.MapKey():  {eOddR},
			},
		},
		{
			name: "both ungrouped -> merge",
			lhs: SeriesSet{
				noLabelsSeriesMapKey: series(Labels{}, []Exemplar{eAll1}),
			},
			rhs: SeriesSet{
				noLabelsSeriesMapKey: series(Labels{}, []Exemplar{eAll2}),
			},
			want: map[SeriesMapKey][]Exemplar{
				noLabelsSeriesMapKey: {eAll1, eAll2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := applyBinaryOp(OpDiv, tt.lhs, tt.rhs)
			require.Len(t, out, len(tt.want))
			for k, want := range tt.want {
				require.Equal(t, want, out[k].Exemplars)
			}
		})
	}
}
