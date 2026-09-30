package render

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare/internal/comparetest"
	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
)

func TestNewTable(t *testing.T) {
	base := comparetest.Summary(15e6, 26e6, 32.3e6, 37e6, 40.3e6, 47.8e6, 1.12e9)
	faster := comparetest.Summary(14e6, 23e6, 28.1e6, 33e6, 36.9e6, 44e6, 0.98e9)
	s := compare.Series{Unit: compare.Nanoseconds, Summaries: []*metrics.Summary{&base, &faster, nil}}
	names := []string{"base", "rb-4M", "gone"}

	v := NewTable(names, s, 0)
	require.Equal(t, []string{"run", "n", "p50", "p90", "p99", "max", "Δp50", "Δp99"}, v.Header)
	require.Equal(t, []TableRow{
		{Name: Text{Text: "base", Style: BaselineName}, Cells: []string{"10", "32.3ms", "40.3ms", "47.8ms", "1.12s", "", ""}},
		{Name: Text{Text: "rb-4M", Style: RunName, Run: 1}, Cells: []string{"10", "28.1ms", "36.9ms", "44ms", "980ms", "-13.0%", "-7.9%"}},
		{Name: Text{Text: "gone", Style: RunName, Run: 2}, Cells: []string{"–", "–", "–", "–", "–", "", ""}},
	}, v.Rows)

	// Against another baseline the changes are from it instead, and a baseline
	// without data has nothing to compare against.
	require.Equal(t, []string{"+14.9%", "+8.6%"}, NewTable(names, s, 1).Rows[0].Cells[5:])
	require.Equal(t, []string{"", ""}, NewTable(names, s, 2).Rows[0].Cells[5:])

	zero := comparetest.Summary(0, 0, 0, 0, 0, 0, 0)
	s = compare.Series{Unit: compare.Count, Summaries: []*metrics.Summary{&zero, &faster}}
	require.Equal(t, []string{notApplicable, notApplicable}, NewTable([]string{"a", "b"}, s, 0).Rows[1].Cells[5:])
}
