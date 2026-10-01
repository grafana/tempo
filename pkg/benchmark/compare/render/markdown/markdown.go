// Package markdown writes a comparison for a pull request or an issue: how the
// runs differ, then a summary table of every case for each metric.
package markdown

import (
	"fmt"
	"io"
	"strings"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare/render"
)

// Write writes the comparison of every case for each metric at one stat.
func Write(w io.Writer, c *compare.Comparison, metrics []string, stat compare.Stat, baseline int) error {
	runs := render.NewRuns(c, baseline)

	var b strings.Builder
	fmt.Fprintf(&b, "### Benchmark comparison\n\n%s.\n\n", escape(strings.ToUpper(runs.Heading[:1])+runs.Heading[1:]))
	b.WriteString("| run | setup |\n|:--|:--|\n")
	for _, r := range runs.Runs {
		name := "**" + escape(r.Name.Text) + "**"
		if runs.Numbered {
			name = r.Label + " " + name
		}
		fmt.Fprintf(&b, "| %s | %s |\n", name, escape(r.Description()))
	}

	for _, metric := range metrics {
		v := render.NewSummary(c, metric, stat, baseline)
		base := escape(v.Columns[baseline].Text)
		fmt.Fprintf(&b, "\n#### %s · %s per execution\n\nChange from %s.\n\n", metric, stat, base)

		b.WriteString("| case | " + base + " |")
		align := "|:--|--:|"
		for run, col := range v.Columns {
			if run != baseline {
				b.WriteString(" " + escape(col.Text) + " | |")
				align += "--:|--:|"
			}
		}
		b.WriteString("\n" + align + "\n")

		for _, r := range v.Rows {
			if r.Group != "" {
				continue
			}
			label := r.ID
			if len(r.Problems) > 0 {
				label += " " + render.Flag
			}
			fmt.Fprintf(&b, "| %s | %s |", escape(label), r.Cells[baseline].Value)
			for run, cell := range r.Cells {
				switch {
				case run == baseline:
				case cell.Incomparable:
					fmt.Fprintf(&b, " %s | |", render.NotComparable)
				default:
					fmt.Fprintf(&b, " %s | %s |", cell.Value, change(cell.Change))
				}
			}
			b.WriteString("\n")
		}
		if notes := v.Notes(); len(notes) > 0 {
			b.WriteString("\n")
			for _, n := range notes {
				fmt.Fprintf(&b, "- %s %s\n", render.Flag, escape(n))
			}
		}
	}

	if _, err := io.WriteString(w, b.String()); err != nil {
		return fmt.Errorf("writing markdown: %w", err)
	}
	return nil
}

// change writes a change, in bold when it stands out.
func change(t render.Text) string {
	if t.Text != "" && t.Style.Bold() {
		return "**" + t.Text + "**"
	}
	return t.Text
}

// escape keeps text from breaking a markdown table.
func escape(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}
