package compare

import (
	"fmt"
	"io"
	"slices"
	"strings"
)

// Title names what a series shows. Summaries are per execution: one trace
// lookup, or one shard of a search.
func Title(cs Case, metric string) string {
	return cs.ID + " · " + metric + " · per execution"
}

// RunsHeading says how the runs were named and ordered, and which one the
// others are compared against.
func (c *Comparison) RunsHeading(baseline int) string {
	s := "runs"
	if len(c.NamedBy) > 0 {
		s += " named by " + strings.Join(c.NamedBy, ", ")
	}
	if c.OrderedBy != "" {
		if slices.Equal(c.NamedBy, []string{c.OrderedBy}) {
			s += ", in its order"
		} else {
			s += " in order of " + c.OrderedBy
		}
	}
	return s + "; baseline " + c.Runs[baseline].Name
}

// WriteReport prints how the runs differ, a summary of every case for each
// metric at one stat, then every case for each metric as a box plot over a
// table, in about width columns. It is what the interactive view shows one
// screen at a time.
func WriteReport(w io.Writer, c *Comparison, metrics []string, stat Stat, baseline, width int) error {
	names := c.Names()

	var b strings.Builder
	b.WriteString(c.RunsHeading(baseline) + "\n")
	for i, lead := range c.RunLeads() {
		b.WriteString("  " + lead + DetailText(c.Describe(i, baseline)) + "\n")
	}
	b.WriteString(Legend + "\n")

	for _, metric := range metrics {
		b.WriteString("\n")
		for _, line := range c.Summary(metric, stat, baseline).Lines() {
			b.WriteString(line + "\n")
		}
	}

	for _, cs := range c.Cases {
		problems := c.Problems(cs, baseline)
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

// WriteMarkdown writes the comparison for a pull request or an issue: how the
// runs differ, then a summary table of every case for each metric at one stat.
func WriteMarkdown(w io.Writer, c *Comparison, metrics []string, stat Stat, baseline int) error {
	labels, numbered := c.RunLabels()

	var b strings.Builder
	heading := c.RunsHeading(baseline)
	fmt.Fprintf(&b, "### Benchmark comparison\n\n%s.\n\n", mdEscape(strings.ToUpper(heading[:1])+heading[1:]))
	b.WriteString("| run | setup |\n|:--|:--|\n")
	for i, name := range c.Names() {
		run := "**" + mdEscape(name) + "**"
		if numbered {
			run = labels[i] + " " + run
		}
		fmt.Fprintf(&b, "| %s | %s |\n", run, mdEscape(DetailText(c.Describe(i, baseline))))
	}

	for _, metric := range metrics {
		sm := c.Summary(metric, stat, baseline)
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

		for _, r := range sm.Rows {
			if r.Group != "" {
				continue
			}
			fmt.Fprintf(&b, "| %s | %s |", mdEscape(r.Label), r.Cells[baseline].Value)
			for run, cell := range r.Cells {
				switch {
				case run == baseline:
				case cell.Incomparable:
					fmt.Fprintf(&b, " %s | |", notComparable)
				default:
					fmt.Fprintf(&b, " %s | %s |", cell.Value, cell.Change)
				}
			}
			b.WriteString("\n")
		}
		if len(sm.Notes) > 0 {
			b.WriteString("\n")
			for _, n := range sm.Notes {
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
