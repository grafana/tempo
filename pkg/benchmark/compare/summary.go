package compare

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
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

// SummaryCell is one run's value of a stat for one case, and how it compares
// with the baseline's.
type SummaryCell struct {
	Value string
	// Change is the change from the baseline as a percentage, or empty for the
	// baseline itself or a run without data.
	Change string
	// Delta is the change in percent.
	Delta float64
	// Notable says the change is at least MinorChange.
	Notable bool
	// Incomparable says the run cannot be compared with the baseline, so the
	// cell has no value or change.
	Incomparable bool
}

// SummaryRow is a heading naming the API of the cases under it, or a case.
type SummaryRow struct {
	Group string
	// Case indexes Comparison.Cases on a case row, and is -1 on a heading.
	Case int
	// Label is the case ID, flagged when a run cannot be compared on it.
	Label string
	// Cells holds a cell per run, in run order, the baseline's among them.
	Cells []SummaryCell
}

// Summary is one stat of one metric for every case, as benchstat lays out a
// comparison: the baseline's value, then each other run's value and change.
type Summary struct {
	Metric   string
	Stat     Stat
	Baseline int
	// Labels head the runs' columns, as RunLabels gives them.
	Labels []string
	Rows   []SummaryRow
	// Notes say why runs could not be compared, one per case and run.
	Notes []string
}

// Summary lines up one stat of a metric for every case against the baseline.
func (c *Comparison) Summary(metric string, stat Stat, baseline int) Summary {
	labels, _ := c.RunLabels()
	sm := Summary{Metric: metric, Stat: stat, Baseline: baseline, Labels: labels}

	group := ""
	for i, cs := range c.Cases {
		if cs.API != group {
			group = cs.API
			sm.Rows = append(sm.Rows, SummaryRow{Group: group, Case: -1})
		}
		row := SummaryRow{Case: i, Label: cs.ID, Cells: make([]SummaryCell, len(c.Runs))}
		if problems := c.Problems(cs, baseline); len(problems) > 0 {
			row.Label += " ⚠"
			for _, p := range problems {
				sm.Notes = append(sm.Notes, cs.ID+" "+p)
			}
		}

		s := cs.Series(metric)
		for run := range c.Runs {
			row.Cells[run] = summaryCell(cs, s, stat, run, baseline)
		}
		sm.Rows = append(sm.Rows, row)
	}
	return sm
}

func summaryCell(cs Case, s Series, stat Stat, run, baseline int) SummaryCell {
	if run != baseline && cs.Incomparable(run, baseline) != "" {
		return SummaryCell{Incomparable: true}
	}
	base, sum := s.Summaries[baseline], s.Summaries[run]
	if sum == nil {
		return SummaryCell{Value: noValue}
	}
	cell := SummaryCell{Value: Format(s.Unit, stat.of(sum))}
	if run == baseline || base == nil {
		return cell
	}

	pct, ok := delta(stat.of(base), stat.of(sum))
	if !ok {
		cell.Change = notApplicable
		return cell
	}
	cell.Delta, cell.Change = pct, formatPercent(pct)
	cell.Notable = math.Abs(pct) >= MinorChange
	return cell
}

// Title names what the summary shows.
func (sm Summary) Title() string {
	return fmt.Sprintf("%s · %s per execution · change from %s", sm.Metric, sm.Stat, sm.Labels[sm.Baseline])
}

// Layout lays the summary out in columns: the case, the baseline's value, then
// each other run's value and change.
func (sm Summary) Layout() (header Line, rows []TableLine) {
	labelWidth := utf8.RuneCountInString("case")
	values, changes := make([]int, len(sm.Labels)), make([]int, len(sm.Labels))
	incomparable := make([]bool, len(sm.Labels))
	for _, r := range sm.Rows {
		labelWidth = max(labelWidth, utf8.RuneCountInString(r.Label))
		for run, cell := range r.Cells {
			values[run] = max(values[run], utf8.RuneCountInString(cell.Value))
			changes[run] = max(changes[run], utf8.RuneCountInString(cell.Change))
			incomparable[run] = incomparable[run] || cell.Incomparable
		}
	}

	// A run's column is its value and change side by side, widened for its
	// label, or for saying it cannot be compared, which takes both.
	widths := make([]int, len(sm.Labels))
	for run, label := range sm.Labels {
		if run == sm.Baseline {
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
	for run, label := range sm.Labels {
		text := center(label, widths[run])
		if run == sm.Baseline {
			text = padLeft(label, widths[run])
		}
		header = append(header, Segment{Text: columnSep, Kind: DimSegment}, Segment{Text: text, Kind: RunSegment, Run: run})
	}

	for _, r := range sm.Rows {
		if r.Group != "" {
			rows = append(rows, TableLine{Case: -1, Line: Line{{Text: r.Group, Kind: DimSegment}}})
			continue
		}
		l := Line{{Text: pad(r.Label, labelWidth)}}
		for run, cell := range r.Cells {
			l = append(l, Segment{Text: columnSep, Kind: DimSegment})
			switch {
			case cell.Incomparable:
				l = append(l, Segment{Text: padLeft(notComparable, widths[run]), Kind: WarnSegment})
			case run == sm.Baseline:
				l = append(l, Segment{Text: padLeft(cell.Value, widths[run])})
			default:
				change := Segment{Text: padLeft(cell.Change, changes[run]), Kind: DimSegment}
				if cell.Notable {
					change.Kind, change.Delta = ChangeSegment, cell.Delta
				}
				l = append(l, Segment{Text: padLeft(cell.Value, values[run]) + cellGap}, change)
			}
		}
		rows = append(rows, TableLine{Case: r.Case, Line: l})
	}
	return header, rows
}

// Lines writes the summary as plain text, under its title and over its notes.
func (sm Summary) Lines() []string {
	header, rows := sm.Layout()
	lines := []string{sm.Title(), strings.TrimRight("  "+header.String(), " ")}
	for _, r := range rows {
		prefix := "  "
		if r.Case < 0 {
			prefix = " "
		}
		lines = append(lines, strings.TrimRight(prefix+r.Line.String(), " "))
	}
	for _, n := range sm.Notes {
		lines = append(lines, "  ⚠ "+n)
	}
	return lines
}

// SegmentKind says what a piece of a line shows, so a view can style it.
type SegmentKind int

const (
	PlainSegment SegmentKind = iota
	// DimSegment is scaffolding: headings, separators, and changes too small
	// to call one.
	DimSegment
	// RunSegment names a run, in Segment.Run.
	RunSegment
	// ChangeSegment is a change from the baseline, in Segment.Delta.
	ChangeSegment
	// WarnSegment says a run cannot be compared with the baseline.
	WarnSegment
)

// Segment is a piece of a line.
type Segment struct {
	Text string
	Kind SegmentKind
	// Run indexes Comparison.Runs, on a RunSegment.
	Run int
	// Delta is in percent, on a ChangeSegment.
	Delta float64
}

// Line is a line of segments. Its plain text is theirs run together.
type Line []Segment

func (l Line) String() string {
	var b strings.Builder
	for _, s := range l {
		b.WriteString(s.Text)
	}
	return b.String()
}

// TableLine is one laid-out row of a table.
type TableLine struct {
	// Case indexes Comparison.Cases, or is -1 on a heading.
	Case int
	Line Line
}

// maxRunLabel is the longest run name that heads a column. Past it, runs are
// numbered instead, and the runs are listed with their numbers.
const maxRunLabel = 8

// RunLabels head the runs' columns: their names, or their numbers in order
// when any name is too long to. Labels do not depend on the baseline, so they
// stay put when it moves.
func (c *Comparison) RunLabels() (labels []string, numbered bool) {
	names := c.Names()
	for _, n := range names {
		if utf8.RuneCountInString(n) > maxRunLabel {
			numbered = true
		}
	}
	if !numbered {
		return names, false
	}
	labels = make([]string, len(names))
	for i := range names {
		labels[i] = fmt.Sprintf("#%d", i+1)
	}
	return labels, true
}

// RunLeads lead each run's line of description: its name padded to line the
// descriptions up, after its number when runs are numbered to head columns,
// so the lines are the key to them.
func (c *Comparison) RunLeads() []string {
	names := c.Names()
	labels, numbered := c.RunLabels()
	width := nameWidth(names)
	leads := make([]string, len(names))
	for i, name := range names {
		leads[i] = fitName(name, width)
		if numbered {
			leads[i] = pad(labels[i], longest(labels)+1) + leads[i]
		}
	}
	return leads
}

func center(s string, width int) string {
	gap := max(0, width-utf8.RuneCountInString(s))
	return strings.Repeat(" ", gap/2) + s + strings.Repeat(" ", gap-gap/2)
}

func padLeft(s string, width int) string {
	return strings.Repeat(" ", max(0, width-utf8.RuneCountInString(s))) + s
}
