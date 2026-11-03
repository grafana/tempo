package compare

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
)

func TestSummary(t *testing.T) {
	run := func(name string, size int, p50 float64, matched int64) Run {
		r := newResult("mac",
			caseResult("traceid/present", 1, metrics.Set{"harness.wallNs": measurement(p50)}),
			caseResult("search/nopredicate", matched, metrics.Set{"harness.wallNs": measurement(0)}),
		)
		r.Options.ReadBufferSize = size
		return Run{Name: name, Result: r}
	}
	c, err := New([]Run{
		run("base", 0, 100, 5),
		run("2MiB", 2<<20, 98, 5),
		run("4MiB", 4<<20, 87, 6),
	})
	require.NoError(t, err)
	c.Runs[1].Result.Cases[0].Metrics = nil // 2MiB did not report it

	sm := c.Summary("harness.wallNs", P90, 0)
	require.Equal(t, "harness.wallNs", sm.Metric)
	require.Equal(t, P90, sm.Stat)
	require.Equal(t, Nanoseconds, sm.Unit)
	require.Len(t, sm.Rows, 2)
	require.Equal(t, 0, sm.Rows[0].Case)

	present := sm.Rows[0].Cells
	require.Equal(t, SummaryCell{Value: 140, HasValue: true}, present[0], "the baseline has a value and no change")
	require.Equal(t, SummaryCell{}, present[1], "a run without data has neither")
	require.True(t, present[2].HasValue)
	require.InDelta(t, 121.8, present[2].Value, 1e-9, "the stat asked for, p90")
	require.True(t, present[2].HasDelta)
	require.InDelta(t, -13.0, present[2].Delta, 1e-9)

	search := sm.Rows[1].Cells
	require.Equal(t, SummaryCell{Value: 0, HasValue: true}, search[1], "no change from a baseline of zero")
	require.Equal(t, SummaryCell{Incomparable: "matched 5 vs 6"}, search[2])
}

func TestFamilyOf(t *testing.T) {
	require.Equal(t, "metrics/ranked_generic/rate", FamilyOf("metrics/ranked_generic/rate/high/span.foo"))
	require.Equal(t, "metrics/ranked_generic/rate/*/*", FamilyGlob("metrics/ranked_generic/rate/high/span.foo"))
	require.Equal(t, "metadata/ranked_dedicated/tagvalues/*", FamilyGlob("metadata/ranked_dedicated/tagvalues/span.foo"))
	require.Equal(t, "metadata/ranked_dedicated/tagvalues", FamilyOf("metadata/ranked_dedicated/tagvalues/span.foo"))
	require.Empty(t, FamilyOf("traceid/present"))
	require.Empty(t, FamilyOf("metadata/tagnames/span"), "a fixed case has no family")
}

func TestFamilies(t *testing.T) {
	run := func(name string, half, double, zero float64, matched int64) Run {
		return Run{Name: name, Result: newResult("mac",
			caseResult("search/ranked_generic/high/half", 1, metrics.Set{"harness.wallNs": measurement(half)}),
			caseResult("search/nopredicate", 1, metrics.Set{"harness.wallNs": measurement(1)}),
			caseResult("search/ranked_generic/high/double", 1, metrics.Set{"harness.wallNs": measurement(double)}),
			caseResult("search/ranked_generic/high/zero", 1, metrics.Set{"harness.wallNs": measurement(zero)}),
			caseResult("search/ranked_generic/high/other", matched, metrics.Set{"harness.wallNs": measurement(1)}),
			caseResult("search/ranked_dedicated/high/one", 1, metrics.Set{"harness.wallNs": measurement(1)}),
		)}
	}
	c, err := New([]Run{run("base", 100, 100, 0, 5), run("next", 50, 200, 3, 6)})
	require.NoError(t, err)

	fs := c.Families(c.Summary("harness.wallNs", P50, 0))
	require.Len(t, fs, 2, "a fixed case has no family")

	f := fs[0]
	require.Equal(t, "search/ranked_generic", f.Family)
	require.Equal(t, "search/ranked_generic/*/*", f.Glob)
	require.Equal(t, []int{0, 2, 3, 4}, f.Cases)
	require.Equal(t, 2, f.Counted, "a zero and a case that cannot be compared are left out")
	require.InDelta(t, 100, f.Cells[0].Value, 1e-9)
	require.False(t, f.Cells[0].HasDelta)
	// Halving one case and doubling another is no change overall, as a
	// geometric mean has it.
	require.InDelta(t, 100, f.Cells[1].Value, 1e-9)
	require.True(t, f.Cells[1].HasDelta)
	require.InDelta(t, 0, f.Cells[1].Delta, 1e-9)
	require.InDelta(t, -50, f.Cells[1].Least, 1e-9)
	require.InDelta(t, 100, f.Cells[1].Most, 1e-9)

	require.Equal(t, "search/ranked_dedicated", fs[1].Family)
	require.Equal(t, 1, fs[1].Counted)
}
