package html

import (
	"strconv"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare/render"
	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
)

// tableHeader heads a case's table of each run's numbers.
var tableHeader = []string{"run", "n", "p50", "p90", "p99", "max", "Δp50", "Δp99"}

// tableRowView is one run's numbers for a series, and their change from the
// baseline's.
type tableRowView struct {
	Name  textView
	Cells []string
}

// tableRows lists every run's summary of a series, in run order.
func tableRows(names []string, s compare.Series, baseline int) []tableRowView {
	rows := make([]tableRowView, len(s.Summaries))
	base := s.Summaries[baseline]
	for i, sum := range s.Summaries {
		rows[i].Name = text(render.RunText(names[i], i, baseline))
		if sum == nil {
			rows[i].Cells = []string{render.NoValue, render.NoValue, render.NoValue, render.NoValue, render.NoValue, "", ""}
			continue
		}
		rows[i].Cells = []string{
			strconv.Itoa(sum.Count),
			render.Format(s.Unit, sum.P50),
			render.Format(s.Unit, sum.P90),
			render.Format(s.Unit, sum.P99),
			render.Format(s.Unit, sum.Max),
			"", "",
		}
		if i != baseline && base != nil {
			rows[i].Cells[5] = formatDelta(compare.P50, base, sum)
			rows[i].Cells[6] = formatDelta(compare.P99, base, sum)
		}
	}
	return rows
}

func formatDelta(stat compare.Stat, base, sum *metrics.Summary) string {
	pct, ok := compare.Delta(stat.Of(base), stat.Of(sum))
	if !ok {
		return render.NotApplicable
	}
	return render.FormatChange(pct)
}
