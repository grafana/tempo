package term

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare/internal/comparetest"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare/render"
	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
)

// tableText is a summary table as it reads.
func tableText(st SummaryTable) []string {
	lines := []string{strings.TrimRight("  "+st.Header.String(), " ")}
	for _, r := range st.Rows {
		prefix := "  "
		if r.Case < 0 {
			prefix = " "
		}
		lines = append(lines, strings.TrimRight(prefix+r.Line.String(), " "))
	}
	return lines
}

func TestSummaryTable(t *testing.T) {
	c := comparetest.Comparison(t)
	require.Equal(t, []string{
		"  case                 │  base │    again     │    2MiB     │      4MiB",
		" traceByID",
		"  traceid/present      │ 100ns │ 104ns  +4.0% │ 98ns  -2.0% │   87ns  -13.0%",
		" search",
		"  search/nopredicate ⚠ │  10ns │  10ns     0% │ 10ns     0% │ not comparable",
	}, tableText(NewSummaryTable(render.NewSummary(c, "harness.wallNs", compare.P50, 0))))

	// Against another baseline, the changes are from it, and every other run
	// is the one that cannot be compared on the search.
	st := NewSummaryTable(render.NewSummary(c, "harness.wallNs", compare.P50, 3))
	require.Equal(t, "  traceid/present      │  100ns  +14.9% │  104ns  +19.5% │   98ns  +12.6% │ 87ns", tableText(st)[2])
}

func TestSummaryTableStyles(t *testing.T) {
	c := comparetest.Comparison(t)
	st := NewSummaryTable(render.NewSummary(c, "harness.wallNs", compare.P50, 0))

	var heads []render.Style
	for _, t := range st.Header {
		if t.Style == render.RunName || t.Style == render.BaselineName {
			heads = append(heads, t.Style)
		}
	}
	require.Equal(t, []render.Style{render.BaselineName, render.RunName, render.RunName, render.RunName}, heads, "every run heads its column")

	styles := func(l Line) map[render.Style]int {
		out := map[render.Style]int{}
		for _, t := range l {
			out[t.Style]++
		}
		return out
	}
	present := styles(st.Rows[1].Line)
	require.Equal(t, 1, present[render.Worse], "+4.0%")
	require.Equal(t, 1, present[render.Better], "-2.0%")
	require.Equal(t, 1, present[render.MuchBetter], "-13.0%")
	search := styles(st.Rows[3].Line)
	require.Equal(t, 1, search[render.Warn], "the run that cannot be compared")
	require.Equal(t, 2, search[render.Dim]-len(c.Runs), "no change is not a change to call out, past the separators")
}

func TestSummaryTableWithoutData(t *testing.T) {
	run := func(name string, set metrics.Set) compare.Run {
		return compare.Run{Name: name, Result: comparetest.Result("mac", comparetest.Case("search/nopredicate", 5, set))}
	}
	c, err := compare.New([]compare.Run{run("a", metrics.Set{"harness.wallNs": comparetest.Measurement(10)}), run("b", nil)})
	require.NoError(t, err)

	lines := tableText(NewSummaryTable(render.NewSummary(c, "harness.wallNs", compare.P50, 0)))
	require.Equal(t, "  search/nopredicate │ 10ns │ –", lines[2])
}
