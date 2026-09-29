package benchmark

import (
	"math"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

// A positive value must sit at or below its bucket's upper bound, and a
// negative one strictly inside its magnitude range, or selectivities read from
// bucket sums stop being exact. Values on a bound exercise the correction.
func TestBucketIndexBounds(t *testing.T) {
	for _, m := range []float64{1e-9, 0.5, 1, 1.0219, 2, 3, 1000, 1e9, bucketBound(5), bucketBound(-7)} {
		i := posBucket(m)
		require.Greater(t, m, bucketBound(i-1), "pos %v", m)
		require.LessOrEqual(t, m, bucketBound(i), "pos %v", m)

		j := negBucket(m)
		require.GreaterOrEqual(t, m, bucketBound(j-1), "neg %v", m)
		require.Less(t, m, bucketBound(j), "neg %v", m)
	}
}

// A quantile's selectivity is turned into a `> value` predicate by the
// benchmark, so it must be the exact share of values above the bound.
func TestHistogramQuantileSelectivityIsExact(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	vals := []float64{0, 0, 0, -1, 1}
	for range 5000 {
		vals = append(vals, math.Round(rng.NormFloat64()*1000))
	}

	h := newLogHistogram()
	for _, v := range vals {
		h.add(v, 1)
	}

	// Half the spans carry no value, so selectivity is over twice as many.
	totalSpans := uint64(2 * len(vals))
	qs := []float64{0.01, 0.1, 0.5, 0.9, 0.99}
	got := h.quantiles(qs, totalSpans)
	require.Len(t, got, len(qs))

	for i, q := range got {
		require.Equal(t, qs[i], q.q)

		var above int
		for _, v := range vals {
			if v > q.upper {
				above++
			}
		}
		require.InDelta(t, float64(above)/float64(totalSpans), q.selectivity, 1e-12, "q=%v upper=%v", q.q, q.upper)
		// At least a q share of the values lie at or below the quantile.
		require.GreaterOrEqual(t, float64(len(vals)-above), q.q*float64(len(vals)), "q=%v", q.q)
	}
}

func TestHistogramSkipsNonFinite(t *testing.T) {
	h := newLogHistogram()
	h.add(math.NaN(), 1)
	h.add(math.Inf(1), 1)
	h.add(math.Inf(-1), 1)
	h.add(1, 1)

	require.Equal(t, uint64(1), h.total)
}
