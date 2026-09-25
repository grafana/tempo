package compare

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
)

const (
	noValue = "–"
	// notApplicable stands for a change from a baseline of zero, which no
	// percentage can say.
	notApplicable = "n/a"
)

// Table lists each run's summary of a series with its change from the baseline
// run. Rows line up with the runs, so a caller can style each run's row on its
// own.
func Table(names []string, s Series, baseline int) (header string, rows []string) {
	cells := [][]string{{"run", "n", "p50", "p90", "p99", "max", "Δp50", "Δp99"}}
	base := s.Summaries[baseline]
	for i, sum := range s.Summaries {
		name := clip(names[i], maxNameWidth)
		if sum == nil {
			cells = append(cells, []string{name, noValue, noValue, noValue, noValue, noValue, "", ""})
			continue
		}
		row := []string{
			name,
			strconv.Itoa(sum.Count),
			Format(s.Unit, sum.P50),
			Format(s.Unit, sum.P90),
			Format(s.Unit, sum.P99),
			Format(s.Unit, sum.Max),
			"", "",
		}
		if i != baseline && base != nil {
			row[6] = formatDelta(P50, base, sum)
			row[7] = formatDelta(P99, base, sum)
		}
		cells = append(cells, row)
	}

	lines := alignColumns(cells)
	return lines[0], lines[1:]
}

func formatDelta(stat Stat, base, sum *metrics.Summary) string {
	pct, ok := delta(stat.of(base), stat.of(sum))
	if !ok {
		return notApplicable
	}
	return formatPercent(pct)
}

// formatPercent writes a change to a tenth of a percent, and no change as 0%
// rather than a signed zero.
func formatPercent(pct float64) string {
	if pct == 0 {
		return "0%"
	}
	return fmt.Sprintf("%+.1f%%", pct)
}

// alignColumns pads cells into columns two spaces apart, the first aligned left
// and the rest, which are numbers, aligned right.
func alignColumns(cells [][]string) []string {
	widths := make([]int, len(cells[0]))
	for _, row := range cells {
		for j, cell := range row {
			widths[j] = max(widths[j], utf8.RuneCountInString(cell))
		}
	}

	lines := make([]string, 0, len(cells))
	for _, row := range cells {
		var b strings.Builder
		for j, cell := range row {
			gap := strings.Repeat(" ", widths[j]-utf8.RuneCountInString(cell))
			if j == 0 {
				b.WriteString(cell + gap)
				continue
			}
			b.WriteString("  " + gap + cell)
		}
		lines = append(lines, strings.TrimRight(b.String(), " "))
	}
	return lines
}
