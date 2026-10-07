package compare

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark"
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

func TestTotalSummary(t *testing.T) {
	totalMeasurement := func(total float64) metrics.Measurement {
		m := measurement(10)
		m.Total = total
		return m
	}
	cr := func(matched int64, executions int, total float64) *benchmark.CaseResult {
		return &benchmark.CaseResult{
			ID: "search/nopredicate", API: "search", Executions: executions, Matched: matched,
			Metrics: metrics.Set{"harness.wallNs": totalMeasurement(total)},
		}
	}
	run := func(repeat int) Run {
		return Run{Result: &benchmark.Result{Options: benchmark.RunOptions{Repeat: repeat}}}
	}
	// Built by hand rather than through New, so the run order is exactly this.
	c := &Comparison{
		Runs: []Run{run(1), run(1), run(3), run(1)},
		Cases: []Case{{Results: []*benchmark.CaseResult{
			cr(5, 55, 100),
			cr(5, 28, 80),
			// Matched and Executions add up over passes: 5 and 55 a pass, 3 passes.
			cr(15, 165, 300),
			cr(6, 55, 50),
		}}},
	}

	sm := c.TotalSummary("harness.wallNs", 0)
	require.Equal(t, "harness.wallNs", sm.Metric)
	require.Equal(t, Nanoseconds, sm.Unit)
	require.Len(t, sm.Rows, 1)

	cells := sm.Rows[0].Cells
	require.Equal(t, SummaryCell{Value: 100, HasValue: true}, cells[0], "the baseline has a value and no change")
	// The executions differ, which blocks the per-execution comparison but
	// not the totals: the match counts agree, so the runs answered the same
	// question.
	require.Equal(t, SummaryCell{Value: 80, HasValue: true, Delta: -20, HasDelta: true}, cells[1])
	require.Equal(t, SummaryCell{Value: 100, HasValue: true, Delta: 0, HasDelta: true}, cells[2], "totals are compared per pass")
	require.Equal(t, SummaryCell{Incomparable: "matched 5 vs 6"}, cells[3])

	// A run that did not report the metric has neither a value nor a change.
	c.Cases[0].Results[1].Metrics = nil
	sm = c.TotalSummary("harness.wallNs", 0)
	require.Equal(t, SummaryCell{}, sm.Rows[0].Cells[1])

	// A gauge has no total to compare: its Total is the value left behind
	// when the case finished, not a sum, so it is not one either.
	gauge := totalMeasurement(100)
	gauge.Kind = metrics.Gauge
	c.Cases[0].Results[1].Metrics = metrics.Set{"harness.wallNs": gauge}
	sm = c.TotalSummary("harness.wallNs", 0)
	require.Equal(t, SummaryCell{}, sm.Rows[0].Cells[1])
}
