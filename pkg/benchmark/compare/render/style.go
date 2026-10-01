package render

import "math"

// Style says what a piece of a view means, so that every output shows it the
// same way, each by its own means.
type Style int

const (
	Plain Style = iota
	// Title names what a view shows.
	Title
	// Dim is scaffolding, and changes too small to call one.
	Dim
	// Warn says a run cannot be compared, or ran somewhere else.
	Warn
	// Better and Worse are changes of at least MinorChange, and MuchBetter
	// and MuchWorse changes of MajorChange or more. Every default metric is
	// better lower.
	Better
	MuchBetter
	Worse
	MuchWorse
	// RunName shows something as the run it belongs to, and BaselineName as
	// the baseline, which stands out since every change is measured from it.
	RunName
	BaselineName
)

// Flag marks a case a run cannot be compared on, or a run that happened
// somewhere else.
const Flag = "⚠"

// Text is a piece of a view, with what it means.
type Text struct {
	Text  string
	Style Style
	// Run is the run a RunName or BaselineName piece belongs to.
	Run int
}

// Color is a colour, as each output writes it.
type Color struct {
	// Hex is the colour as CSS writes it, readable on white.
	Hex string
}

var (
	dimColor  = Color{Hex: "#6e7781"}
	warnColor = Color{Hex: "#b35900"}
	// Blue for better and orange for worse read for most colour-blind eyes
	// as well.
	betterColor = Color{Hex: "#0969da"}
	worseColor  = Color{Hex: "#c24e00"}
	// runColors tell runs apart, and stay clear of the blue and orange changes
	// are read in.
	runColors = []Color{
		{Hex: "#a3329a"}, // magenta
		{Hex: "#2e7d4f"}, // green
		{Hex: "#9a7300"}, // ochre
		{Hex: "#0b7fa3"}, // cyan
		{Hex: "#6c4bbd"}, // purple
		{Hex: "#c2185b"}, // pink
		{Hex: "#00796b"}, // teal
		{Hex: "#7d5a2b"}, // brown
	}
)

// Color is the colour a piece is shown in, if it has one.
func (t Text) Color() (Color, bool) {
	switch t.Style {
	case Dim:
		return dimColor, true
	case Warn:
		return warnColor, true
	case Better, MuchBetter:
		return betterColor, true
	case Worse, MuchWorse:
		return worseColor, true
	case RunName, BaselineName:
		return runColors[t.Run%len(runColors)], true
	default:
		return Color{}, false
	}
}

// Bold says whether a piece in this style stands out.
func (s Style) Bold() bool {
	switch s {
	case Title, MuchBetter, MuchWorse, BaselineName:
		return true
	default:
		return false
	}
}

// changeStyle styles a change by which way it went, and how far.
func changeStyle(pct float64) Style {
	a := math.Abs(pct)
	switch {
	case a < MinorChange:
		return Dim
	case pct < 0 && a >= MajorChange:
		return MuchBetter
	case pct < 0:
		return Better
	case a >= MajorChange:
		return MuchWorse
	default:
		return Worse
	}
}

// runText shows text as the run it belongs to.
func runText(text string, run, baseline int) Text {
	style := RunName
	if run == baseline {
		style = BaselineName
	}
	return Text{Text: text, Style: style, Run: run}
}
