package compare

import (
	"math"
	"strings"
	"unicode/utf8"

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

const (
	// targetTicks is about how many intervals an axis is divided into.
	targetTicks = 5
	// maxReach is how far past the highest p99 the axis stretches to take in the
	// highest max. Past it the max is clipped, since one slow execution would
	// otherwise squash every box into a few columns.
	maxReach = 1.25
	// minPlotWidth keeps a plot readable in a narrow terminal.
	minPlotWidth = 20
	// maxNameWidth cuts long run names, like several settings run together,
	// before they take the plot's room.
	maxNameWidth = 28
)

// Axis maps values onto the columns of a plot.
type Axis struct {
	Min, Max float64
	// Step is the distance between ticks.
	Step  float64
	Width int
}

// NewAxis fits an axis in width columns to every run of the series. It spans
// the lowest min to the highest p99, rounded out to whole ticks, and reaches
// the highest max only when that is close.
func NewAxis(s Series, width int) Axis {
	lo, hi, top := math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, sum := range s.Summaries {
		if sum == nil {
			continue
		}
		lo = min(lo, sum.Min)
		hi = max(hi, sum.P99)
		top = max(top, sum.Max)
	}
	if math.IsInf(lo, 1) {
		lo, hi, top = 0, 1, 1
	}
	if top <= hi*maxReach {
		hi = top
	}
	if hi <= lo {
		// Every value is the same. Pad around it so it lands mid-axis.
		pad := math.Abs(lo) * 0.1
		if pad == 0 {
			pad = 1
		}
		if lo >= 0 {
			lo = max(0, lo-pad)
		} else {
			lo -= pad
		}
		hi += pad
	}

	step := niceStep((hi - lo) / targetTicks)
	return Axis{
		Min:   math.Floor(lo/step) * step,
		Max:   math.Ceil(hi/step) * step,
		Step:  step,
		Width: max(width, minPlotWidth),
	}
}

// niceStep rounds a tick distance to 1, 2 or 5 times a power of ten, so tick
// labels read as round numbers.
func niceStep(raw float64) float64 {
	if raw <= 0 || math.IsNaN(raw) || math.IsInf(raw, 0) {
		return 1
	}
	mag := math.Pow(10, math.Floor(math.Log10(raw)))
	switch n := raw / mag; {
	case n < 1.5:
		return mag
	case n < 3:
		return 2 * mag
	case n < 7:
		return 5 * mag
	default:
		return 10 * mag
	}
}

// col is the column a value falls in, clamped to the axis.
func (a Axis) col(v float64) int {
	if a.Max <= a.Min {
		return 0
	}
	c := int(math.Round((v - a.Min) / (a.Max - a.Min) * float64(a.Width-1)))
	return min(max(c, 0), a.Width-1)
}

// Ticks draws the axis: a line of tick labels above a ruled line with a tick at
// each step.
func (a Axis) Ticks(u Unit) (labels, line string) {
	rule := []rune(strings.Repeat(string(markWhisker), a.Width))
	text := []rune(strings.Repeat(" ", a.Width))

	n := int(math.Round((a.Max-a.Min)/a.Step)) + 1
	for i := range n {
		rule[a.col(a.Min+float64(i)*a.Step)] = '┼'
	}
	rule[0], rule[a.Width-1] = '├', '┤'

	// A label keeps a column clear either side of it, so two never touch.
	taken := make([]bool, a.Width)
	place := func(start int, label []rune) {
		if start < 0 || start+len(label) > a.Width {
			return
		}
		for c := max(0, start-1); c < min(a.Width, start+len(label)+1); c++ {
			if taken[c] {
				return
			}
		}
		copy(text[start:], label)
		for c := start; c < start+len(label); c++ {
			taken[c] = true
		}
	}
	label := func(i int) []rune { return []rune(Format(u, a.Min+float64(i)*a.Step)) }

	// The ends go first, since they say what the axis spans. The first starts at
	// its tick and the last ends at it, so neither runs off the axis; the rest
	// are centred on theirs where there is room.
	place(0, label(0))
	if n > 1 {
		last := label(n - 1)
		place(a.Width-len(last), last)
	}
	for i := 1; i < n-1; i++ {
		l := label(i)
		place(a.col(a.Min+float64(i)*a.Step)-len(l)/2, l)
	}
	return strings.TrimRight(string(text), " "), string(rule)
}

// Row draws a summary as a box and whiskers across the axis. A max past the axis
// is marked at its edge and reported, so the caller can say where it is.
func (a Axis) Row(s metrics.Summary) (row string, clipped bool) {
	cells := []rune(strings.Repeat(" ", a.Width))
	fill := func(from, to int, r rune) {
		for i := from; i <= to; i++ {
			cells[i] = r
		}
	}

	p99 := a.col(s.P99)
	fill(a.col(s.Min), p99, markWhisker)
	clipped = s.Max > a.Max
	if top := a.col(s.Max); !clipped && top > p99 {
		fill(p99+1, top, markTail)
		cells[top] = markMax
	}
	if clipped {
		fill(p99+1, a.Width-1, markTail)
	}

	// Later marks win a shared column, so the most telling ones go last.
	cells[a.col(s.P90)] = markP90
	cells[a.col(s.Min)] = markMin
	cells[p99] = markP99
	if clipped {
		cells[a.Width-1] = markClipped
	}
	fill(a.col(s.P25), a.col(s.P75), markBox)
	cells[a.col(s.P50)] = markMedian
	return string(cells), clipped
}

// Plot is a series drawn as one box per run under a shared axis. Rows line up
// with the runs, so a caller can style each run's row on its own.
type Plot struct {
	Header []string
	Rows   []string
}

// BoxPlot draws a series in about width columns, each row led by its run's
// name.
func BoxPlot(names []string, s Series, width int) Plot {
	nw := nameWidth(names)
	axis := NewAxis(s, 0)

	// A clipped max is written after the plot, so leave room for the longest.
	tail := 0
	for _, sum := range s.Summaries {
		if sum != nil && sum.Max > axis.Max {
			tail = max(tail, 1+utf8.RuneCountInString(Format(s.Unit, sum.Max)))
		}
	}
	axis.Width = max(width-nw-tail, minPlotWidth)

	indent := strings.Repeat(" ", nw)
	labels, line := axis.Ticks(s.Unit)
	p := Plot{Header: []string{indent + labels, indent + line}}

	for i, sum := range s.Summaries {
		name := fitName(names[i], nw)
		if sum == nil {
			p.Rows = append(p.Rows, name+"no data")
			continue
		}
		row, clipped := axis.Row(*sum)
		if clipped {
			row += " " + Format(s.Unit, sum.Max)
		}
		p.Rows = append(p.Rows, strings.TrimRight(name+row, " "))
	}
	return p
}

// nameWidth is the column run names are padded to, so rows line up after them.
// Long names are cut before they take the plot's room.
func nameWidth(names []string) int {
	return min(longest(names), maxNameWidth) + 2
}

// fitName cuts and pads a run name to a column of nameWidth.
func fitName(name string, width int) string {
	return pad(clip(name, width-2), width)
}

func longest(ss []string) int {
	n := 0
	for _, s := range ss {
		n = max(n, utf8.RuneCountInString(s))
	}
	return n
}

func pad(s string, width int) string {
	return s + strings.Repeat(" ", max(0, width-utf8.RuneCountInString(s)))
}

// clip cuts s to width runes, marking the cut.
func clip(s string, width int) string {
	if utf8.RuneCountInString(s) <= width {
		return s
	}
	if width <= 0 {
		return ""
	}
	return string([]rune(s)[:width-1]) + "…"
}
