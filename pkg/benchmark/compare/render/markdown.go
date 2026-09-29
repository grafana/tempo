package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
)

// WriteMarkdown writes a comparison for a pull request or an issue: how the
// runs differ, then a summary table of every case for each metric at one stat.
func WriteMarkdown(w io.Writer, c *compare.Comparison, metrics []string, stat compare.Stat, baseline int) error {
	labels, numbered := Labels(c)

	var b strings.Builder
	heading := Heading(c, baseline)
	fmt.Fprintf(&b, "### Benchmark comparison\n\n%s.\n\n", mdEscape(strings.ToUpper(heading[:1])+heading[1:]))
	b.WriteString("| run | setup |\n|:--|:--|\n")
	for i, name := range c.Names() {
		run := "**" + mdEscape(name) + "**"
		if numbered {
			run = labels[i] + " " + run
		}
		fmt.Fprintf(&b, "| %s | %s |\n", run, mdEscape(DetailText(Describe(c, i, baseline))))
	}

	for _, metric := range metrics {
		rows, notes := summaryRows(c, c.Summary(metric, stat, baseline))
		fmt.Fprintf(&b, "\n#### %s · %s per execution\n\nChange from %s.\n\n", metric, stat, mdEscape(labels[baseline]))

		b.WriteString("| case | " + mdEscape(labels[baseline]) + " |")
		align := "|:--|--:|"
		for run, label := range labels {
			if run != baseline {
				b.WriteString(" " + mdEscape(label) + " | |")
				align += "--:|--:|"
			}
		}
		b.WriteString("\n" + align + "\n")

		for _, r := range rows {
			if r.group != "" {
				continue
			}
			fmt.Fprintf(&b, "| %s | %s |", mdEscape(r.label), r.cells[baseline].value)
			for run, cell := range r.cells {
				switch {
				case run == baseline:
				case cell.incomparable:
					fmt.Fprintf(&b, " %s | |", notComparable)
				default:
					fmt.Fprintf(&b, " %s | %s |", cell.value, cell.change)
				}
			}
			b.WriteString("\n")
		}
		if len(notes) > 0 {
			b.WriteString("\n")
			for _, n := range notes {
				fmt.Fprintf(&b, "- ⚠ %s\n", mdEscape(n))
			}
		}
	}

	if _, err := io.WriteString(w, b.String()); err != nil {
		return fmt.Errorf("writing markdown: %w", err)
	}
	return nil
}

// mdEscape keeps text from breaking a markdown table.
func mdEscape(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}
