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
