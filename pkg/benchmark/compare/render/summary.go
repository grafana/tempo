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

// SummaryRow is a heading naming the API of the cases under it, a case, or a
// family of cases rolled up into one.
type SummaryRow struct {
	// Group is set on a heading, which has nothing else.
	Group string
	// Case indexes Comparison.Cases, and is -1 on a heading or a family.
	Case int
	// ID is the case's, or a family's glob over its cases' IDs.
	ID string
	// Family is set on a family's row, with how many cases it has.
	Family string
	Cases  int
	// Problems say why runs cannot be compared with the baseline on the case,
	// or how many cases a family left out of its means.
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
	// Range is the least to the most change of a family's cases, on a
	// family's row.
	Range Text
}

// NewSummary lines up one stat of a metric for every case against the
// baseline, with each family of generated cases rolled up into one row.
func NewSummary(c *compare.Comparison, metric string, stat compare.Stat, baseline int) SummaryView {
	return newSummary(c, metric, stat, baseline, true, "")
}

// NewCaseSummary is NewSummary with a row for every case, or only for the
// family's cases when a family is given.
func NewCaseSummary(c *compare.Comparison, metric string, stat compare.Stat, baseline int, family string) SummaryView {
	return newSummary(c, metric, stat, baseline, false, family)
}

func newSummary(c *compare.Comparison, metric string, stat compare.Stat, baseline int, rollUp bool, family string) SummaryView {
	sm := c.Summary(metric, stat, baseline)
	labels, _ := Labels(c)
	v := SummaryView{
		Title:    fmt.Sprintf("%s · %s per execution · change from %s", metric, stat, labels[baseline]),
		Baseline: baseline,
	}
	for run, label := range labels {
		v.Columns = append(v.Columns, RunText(label, run, baseline))
	}

	rolled := map[string]compare.FamilySummary{}
	if rollUp {
		for _, f := range c.Families(sm) {
			if len(f.Cases) > 1 {
				rolled[f.Family] = f
			}
		}
	}

	group := ""
	shown := map[string]bool{}
	for _, r := range sm.Rows {
		cs := c.Cases[r.Case]
		fam := compare.FamilyOf(cs.ID)
		f, isRolled := rolled[fam]
		if (family != "" && fam != family) || shown[fam] {
			continue
		}
		if cs.API != group {
			group = cs.API
			v.Rows = append(v.Rows, SummaryRow{Group: group, Case: -1})
		}
		if isRolled {
			shown[fam] = true
			v.Rows = append(v.Rows, familyRow(sm.Unit, f, baseline))
			continue
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

// HasFamilies says whether any of the rows rolls up a family.
func (v SummaryView) HasFamilies() bool {
	for _, r := range v.Rows {
		if r.Family != "" {
			return true
		}
	}
	return false
}

func familyRow(u compare.Unit, f compare.FamilySummary, baseline int) SummaryRow {
	row := SummaryRow{Case: -1, ID: f.Glob, Family: f.Family, Cases: len(f.Cases)}
	if left := len(f.Cases) - f.Counted; left > 0 {
		row.Problems = []string{fmt.Sprintf("leaves %d of %d cases out of its means", left, len(f.Cases))}
	}
	for run, cell := range f.Cells {
		switch {
		case f.Counted == 0:
			row.Cells = append(row.Cells, SummaryCell{Value: NoValue})
		case run == baseline:
			row.Cells = append(row.Cells, SummaryCell{Value: Format(u, cell.Value)})
		default:
			row.Cells = append(row.Cells, SummaryCell{
				Value:  Format(u, cell.Value),
				Change: Text{Text: FormatChange(cell.Delta), Style: changeStyle(cell.Delta)},
				Range:  Text{Text: "[" + FormatChange(cell.Least) + " … " + FormatChange(cell.Most) + "]", Style: Dim},
			})
		}
	}
	return row
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
