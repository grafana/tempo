package compare

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
)

// newSummaryComparison has four runs: the baseline, another run set up like
// it, and two read buffer sizes.
func newSummaryComparison(t *testing.T) *Comparison {
	t.Helper()
	run := func(name string, size int, p50 float64, matched int64) Run {
		r := newResult("mac",
			caseResult("traceid/present", 1, metrics.Set{
				"harness.wallNs": measurement(p50),
				"backend.reads":  measurement(200),
			}),
			caseResult("search/nopredicate", matched, metrics.Set{"harness.wallNs": measurement(10)}),
		)
		r.Cases[0].API = "traceByID"
		r.Options.ReadBufferSize = size
		return Run{Name: name, Result: r}
	}
	c, err := New([]Run{
		run("base", 0, 100, 5),
		run("again", 0, 104, 5),
		run("2MiB", 2<<20, 98, 5),
		run("4MiB", 4<<20, 87, 6),
	})
	require.NoError(t, err)
	return c
}

func TestSummary(t *testing.T) {
	c := newSummaryComparison(t)
	require.Equal(t, []string{"base", "again", "2MiB", "4MiB"}, c.Names())

	sm := c.Summary("harness.wallNs", P50, 0)
	require.Len(t, sm.Rows, 4)
	require.Equal(t, "traceByID", sm.Rows[0].Group)

	present := sm.Rows[1].Cells
	require.Equal(t, SummaryCell{Value: "100ns"}, present[0], "the baseline has a value and no change")
	require.Equal(t, "+4.0%", present[1].Change)
	require.Equal(t, "-2.0%", present[2].Change)
	require.True(t, present[2].Notable, "a change of MinorChange is worth calling out")
	require.Equal(t, "-13.0%", present[3].Change)
	require.True(t, present[3].Notable)
	require.Equal(t, "87ns", present[3].Value)

	// A run that matched differently cannot be compared, and says why.
	search := sm.Rows[3].Cells
	require.True(t, search[3].Incomparable)
	require.Empty(t, search[3].Value)
	require.Equal(t, []string{"search/nopredicate vs 4MiB: matched 5 vs 6"}, sm.Notes)

	// Identical values are no change, and not a signed zero.
	reads := c.Summary("backend.reads", P50, 0).Rows[1].Cells
	require.Equal(t, "0%", reads[3].Change)
	require.False(t, reads[3].Notable)
}

func TestSummaryLines(t *testing.T) {
	c := newSummaryComparison(t)
	require.Equal(t, []string{
		"harness.wallNs · p50 per execution · change from base",
		"  case                 │  base │    again     │    2MiB     │      4MiB",
		" traceByID",
		"  traceid/present      │ 100ns │ 104ns  +4.0% │ 98ns  -2.0% │   87ns  -13.0%",
		" search",
		"  search/nopredicate ⚠ │  10ns │  10ns     0% │ 10ns     0% │ not comparable",
		"  ⚠ search/nopredicate vs 4MiB: matched 5 vs 6",
	}, c.Summary("harness.wallNs", P50, 0).Lines())
}

func TestSummaryLayoutSegments(t *testing.T) {
	c := newSummaryComparison(t)
	header, rows := c.Summary("harness.wallNs", P50, 0).Layout()

	var runs []int
	for _, s := range header {
		if s.Kind == RunSegment {
			runs = append(runs, s.Run)
		}
	}
	require.Equal(t, []int{0, 1, 2, 3}, runs, "every run heads its column")

	kinds := func(l Line) map[SegmentKind]int {
		out := map[SegmentKind]int{}
		for _, s := range l {
			out[s.Kind]++
		}
		return out
	}
	require.Equal(t, 3, kinds(rows[1].Line)[ChangeSegment], "every change of MinorChange or more")
	require.Equal(t, 1, kinds(rows[3].Line)[WarnSegment], "the run that cannot be compared")
}

func TestRunLabels(t *testing.T) {
	c := &Comparison{Runs: []Run{{Name: "default"}, {Name: "4MiB"}}}
	labels, numbered := c.RunLabels()
	require.False(t, numbered)
	require.Equal(t, []string{"default", "4MiB"}, labels)

	c.Runs = append(c.Runs, Run{Name: "readBufferSize=4MiB"})
	labels, numbered = c.RunLabels()
	require.True(t, numbered, "one name too long numbers every run, so they read alike")
	require.Equal(t, []string{"#1", "#2", "#3"}, labels)
}
