package render

import (
	"fmt"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
)

// NotComparable stands in a run's value and change when the run cannot be
// compared with the baseline.
const NotComparable = "not comparable"

// SummaryView is one stat of one metric for every case, as benchstat lays out
// a comparison: the baseline's value, then each other run's value and change.
// Cases are grouped under the API they query.
type SummaryView struct {
	Title    string
	Baseline int
	// Columns head the runs' columns, in run order.
	Columns []Text
	Rows    []SummaryRow
}

// SummaryRow is a heading naming the API of the cases under it, or a case.
type SummaryRow struct {
	// Group is set on a heading, which has nothing else.
	Group string
	// Case indexes Comparison.Cases, and is -1 on a heading.
	Case int
	ID   string
	// Problems say why runs cannot be compared with the baseline on the case.
	Problems []string
	// Cells hold a cell per run, in run order, the baseline's among them.
	Cells []SummaryCell
}

// SummaryCell is one run's value of the stat for a case, and its change from
// the baseline's.
type SummaryCell struct {
	Value string
	// Change is empty for the baseline itself, and for a run without data or
	// a baseline without it.
	Change Text
	// Incomparable says the run cannot be compared with the baseline on the
	// case, so the cell has neither a value nor a change.
	Incomparable bool
}

// NewSummary lines up one stat of a metric for every case against the
// baseline.
func NewSummary(c *compare.Comparison, metric string, stat compare.Stat, baseline int) SummaryView {
	sm := c.Summary(metric, stat, baseline)
	labels, _ := Labels(c)
	v := SummaryView{
		Title:    fmt.Sprintf("%s · %s per execution · change from %s", metric, stat, labels[baseline]),
		Baseline: baseline,
	}
	for run, label := range labels {
		v.Columns = append(v.Columns, RunText(label, run, baseline))
	}

	group := ""
	for _, r := range sm.Rows {
		cs := c.Cases[r.Case]
		if cs.API != group {
			group = cs.API
			v.Rows = append(v.Rows, SummaryRow{Group: group, Case: -1})
		}
		row := SummaryRow{Case: r.Case, ID: cs.ID, Problems: Problems(c, cs, baseline)}
		base := r.Cells[baseline]
		for run, cell := range r.Cells {
			row.Cells = append(row.Cells, summaryCell(sm.Unit, cell, base, run == baseline))
		}
		v.Rows = append(v.Rows, row)
	}
	return v
}

// Notes are every case's problems, for an output that lists them under the
// table.
func (v SummaryView) Notes() []string {
	var notes []string
	for _, r := range v.Rows {
		for _, p := range r.Problems {
			notes = append(notes, r.ID+" "+p)
		}
	}
	return notes
}

func summaryCell(u compare.Unit, cell, base compare.SummaryCell, isBaseline bool) SummaryCell {
	switch {
	case cell.Incomparable != "":
		return SummaryCell{Incomparable: true}
	case !cell.HasValue:
		return SummaryCell{Value: NoValue}
	}
	out := SummaryCell{Value: Format(u, cell.Value)}
	switch {
	case cell.HasDelta:
		out.Change = Text{Text: FormatChange(cell.Delta), Style: changeStyle(cell.Delta)}
	case !isBaseline && base.HasValue:
		// A baseline of zero leaves no percentage to show.
		out.Change = Text{Text: NotApplicable, Style: Dim}
	}
	return out
}
