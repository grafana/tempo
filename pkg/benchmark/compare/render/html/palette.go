package html

import "github.com/grafana/tempo/v3/pkg/benchmark/compare/render"

// The palette, as CSS writes colours, each readable on white.
const (
	dimColor  = "#6e7781"
	warnColor = "#b35900"
	// Blue for better and orange for worse read for most colour-blind eyes
	// as well.
	betterColor = "#0969da"
	worseColor  = "#c24e00"
)

// runColors tell runs apart, and stay clear of the blue and orange changes are
// read in.
var runColors = []string{
	"#a3329a", // magenta
	"#2e7d4f", // green
	"#9a7300", // ochre
	"#0b7fa3", // cyan
	"#6c4bbd", // purple
	"#c2185b", // pink
	"#00796b", // teal
	"#7d5a2b", // brown
}

// color is the colour a piece is shown in, or nothing for one shown plainly.
func color(t render.Text) string {
	switch t.Style {
	case render.Dim:
		return dimColor
	case render.Warn:
		return warnColor
	case render.Better, render.MuchBetter:
		return betterColor
	case render.Worse, render.MuchWorse:
		return worseColor
	case render.RunName, render.BaselineName:
		return runColors[t.Run%len(runColors)]
	default:
		return ""
	}
}
