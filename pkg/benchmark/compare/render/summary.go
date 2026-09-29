package render

import (
	"fmt"
	"math"
	"unicode/utf8"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
)

const (
	// notComparable stands in a run's column, value and change both, when the
	// run cannot be compared with the baseline.
	notComparable = "not comparable"
	// columnSep parts a table's columns: the case from the baseline, and each
	// run from the next.
	columnSep = " │ "
	// cellGap parts a value from its change.
	cellGap = "  "
)

// SummaryTable is a summary laid out as benchstat lays out a comparison: the
// case, the baseline's value, then each other run's value and change. Cases
// are grouped under the API they query.
type SummaryTable struct {
	Title  string
	Header Line
	Rows   []TableLine
	// Notes say why runs could not be compared, one per case and run.
	Notes []string
}

// NewSummaryTable lays out one stat of a metric for every case against the
// baseline.
func NewSummaryTable(c *compare.Comparison, metric string, stat compare.Stat, baseline int) SummaryTable {
	sm := c.Summary(metric, stat, baseline)
	labels, _ := Labels(c)
	rows, notes := summaryRows(c, sm)
	header, lines := layoutSummary(rows, labels, baseline)
	return SummaryTable{
		Title:  fmt.Sprintf("%s · %s per execution · change from %s", metric, stat, labels[baseline]),
		Header: header,
		Rows:   lines,
		Notes:  notes,
	}
}

// summaryRow is a summary's row written out: a heading, or a case's cells.
type summaryRow struct {
	// group is set on a heading.
	group string
	// caseIndex indexes Comparison.Cases, or is -1 on a heading.
	caseIndex int
	// label is the case ID, flagged when a run cannot be compared on it.
	label string
	cells []summaryText
}

// summaryText is one run's value and change for a case, written out.
type summaryText struct {
	value, change string
	delta         float64
	// notable says the change is at least MinorChange.
	notable      bool
	incomparable bool
}

// summaryRows writes a summary's cells out, with a heading for each API, and
// notes on why any run cannot be compared.
func summaryRows(c *compare.Comparison, sm compare.Summary) (rows []summaryRow, notes []string) {
	group := ""
	for _, r := range sm.Rows {
		cs := c.Cases[r.Case]
		if cs.API != group {
			group = cs.API
			rows = append(rows, summaryRow{group: group, caseIndex: -1})
		}
		row := summaryRow{caseIndex: r.Case, label: cs.ID}
		if problems := Problems(c, cs, sm.Baseline); len(problems) > 0 {
			row.label += " ⚠"
			for _, p := range problems {
				notes = append(notes, cs.ID+" "+p)
			}
		}
		base := r.Cells[sm.Baseline]
		for run, cell := range r.Cells {
			row.cells = append(row.cells, summaryCellText(sm.Unit, cell, base, run == sm.Baseline))
		}
		rows = append(rows, row)
	}
	return rows, notes
}

func summaryCellText(u compare.Unit, cell, base compare.SummaryCell, isBaseline bool) summaryText {
	switch {
	case cell.Incomparable != "":
		return summaryText{incomparable: true}
	case !cell.HasValue:
		return summaryText{value: noValue}
	}
	t := summaryText{value: Format(u, cell.Value)}
	switch {
	case cell.HasDelta:
		t.change, t.delta = formatChange(cell.Delta), cell.Delta
		t.notable = math.Abs(cell.Delta) >= MinorChange
	case !isBaseline && base.HasValue:
		// A baseline of zero leaves no percentage to show.
		t.change = notApplicable
	}
	return t
}

// layoutSummary lays written-out rows in columns under a header of run labels.
func layoutSummary(rows []summaryRow, labels []string, baseline int) (header Line, lines []TableLine) {
	labelWidth := utf8.RuneCountInString("case")
	values, changes := make([]int, len(labels)), make([]int, len(labels))
	incomparable := make([]bool, len(labels))
	for _, r := range rows {
		labelWidth = max(labelWidth, utf8.RuneCountInString(r.label))
		for run, cell := range r.cells {
			values[run] = max(values[run], utf8.RuneCountInString(cell.value))
			changes[run] = max(changes[run], utf8.RuneCountInString(cell.change))
			incomparable[run] = incomparable[run] || cell.incomparable
		}
	}

	// A run's column is its value and change side by side, widened for its
	// label, or for saying it cannot be compared, which takes both.
	widths := make([]int, len(labels))
	for run, label := range labels {
		if run == baseline {
			widths[run] = max(values[run], utf8.RuneCountInString(label))
			continue
		}
		widths[run] = values[run] + len(cellGap) + changes[run]
		want := utf8.RuneCountInString(label)
		if incomparable[run] {
			want = max(want, len(notComparable))
		}
		if want > widths[run] {
			values[run] += want - widths[run]
			widths[run] = want
		}
	}

	header = Line{{Text: pad("case", labelWidth)}}
	for run, label := range labels {
		text := center(label, widths[run])
		if run == baseline {
			text = padLeft(label, widths[run])
		}
		header = append(header, Segment{Text: columnSep, Kind: DimSegment}, Segment{Text: text, Kind: RunSegment, Run: run})
	}

	for _, r := range rows {
		if r.group != "" {
			lines = append(lines, TableLine{Case: -1, Line: Line{{Text: r.group, Kind: DimSegment}}})
			continue
		}
		l := Line{{Text: pad(r.label, labelWidth)}}
		for run, cell := range r.cells {
			l = append(l, Segment{Text: columnSep, Kind: DimSegment})
			switch {
			case cell.incomparable:
				l = append(l, Segment{Text: padLeft(notComparable, widths[run]), Kind: WarnSegment})
			case run == baseline:
				l = append(l, Segment{Text: padLeft(cell.value, widths[run])})
			default:
				change := Segment{Text: padLeft(cell.change, changes[run]), Kind: DimSegment}
				if cell.notable {
					change.Kind, change.Delta = ChangeSegment, cell.delta
				}
				l = append(l, Segment{Text: padLeft(cell.value, values[run]) + cellGap}, change)
			}
		}
		lines = append(lines, TableLine{Case: r.caseIndex, Line: l})
	}
	return header, lines
}
