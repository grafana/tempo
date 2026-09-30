package render

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
)

// tableText is a summary table as it reads.
func tableText(st SummaryTable, notes []string) []string {
	lines := []string{st.Title, strings.TrimRight("  "+st.Header.String(), " ")}
	for _, r := range st.Rows {
		prefix := "  "
		if r.Case < 0 {
			prefix = " "
		}
		lines = append(lines, strings.TrimRight(prefix+r.Line.String(), " "))
	}
	for _, n := range notes {
		lines = append(lines, "  ⚠ "+n)
	}
	return lines
}

func TestSummaryTable(t *testing.T) {
	c := newSummaryComparison(t)
	_, notes := summaryRows(c, c.Summary("harness.wallNs", compare.P50, 0))
	require.Equal(t, []string{
		"harness.wallNs · p50 per execution · change from base",
		"  case                 │  base │    again     │    2MiB     │      4MiB",
		" traceByID",
		"  traceid/present      │ 100ns │ 104ns  +4.0% │ 98ns  -2.0% │   87ns  -13.0%",
		" search",
		"  search/nopredicate ⚠ │  10ns │  10ns     0% │ 10ns     0% │ not comparable",
		"  ⚠ search/nopredicate vs 4MiB: matched 5 vs 6",
	}, tableText(NewSummaryTable(c, "harness.wallNs", compare.P50, 0), notes))

	// Against another baseline, the changes are from it, and every other run
	// is the one that cannot be compared on the search.
	st := NewSummaryTable(c, "harness.wallNs", compare.P50, 3)
	require.Equal(t, "harness.wallNs · p50 per execution · change from 4MiB", st.Title)
	require.Equal(t, "  traceid/present      │  100ns  +14.9% │  104ns  +19.5% │   98ns  +12.6% │ 87ns", tableText(st, nil)[3])
	_, notes = summaryRows(c, c.Summary("harness.wallNs", compare.P50, 3))
	require.Len(t, notes, 3)
}

func TestSummaryTableSegments(t *testing.T) {
	c := newSummaryComparison(t)
	st := NewSummaryTable(c, "harness.wallNs", compare.P50, 0)

	var runs []int
	for _, s := range st.Header {
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
	require.Equal(t, 3, kinds(st.Rows[1].Line)[ChangeSegment], "every change of MinorChange or more")
	require.Equal(t, 1, kinds(st.Rows[3].Line)[WarnSegment], "the run that cannot be compared")
	require.Zero(t, kinds(st.Rows[3].Line)[ChangeSegment], "no change is not a change to call out")
}

func TestSummaryTableCells(t *testing.T) {
	value := func(v float64) compare.SummaryCell { return compare.SummaryCell{Value: v, HasValue: true} }
	change := func(v, pct float64) compare.SummaryCell {
		return compare.SummaryCell{Value: v, HasValue: true, Delta: pct, HasDelta: true}
	}
	none := compare.SummaryCell{}

	tests := []struct {
		name       string
		cell, base compare.SummaryCell
		isBaseline bool
		want       summaryText
	}{
		{"the baseline", value(10), none, true, summaryText{value: "10ns"}},
		{"a change", change(8.7, -13), value(10), false, summaryText{value: "8.7ns", change: "-13.0%", delta: -13, notable: true}},
		{"a small change", change(9.9, -1), value(10), false, summaryText{value: "9.9ns", change: "-1.0%", delta: -1}},
		{"from a baseline of zero", value(3), value(0), false, summaryText{value: "3ns", change: notApplicable}},
		{"with the baseline missing", value(3), none, false, summaryText{value: "3ns"}},
		{"without data", none, value(10), false, summaryText{value: noValue}},
		{"not comparable", compare.SummaryCell{Incomparable: "matched 5 vs 6"}, value(10), false, summaryText{incomparable: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, summaryCellText(compare.Nanoseconds, tt.cell, tt.base, tt.isBaseline))
		})
	}
}

func TestSummaryTableWithoutData(t *testing.T) {
	r := func(set metrics.Set) *compare.Run {
		return &compare.Run{Result: newResult("mac", caseResult("search/nopredicate", 5, set))}
	}
	a, b := r(metrics.Set{"harness.wallNs": measurement(10)}), r(nil)
	a.Name, b.Name = "a", "b"
	c, err := compare.New([]compare.Run{*a, *b})
	require.NoError(t, err)

	lines := tableText(NewSummaryTable(c, "harness.wallNs", compare.P50, 0), nil)
	require.Equal(t, "  search/nopredicate │ 10ns │ –", lines[3])
}
