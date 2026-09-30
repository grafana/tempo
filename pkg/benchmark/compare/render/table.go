package render

import (
	"strconv"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
)

// TableView is each run's numbers for a series, and its change from the
// baseline's.
type TableView struct {
	Header []string
	// Rows hold a row per run, in run order.
	Rows []TableRow
}

// TableRow is one run's numbers.
type TableRow struct {
	// Name is the run's name, shown as the run, as its whole row is.
	Name  Text
	Cells []string
}

// NewTable lists every run's summary of a series.
func NewTable(names []string, s compare.Series, baseline int) TableView {
	v := TableView{Header: []string{"run", "n", "p50", "p90", "p99", "max", "Δp50", "Δp99"}}
	base := s.Summaries[baseline]
	for i, sum := range s.Summaries {
		row := TableRow{Name: runText(names[i], i, baseline)}
		if sum == nil {
			row.Cells = []string{noValue, noValue, noValue, noValue, noValue, "", ""}
			v.Rows = append(v.Rows, row)
			continue
		}
		row.Cells = []string{
			strconv.Itoa(sum.Count),
			Format(s.Unit, sum.P50),
			Format(s.Unit, sum.P90),
			Format(s.Unit, sum.P99),
			Format(s.Unit, sum.Max),
			"", "",
		}
		if i != baseline && base != nil {
			row.Cells[5] = formatDelta(compare.P50, base, sum)
			row.Cells[6] = formatDelta(compare.P99, base, sum)
		}
		v.Rows = append(v.Rows, row)
	}
	return v
}

func formatDelta(stat compare.Stat, base, sum *metrics.Summary) string {
	pct, ok := compare.Delta(stat.Of(base), stat.Of(sum))
	if !ok {
		return notApplicable
	}
	return formatChange(pct)
}
