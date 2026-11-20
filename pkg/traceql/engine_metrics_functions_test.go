package traceql

import (
	"math"
	"testing"
	"time"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	commonv1proto "github.com/grafana/tempo/v3/pkg/tempopb/common/v1"
	"github.com/stretchr/testify/require"
)

func TestAccumulators(t *testing.T) {
	type obs struct{ v, w float64 }
	nanObs := obs{math.NaN(), 1}

	tcs := []struct {
		name     string
		acc      accumulator
		in       []obs
		expected float64
	}{
		{"sum empty", ptr(emptySumAcc), nil, math.NaN()},
		{"sum only NaN", ptr(emptySumAcc), []obs{nanObs}, math.NaN()},
		{"sum skips NaN", ptr(emptySumAcc), []obs{{1, 1}, nanObs, {2, 1}}, 3},
		{"sum compensates", ptr(emptySumAcc), []obs{{1, 1}, {1e100, 1}, nanObs, {1, 1}, {-1e100, 1}}, 2},
		{"sum weights", ptr(emptySumAcc), []obs{{2, 3}, {1, 1}}, 7},
		{"min empty", ptr(emptyMinAcc), nil, math.NaN()},
		{"min skips NaN", ptr(emptyMinAcc), []obs{{3, 1}, nanObs, {-1, 1}, {2, 1}}, -1},
		{"min ignores weight", ptr(emptyMinAcc), []obs{{3, 10}}, 3},
		{"max empty", ptr(emptyMaxAcc), nil, math.NaN()},
		{"max skips NaN", ptr(emptyMaxAcc), []obs{{-3, 1}, nanObs, {1, 1}, {-2, 1}}, 1},
		{"avg empty", ptr(emptyAvgAcc), nil, math.NaN()},
		{"avg skips NaN", ptr(emptyAvgAcc), []obs{{1, 1}, nanObs, {3, 1}}, 2},
		// Partial results arrive as (mean, count).
		{"avg merges partials", ptr(emptyAvgAcc), []obs{{150, 2}, {600, 3}}, 420},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			for _, o := range tc.in {
				tc.acc.add(o.v, o.w)
			}
			if math.IsNaN(tc.expected) {
				require.True(t, math.IsNaN(tc.acc.value()))
				return
			}
			require.Equal(t, tc.expected, tc.acc.value())
		})
	}
}

// sum stays NaN once Inf + -Inf poisons it, instead of restarting from the next value.
func TestSumAccStaysNaNAfterInfCancel(t *testing.T) {
	a := emptySumAcc
	a.add(math.Inf(1), 1)
	a.add(math.Inf(-1), 1)
	a.add(5, 1)
	require.True(t, math.IsNaN(a.value()))
}

func ptr[T any](v T) *T { return &v }

func TestAvgAccCount(t *testing.T) {
	a := emptyAvgAcc
	require.True(t, math.IsNaN(a.count()))
	a.add(1, 2)
	a.add(math.NaN(), 5)
	a.add(4, 1)
	require.Equal(t, 3.0, a.count())
	require.Equal(t, 2.0, a.value())
}

// A huge value followed by many small ones used to drift: the running mean
// subtracted the Kahan compensation on every merge.
func TestAvgAccMergePrecision(t *testing.T) {
	a := emptyAvgAcc
	a.add(1e16, 1)
	for range 99 {
		a.add(1, 1)
	}
	// Exact mean is 100000000000000.99; one ulp at 1e14 is ~0.016. The old code was off by ~14.
	require.InDelta(t, 100000000000000.99, a.value(), 0.02)
}

// Accumulator state must not leak between series or intervals.
func TestSimpleCombinerOverTimeIndependentSeries(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Start: uint64(1 * time.Second),
		End:   uint64(3 * time.Second),
		Step:  uint64(time.Second),
	}
	series := func(name string, values ...float64) *tempopb.TimeSeries {
		ts := &tempopb.TimeSeries{
			Labels: []commonv1proto.KeyValue{{Key: "s", Value: &commonv1proto.AnyValue{Value: &commonv1proto.AnyValue_StringValue{StringValue: name}}}},
		}
		for _, v := range values {
			ts.Samples = append(ts.Samples, tempopb.Sample{TimestampMs: 1500, Value: v})
		}
		return ts
	}

	tcs := []struct {
		op   SimpleAggregationOp
		a, b []float64
		expA float64
		expB float64
	}{
		{sumOverTimeAggregation, []float64{1e16, 1, 1}, []float64{5, 5}, 1e16 + 2, 10},
		{minOverTimeAggregation, []float64{3, math.NaN(), 1}, []float64{5, 7}, 1, 5},
		{maxOverTimeAggregation, []float64{3, math.NaN(), 1}, []float64{5, 7}, 3, 7},
		{sumAggregation, []float64{1, 2}, []float64{5, 5}, 3, 10},
	}
	for _, tc := range tcs {
		agg := NewSimpleCombiner(req, tc.op)
		agg.Combine([]*tempopb.TimeSeries{series("a", tc.a...)})
		agg.Combine([]*tempopb.TimeSeries{series("b", tc.b...)})

		// Results can be read between Combines; read twice to make sure it is stable.
		for range 2 {
			res := agg.Results()
			require.Equal(t, tc.expA, res[LabelsFromArgs("s", "a").MapKey()].Values[0])
			require.Equal(t, tc.expB, res[LabelsFromArgs("s", "b").MapKey()].Values[0])
		}
	}
}
