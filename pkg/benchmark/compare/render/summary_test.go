package render

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare/internal/comparetest"
)

func TestNewSummary(t *testing.T) {
	c := comparetest.Comparison(t)
	v := NewSummary(c, "harness.wallNs", compare.P50, 0)

	require.Equal(t, "harness.wallNs · p50 per execution · change from base", v.Title)
	require.Equal(t, 0, v.Baseline)
	require.Equal(t, []Text{
		{Text: "base", Style: BaselineName, Run: 0},
		{Text: "again", Style: RunName, Run: 1},
		{Text: "2MiB", Style: RunName, Run: 2},
		{Text: "4MiB", Style: RunName, Run: 3},
	}, v.Columns)

	require.Len(t, v.Rows, 4)
	require.Equal(t, SummaryRow{Group: "traceByID", Case: -1}, v.Rows[0])
	require.Equal(t, SummaryRow{Group: "search", Case: -1}, v.Rows[2])

	// Each run's value, and its change styled by which way it went and how far.
	present := v.Rows[1]
	require.Equal(t, 0, present.Case)
	require.Equal(t, "traceid/present", present.ID)
	require.Empty(t, present.Problems)
	require.Equal(t, []SummaryCell{
		{Value: "100ns"},
		{Value: "104ns", Change: Text{Text: "+4.0%", Style: Worse}},
		{Value: "98ns", Change: Text{Text: "-2.0%", Style: Better}},
		{Value: "87ns", Change: Text{Text: "-13.0%", Style: MuchBetter}},
	}, present.Cells)

	// A run that matched differently cannot be compared, and the case says why.
	search := v.Rows[3]
	require.Equal(t, []string{"vs 4MiB: matched 5 vs 6"}, search.Problems)
	require.Equal(t, SummaryCell{Value: "10ns", Change: Text{Text: "0%", Style: Dim}}, search.Cells[1], "no change recedes")
	require.Equal(t, SummaryCell{Incomparable: true}, search.Cells[3])
	require.Equal(t, []string{"search/nopredicate vs 4MiB: matched 5 vs 6"}, v.Notes())
}

func TestSummaryCell(t *testing.T) {
	value := func(v float64) compare.SummaryCell { return compare.SummaryCell{Value: v, HasValue: true} }
	change := func(v, pct float64) compare.SummaryCell {
		return compare.SummaryCell{Value: v, HasValue: true, Delta: pct, HasDelta: true}
	}
	none := compare.SummaryCell{}

	tests := []struct {
		name       string
		cell, base compare.SummaryCell
		isBaseline bool
		want       SummaryCell
	}{
		{"the baseline", value(10), none, true, SummaryCell{Value: "10ns"}},
		{"a large fall", change(8.7, -13), value(10), false, SummaryCell{Value: "8.7ns", Change: Text{Text: "-13.0%", Style: MuchBetter}}},
		{"a large rise", change(12, 20), value(10), false, SummaryCell{Value: "12ns", Change: Text{Text: "+20.0%", Style: MuchWorse}}},
		{"a small change", change(9.9, -1), value(10), false, SummaryCell{Value: "9.9ns", Change: Text{Text: "-1.0%", Style: Dim}}},
		{"from a baseline of zero", value(3), value(0), false, SummaryCell{Value: "3ns", Change: Text{Text: notApplicable, Style: Dim}}},
		{"with the baseline missing", value(3), none, false, SummaryCell{Value: "3ns"}},
		{"without data", none, value(10), false, SummaryCell{Value: noValue}},
		{"not comparable", compare.SummaryCell{Incomparable: "matched 5 vs 6"}, value(10), false, SummaryCell{Incomparable: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, summaryCell(compare.Nanoseconds, tt.cell, tt.base, tt.isBaseline))
		})
	}
}
