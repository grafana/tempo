package html

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare/internal/comparetest"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare/render"
	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
)

func TestTableRows(t *testing.T) {
	base := comparetest.Summary(15e6, 26e6, 32.3e6, 37e6, 40.3e6, 47.8e6, 1.12e9)
	faster := comparetest.Summary(14e6, 23e6, 28.1e6, 33e6, 36.9e6, 44e6, 0.98e9)
	s := compare.Series{Unit: compare.Nanoseconds, Summaries: []*metrics.Summary{&base, &faster, nil}}
	names := []string{"base", "rb-4M", "gone"}

	require.Equal(t, []tableRowView{
		{Name: text(render.RunText("base", 0, 0)), Cells: []string{"10", "32.3ms", "40.3ms", "47.8ms", "1.12s", "", ""}},
		{Name: text(render.RunText("rb-4M", 1, 0)), Cells: []string{"10", "28.1ms", "36.9ms", "44ms", "980ms", "-13.0%", "-7.9%"}},
		{Name: text(render.RunText("gone", 2, 0)), Cells: []string{"–", "–", "–", "–", "–", "", ""}},
	}, tableRows(names, s, 0))

	// Against another baseline the changes are from it instead, and a baseline
	// without data has nothing to compare against.
	require.Equal(t, []string{"+14.9%", "+8.6%"}, tableRows(names, s, 1)[0].Cells[5:])
	require.Equal(t, []string{"", ""}, tableRows(names, s, 2)[0].Cells[5:])

	zero := comparetest.Summary(0, 0, 0, 0, 0, 0, 0)
	s = compare.Series{Unit: compare.Count, Summaries: []*metrics.Summary{&zero, &faster}}
	require.Equal(t, []string{render.NotApplicable, render.NotApplicable}, tableRows([]string{"a", "b"}, s, 0)[1].Cells[5:])
}
