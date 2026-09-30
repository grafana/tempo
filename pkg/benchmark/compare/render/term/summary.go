package term

import (
	"unicode/utf8"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare/render"
)

const (
	// columnSep parts a table's columns: the case from the baseline, and each
	// run from the next.
	columnSep = " │ "
	// cellGap parts a value from its change.
	cellGap = "  "
)

// SummaryTable is a summary laid out in columns, under a header of run labels.
type SummaryTable struct {
	Header Line
	Rows   []Row
}

// Row is one laid-out row of a table.
type Row struct {
	// Case indexes Comparison.Cases, or is -1 on a heading.
	Case int
	Line Line
}

// NewSummaryTable lays a summary out: the case, flagged when a run cannot be
// compared on it, then the baseline's value, then each other run's value and
// change.
func NewSummaryTable(v render.SummaryView) SummaryTable {
	label := func(r render.SummaryRow) string {
		if len(r.Problems) > 0 {
			return r.ID + " " + render.Flag
		}
		return r.ID
	}

	runs := len(v.Columns)
	labelWidth := utf8.RuneCountInString("case")
	values, changes := make([]int, runs), make([]int, runs)
	incomparable := make([]bool, runs)
	for _, r := range v.Rows {
		if r.Group != "" {
			continue
		}
		labelWidth = max(labelWidth, utf8.RuneCountInString(label(r)))
		for run, cell := range r.Cells {
			values[run] = max(values[run], utf8.RuneCountInString(cell.Value))
			changes[run] = max(changes[run], utf8.RuneCountInString(cell.Change.Text))
			incomparable[run] = incomparable[run] || cell.Incomparable
		}
	}

	// A run's column is its value and change side by side, widened for its
	// label, or for saying it cannot be compared, which takes both.
	widths := make([]int, runs)
	for run, col := range v.Columns {
		if run == v.Baseline {
			widths[run] = max(values[run], utf8.RuneCountInString(col.Text))
			continue
		}
		widths[run] = values[run] + len(cellGap) + changes[run]
		want := utf8.RuneCountInString(col.Text)
		if incomparable[run] {
			want = max(want, len(render.NotComparable))
		}
		if want > widths[run] {
			values[run] += want - widths[run]
			widths[run] = want
		}
	}

	t := SummaryTable{Header: Line{{Text: pad("case", labelWidth)}}}
	for run, col := range v.Columns {
		text := center(col.Text, widths[run])
		if run == v.Baseline {
			text = padLeft(col.Text, widths[run])
		}
		t.Header = append(t.Header, render.Text{Text: columnSep, Style: render.Dim}, render.Text{Text: text, Style: col.Style, Run: col.Run})
	}

	for _, r := range v.Rows {
		if r.Group != "" {
			t.Rows = append(t.Rows, Row{Case: -1, Line: Line{{Text: r.Group, Style: render.Dim}}})
			continue
		}
		l := Line{{Text: pad(label(r), labelWidth)}}
		for run, cell := range r.Cells {
			l = append(l, render.Text{Text: columnSep, Style: render.Dim})
			switch {
			case cell.Incomparable:
				l = append(l, render.Text{Text: padLeft(render.NotComparable, widths[run]), Style: render.Warn})
			case run == v.Baseline:
				l = append(l, render.Text{Text: padLeft(cell.Value, widths[run])})
			default:
				l = append(l,
					render.Text{Text: padLeft(cell.Value, values[run]) + cellGap},
					render.Text{Text: padLeft(cell.Change.Text, changes[run]), Style: cell.Change.Style},
				)
			}
		}
		t.Rows = append(t.Rows, Row{Case: r.Case, Line: l})
	}
	return t
}
