package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
)

// WriteReport prints how the runs differ, a summary of every case for each
// metric at one stat, then every case for each metric as a box plot over a
// table, in about width columns. It is what the interactive view shows one
// screen at a time.
func WriteReport(w io.Writer, c *compare.Comparison, metrics []string, stat compare.Stat, baseline, width int) error {
	names := c.Names()

	var b strings.Builder
	b.WriteString(Heading(c, baseline) + "\n")
	for i, lead := range Leads(c) {
		b.WriteString("  " + lead + DetailText(Describe(c, i, baseline)) + "\n")
	}
	b.WriteString(Legend + "\n")

	for _, metric := range metrics {
		st := NewSummaryTable(c, metric, stat, baseline)
		b.WriteString("\n" + st.Title + "\n")
		b.WriteString(strings.TrimRight("  "+st.Header.String(), " ") + "\n")
		for _, r := range st.Rows {
			prefix := "  "
			if r.Case < 0 {
				prefix = " "
			}
			b.WriteString(strings.TrimRight(prefix+r.Line.String(), " ") + "\n")
		}
		for _, n := range st.Notes {
			b.WriteString("  ⚠ " + n + "\n")
		}
	}

	for _, cs := range c.Cases {
		problems := Problems(c, cs, baseline)
		for _, metric := range metrics {
			s := cs.Series(metric)
			b.WriteString("\n" + Title(cs, metric) + "\n")
			if cs.Query != "" {
				b.WriteString("query: " + cs.Query + "\n")
			}

			plot := BoxPlot(names, s, width)
			for _, line := range append(plot.Header, plot.Rows...) {
				b.WriteString(line + "\n")
			}
			b.WriteString("\n")

			header, rows := Table(names, s, baseline)
			for _, line := range append([]string{header}, rows...) {
				b.WriteString(line + "\n")
			}
			for _, p := range problems {
				b.WriteString("⚠ " + p + "\n")
			}
		}
	}

	if _, err := io.WriteString(w, b.String()); err != nil {
		return fmt.Errorf("writing report: %w", err)
	}
	return nil
}
