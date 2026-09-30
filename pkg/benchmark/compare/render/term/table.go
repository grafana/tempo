package term

import (
	"strings"
	"unicode/utf8"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare/render"
)

// Table lays a table out in columns, cut to width, with each run's row shown
// as the run.
func Table(v render.TableView, width int) (header Line, rows []Line) {
	cells := [][]string{v.Header}
	for _, r := range v.Rows {
		cells = append(cells, append([]string{Clip(r.Name.Text, maxNameWidth)}, r.Cells...))
	}
	lines := alignColumns(cells)
	header = Line{{Text: Clip(lines[0], width), Style: render.Dim}}
	for i, r := range v.Rows {
		rows = append(rows, Line{{Text: Clip(lines[i+1], width), Style: r.Name.Style, Run: r.Name.Run}})
	}
	return header, rows
}

// alignColumns pads cells into columns two spaces apart, the first aligned left
// and the rest, which are numbers, aligned right.
func alignColumns(cells [][]string) []string {
	widths := make([]int, len(cells[0]))
	for _, row := range cells {
		for j, cell := range row {
			widths[j] = max(widths[j], utf8.RuneCountInString(cell))
		}
	}

	lines := make([]string, 0, len(cells))
	for _, row := range cells {
		var b strings.Builder
		for j, cell := range row {
			gap := strings.Repeat(" ", widths[j]-utf8.RuneCountInString(cell))
			if j == 0 {
				b.WriteString(cell + gap)
				continue
			}
			b.WriteString("  " + gap + cell)
		}
		lines = append(lines, strings.TrimRight(b.String(), " "))
	}
	return lines
}
