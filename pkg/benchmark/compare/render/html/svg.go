package html

import (
	"fmt"
	"html/template"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare/render"
)

// A box plot in SVG, in pixels. Names are set in a monospace face, so their
// width can be told from their length.
const (
	svgRowHeight  = 30
	svgBoxHeight  = 16
	svgCharWidth  = 7.5
	svgPlotWidth  = 640
	svgTailWidth  = 80 // room for a clipped max's value
	svgAxisHeight = 30
	svgPad        = 8
	// maxNameWidth cuts long run names before they take the plot's room.
	maxNameWidth = 28
)

// boxPlotSVG draws a box plot: a row per run, shown as the run, on the view's
// axis. Every run's numbers are in a tooltip on its row.
func boxPlotSVG(v boxPlot) template.HTML {
	names := make([]string, len(v.Rows))
	longest := 0
	for i, r := range v.Rows {
		names[i] = clip(r.Name.Text, maxNameWidth)
		longest = max(longest, utf8.RuneCountInString(names[i]))
	}
	nameWidth := float64(longest)*svgCharWidth + 2*svgPad
	width := nameWidth + svgPlotWidth + svgTailWidth
	height := float64(len(v.Rows)*svgRowHeight + svgAxisHeight + 2*svgPad)

	x := func(val float64) float64 {
		f := (val - v.Axis.Min) / (v.Axis.Max - v.Axis.Min)
		return nameWidth + math.Max(0, math.Min(1, f))*svgPlotWidth
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<svg class="boxplot" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %.0f %.0f" width="%.0f" height="%.0f">`, width, height, width, height)

	for i, r := range v.Rows {
		y := float64(svgPad + i*svgRowHeight + svgRowHeight/2)
		c := color(r.Name)
		weight := "normal"
		if r.Name.Style.Bold() {
			weight = "bold"
		}
		fmt.Fprintf(&b, `<text class="name" x="%d" y="%.1f" fill="%s" font-weight="%s">%s</text>`,
			svgPad, y+4, c, weight, esc(names[i]))
		sum := r.Summary
		if sum == nil {
			fmt.Fprintf(&b, `<text class="nodata" x="%.1f" y="%.1f">no data</text>`, nameWidth, y+4)
			continue
		}

		f := func(val float64) string { return render.Format(v.Unit, val) }
		top, bottom := y-svgBoxHeight/2, y+svgBoxHeight/2
		fmt.Fprintf(&b, `<g stroke="%s" fill="%s"><title>%s</title>`, c, c, esc(fmt.Sprintf(
			"%s: min %s · p25 %s · p50 %s · p75 %s · p90 %s · p99 %s · max %s",
			r.Name.Text, f(sum.Min), f(sum.P25), f(sum.P50), f(sum.P75), f(sum.P90), f(sum.P99), f(sum.Max))))
		// The whisker runs from the min to the p99, with a tick at each end and
		// at the p90.
		line(&b, x(sum.Min), y, x(sum.P99), y, "")
		line(&b, x(sum.Min), top+3, x(sum.Min), bottom-3, "")
		line(&b, x(sum.P99), top+3, x(sum.P99), bottom-3, "")
		line(&b, x(sum.P90), top+5, x(sum.P90), bottom-5, "")
		fmt.Fprintf(&b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%d" fill-opacity="0.25"/>`,
			x(sum.P25), top, math.Max(1, x(sum.P75)-x(sum.P25)), svgBoxHeight)
		line(&b, x(sum.P50), top, x(sum.P50), bottom, ` stroke-width="2.5"`)
		// A max past the axis is marked at its edge and written out, rather than
		// squashing every box to make room for it.
		if r.Clipped {
			edge := nameWidth + svgPlotWidth
			line(&b, x(sum.P99), y, edge, y, ` stroke-dasharray="2 3"`)
			fmt.Fprintf(&b, `<path d="M%.1f %.1f l6 4 l-6 4 z" stroke="none"/>`, edge, y-4)
			fmt.Fprintf(&b, `<text class="clipped" x="%.1f" y="%.1f" stroke="none">%s</text>`, edge+10, y+4, esc(r.Max))
		} else if x(sum.Max) > x(sum.P99) {
			line(&b, x(sum.P99), y, x(sum.Max), y, ` stroke-dasharray="2 3"`)
			fmt.Fprintf(&b, `<circle cx="%.1f" cy="%.1f" r="2.5" stroke="none"/>`, x(sum.Max), y)
		}
		b.WriteString(`</g>`)
	}

	axisY := float64(svgPad + len(v.Rows)*svgRowHeight)
	line(&b, nameWidth, axisY, nameWidth+svgPlotWidth, axisY, ` class="axis"`)
	for i, t := range v.Ticks {
		anchor := "middle"
		switch i {
		case 0:
			anchor = "start"
		case len(v.Ticks) - 1:
			anchor = "end"
		}
		line(&b, x(t.Value), axisY, x(t.Value), axisY+4, ` class="axis"`)
		fmt.Fprintf(&b, `<text class="tick" x="%.1f" y="%.1f" text-anchor="%s">%s</text>`, x(t.Value), axisY+18, anchor, esc(t.Label))
	}

	b.WriteString(`</svg>`)
	// Every value written into the SVG is a number, a colour from the palette,
	// or escaped text.
	return template.HTML(b.String()) // #nosec G203
}

func line(b *strings.Builder, x1, y1, x2, y2 float64, attrs string) {
	fmt.Fprintf(b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"%s/>`, x1, y1, x2, y2, attrs)
}

func esc(s string) string {
	return template.HTMLEscapeString(s)
}

// clip cuts s to width runes, marking the cut.
func clip(s string, width int) string {
	if utf8.RuneCountInString(s) <= width {
		return s
	}
	return string([]rune(s)[:width-1]) + "…"
}
