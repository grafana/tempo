// Package markdown writes a comparison for a pull request or an issue: how the
// runs differ, then a summary table of every case for each metric.
package markdown

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare/render"
)

// Write writes the comparison of every case for each metric at one stat.
func Write(w io.Writer, c *compare.Comparison, metrics []string, stat compare.Stat, baseline int) error {
	runs := render.NewRuns(c, baseline)

	var b strings.Builder
	fmt.Fprintf(&b, "### Benchmark comparison\n\n%s.\n\n", escape(strings.ToUpper(runs.Heading[:1])+runs.Heading[1:]))
	rows := [][]string{{"run", "setup"}}
	for _, r := range runs.Runs {
		name := "**" + escape(r.Name.Text) + "**"
		if runs.Numbered {
			name = r.Label + " " + name
		}
		rows = append(rows, []string{name, escape(r.Description())})
	}
	table(&b, rows, false)

	writeSummary := func(v render.SummaryView, heading string) {
		base := escape(v.Columns[baseline].Text)
		fmt.Fprintf(&b, "\n#### %s\n\nChange from %s.\n\n", heading, base)

		header := []string{"case", base}
		for run, col := range v.Columns {
			if run != baseline {
				header = append(header, escape(col.Text), "")
			}
		}
		rows := [][]string{header}
		for _, r := range v.Rows {
			if r.Group != "" {
				continue
			}
			label := r.ID
			if len(r.Problems) > 0 {
				label += " " + render.Flag
			}
			row := []string{escape(label), r.Cells[baseline].Value}
			for run, cell := range r.Cells {
				switch {
				case run == baseline:
				case cell.Incomparable:
					row = append(row, render.NotComparable, "")
				default:
					row = append(row, cell.Value, change(cell.Change))
				}
			}
			rows = append(rows, row)
		}
		table(&b, rows, true)

		if notes := v.Notes(); len(notes) > 0 {
			b.WriteString("\n")
			for _, n := range notes {
				fmt.Fprintf(&b, "- %s %s\n", render.Flag, escape(n))
			}
		}
	}

	if note := render.ShardingNote(c); note != "" {
		fmt.Fprintf(&b, "\n%s\n\n", escape(note))
	}

	// Case totals are only shown when the runs' executions measure different
	// amounts of work, as when their blocks shard differently: then they are
	// the comparison that means something, and per-execution numbers are
	// withheld for the cases they cannot compare.
	totals := c.ShardingDiffers()

	for _, metric := range metrics {
		v := render.NewSummary(c, metric, stat, baseline)
		if totals {
			v = render.ComparableOnly(v, baseline)
		}
		if len(v.Rows) > 0 {
			writeSummary(v, fmt.Sprintf("%s · %s per execution", metric, stat))
		}
		if totals {
			writeSummary(render.NewTotalSummary(c, metric, baseline), fmt.Sprintf("%s · case total per pass", metric))
		}
	}

	if _, err := io.WriteString(w, b.String()); err != nil {
		return fmt.Errorf("writing markdown: %w", err)
	}
	return nil
}

// table writes rows as a markdown table, the first its header, with each
// column padded to line up so the table reads in a terminal as it does
// rendered. The first column is aligned left, and so are the rest unless they
// are numbers.
func table(b *strings.Builder, rows [][]string, numbers bool) {
	widths := make([]int, len(rows[0]))
	for _, row := range rows {
		for i, cell := range row {
			// An alignment marker needs three characters.
			widths[i] = max(widths[i], 3, utf8.RuneCountInString(cell))
		}
	}
	right := func(i int) bool { return numbers && i > 0 }

	line := func(cells []string) {
		for i, cell := range cells {
			gap := strings.Repeat(" ", widths[i]-utf8.RuneCountInString(cell))
			if right(i) {
				cell = gap + cell
			} else {
				cell += gap
			}
			b.WriteString("| " + cell + " ")
		}
		b.WriteString("|\n")
	}

	line(rows[0])
	align := make([]string, len(widths))
	for i, w := range widths {
		align[i] = ":" + strings.Repeat("-", w-1)
		if right(i) {
			align[i] = strings.Repeat("-", w-1) + ":"
		}
	}
	line(align)
	for _, row := range rows[1:] {
		line(row)
	}
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
