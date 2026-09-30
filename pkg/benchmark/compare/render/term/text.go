// Package term lays views out in text for a terminal: padded tables, box plots
// drawn in characters, and lines of runs. Lines keep what each piece means
// until Paint styles them, which is the one place a view's meaning becomes a
// terminal's colours.
package term

import (
	"strings"
	"unicode/utf8"

	"charm.land/lipgloss/v2"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare/render"
)

// Line is a line of text, piece by piece.
type Line []render.Text

// String is the line's plain text.
func (l Line) String() string {
	var b strings.Builder
	for _, t := range l {
		b.WriteString(t.Text)
	}
	return b.String()
}

// Paint styles a line for a terminal.
func Paint(l Line) string {
	var b strings.Builder
	for _, t := range l {
		b.WriteString(textStyle(t).Render(t.Text))
	}
	return b.String()
}

// Style is how a style shows in a terminal, for text around a view. A style
// that shows a run needs the run, and Paint gives it one.
func Style(s render.Style) lipgloss.Style {
	return textStyle(render.Text{Style: s})
}

func textStyle(t render.Text) lipgloss.Style {
	style := lipgloss.NewStyle()
	if c, ok := t.Color(); ok {
		style = style.Foreground(lipgloss.Color(c.ANSI))
	}
	return style.Bold(t.Style.Bold())
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
