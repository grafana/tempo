package traceql

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProbabilisticSampler(t *testing.T) {
	s := newProbablisticSampler(0.25)
	require.Equal(t, 4, s.lim)

	var sampled int
	for range 100 {
		if s.Sample() {
			sampled++
		}
	}
	require.Equal(t, 25, sampled)
	require.Equal(t, 4.0, s.FinalScalingFactor())
}

func TestProbabilisticSamplerOutOfRange(t *testing.T) {
	for _, sample := range []float64{2, 1.5, 1, 0, -0.5, -1, math.Inf(1), math.Inf(-1), math.NaN()} {
		s := newProbablisticSampler(sample)
		require.Equal(t, 1, s.lim, "sample=%v", sample)
		for range 10 {
			require.True(t, s.Sample(), "sample=%v", sample)
		}
	}

	s := newProbablisticSampler(1e-300)
	require.Equal(t, math.MaxInt32, s.lim)
	require.True(t, s.Sample())
}

func TestAdaptiveSamplerRate(t *testing.T) {
	testCases := []struct {
		proportion float64
		lim        int
	}{
		// Anything 10% or less is sampled at 100%
		{proportion: 0.01, lim: 1},
		{proportion: 0.05, lim: 1},
		{proportion: 0.1, lim: 1},
		// Between 11% and 20% is sampled at 1 in 2
		{proportion: 0.15, lim: 2},
		{proportion: 0.2, lim: 2},
		{proportion: 0.25, lim: 3},
		{proportion: 0.3, lim: 3},
		{proportion: 0.4, lim: 4},
		{proportion: 0.5, lim: 5},
		{proportion: 0.6, lim: 6},
		{proportion: 0.8, lim: 8},
		{proportion: 0.9, lim: 9},
		{proportion: 1.0, lim: 10}, // in 100% of data, sample 10%
	}

	for _, tc := range testCases {
		require.Equal(t, tc.lim, rateFor(tc.proportion), "sample rate for %f, wanted %d, got %d", tc.proportion, tc.lim, rateFor(tc.proportion))
	}
}
