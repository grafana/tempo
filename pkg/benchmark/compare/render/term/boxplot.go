package term

import (
	"math"
	"strings"
	"unicode/utf8"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare/render"
	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
)

// Legend explains the marks a box plot row is drawn with.
const Legend = "├ min  ▒ p25–p75  ┃ p50  ╎ p90  ┤ p99  • max  ▸ max past the axis"

const (
	markWhisker = '─'
	markTail    = '·'
	markMin     = '├'
	markP90     = '╎'
	markP99     = '┤'
	markBox     = '▒'
	markMedian  = '┃'
	markMax     = '•'
	markClipped = '▸'
)

// minPlotWidth keeps a plot readable in a narrow terminal.
const minPlotWidth = 20

// Plot is a box plot laid out in text: the axis, over a row per run.
type Plot struct {
	Header []Line
	Rows   []Line
}

// BoxPlot draws a box plot in about width columns, each row led by its run's
// name and shown as the run.
func BoxPlot(v render.BoxPlotView, width int) Plot {
	names := make([]string, len(v.Rows))
	tail := 0
	for i, r := range v.Rows {
		names[i] = r.Name.Text
		// A clipped max is written after the plot, so leave room for the
		// longest.
		if r.Clipped {
			tail = max(tail, 1+utf8.RuneCountInString(r.Max))
		}
	}
	nw := nameWidth(names)
	axis := charAxis{Axis: v.Axis, width: max(width-nw-tail, minPlotWidth)}

	indent := strings.Repeat(" ", nw)
	labels, rule := axis.ticks(v.Ticks)
	p := Plot{Header: []Line{
		{{Text: indent + labels, Style: render.Dim}},
		{{Text: indent + rule, Style: render.Dim}},
	}}
	for _, r := range v.Rows {
		text := fitName(r.Name.Text, nw) + "no data"
		if r.Summary != nil {
			row := axis.row(*r.Summary, r.Clipped)
			if r.Clipped {
				row += " " + r.Max
			}
			text = strings.TrimRight(fitName(r.Name.Text, nw)+row, " ")
		}
		p.Rows = append(p.Rows, Line{{Text: text, Style: r.Name.Style, Run: r.Name.Run}})
	}
	return p
}

// charAxis maps an axis onto the columns of a plot.
type charAxis struct {
	render.Axis
	width int
}

// col is the column a value falls in, clamped to the axis.
func (a charAxis) col(v float64) int {
	if a.Max <= a.Min {
		return 0
	}
	c := int(math.Round((v - a.Min) / (a.Max - a.Min) * float64(a.width-1)))
	return min(max(c, 0), a.width-1)
}

// ticks draws the axis: a line of tick labels above a ruled line with a tick at
// each one.
func (a charAxis) ticks(ticks []render.Tick) (labels, line string) {
	rule := []rune(strings.Repeat(string(markWhisker), a.width))
	text := []rune(strings.Repeat(" ", a.width))

	for _, t := range ticks {
		rule[a.col(t.Value)] = '┼'
	}
	rule[0], rule[a.width-1] = '├', '┤'

	// A label keeps a column clear either side of it, so two never touch.
	taken := make([]bool, a.width)
	place := func(start int, label []rune) {
		if start < 0 || start+len(label) > a.width {
			return
		}
		for c := max(0, start-1); c < min(a.width, start+len(label)+1); c++ {
			if taken[c] {
				return
			}
		}
		copy(text[start:], label)
		for c := start; c < start+len(label); c++ {
			taken[c] = true
		}
	}

	// The ends go first, since they say what the axis spans. The first starts at
	// its tick and the last ends at it, so neither runs off the axis; the rest
	// are centred on theirs where there is room.
	n := len(ticks)
	if n > 0 {
		place(0, []rune(ticks[0].Label))
	}
	if n > 1 {
		last := []rune(ticks[n-1].Label)
		place(a.width-len(last), last)
	}
	for i := 1; i < n-1; i++ {
		l := []rune(ticks[i].Label)
		place(a.col(ticks[i].Value)-len(l)/2, l)
	}
	return strings.TrimRight(string(text), " "), string(rule)
}

// row draws a summary as a box and whiskers across the axis. A max past the
// axis is marked at its edge.
func (a charAxis) row(s metrics.Summary, clipped bool) string {
	cells := []rune(strings.Repeat(" ", a.width))
	fill := func(from, to int, r rune) {
		for i := from; i <= to; i++ {
			cells[i] = r
		}
	}

	p99 := a.col(s.P99)
	fill(a.col(s.Min), p99, markWhisker)
	if top := a.col(s.Max); !clipped && top > p99 {
		fill(p99+1, top, markTail)
		cells[top] = markMax
	}
	if clipped {
		fill(p99+1, a.width-1, markTail)
	}

	// Later marks win a shared column, so the most telling ones go last.
	cells[a.col(s.P90)] = markP90
	cells[a.col(s.Min)] = markMin
	cells[p99] = markP99
	if clipped {
		cells[a.width-1] = markClipped
	}
	fill(a.col(s.P25), a.col(s.P75), markBox)
	cells[a.col(s.P50)] = markMedian
	return string(cells)
}
