package render

import (
	"strings"
	"unicode/utf8"
)

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

// maxNameWidth cuts long run names, like several settings run together,
// before they take a plot's or table's room.
const maxNameWidth = 28

// nameWidth is the column run names are padded to, so rows line up after them.
func nameWidth(names []string) int {
	return min(longest(names), maxNameWidth) + 2
}

// fitName cuts and pads a run name to a column of nameWidth.
func fitName(name string, width int) string {
	return pad(Clip(name, width-2), width)
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

func padLeft(s string, width int) string {
	return strings.Repeat(" ", max(0, width-utf8.RuneCountInString(s))) + s
}

func center(s string, width int) string {
	gap := max(0, width-utf8.RuneCountInString(s))
	return strings.Repeat(" ", gap/2) + s + strings.Repeat(" ", gap-gap/2)
}

// Clip cuts s to width runes, marking the cut.
func Clip(s string, width int) string {
	if utf8.RuneCountInString(s) <= width {
		return s
	}
	if width <= 0 {
		return ""
	}
	return string([]rune(s)[:width-1]) + "…"
}
