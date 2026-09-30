package term

import "github.com/grafana/tempo/v3/pkg/benchmark/compare/render"

// Runs lays out how the runs differ, a line per run: its name, after its number
// when runs are numbered, padded so the descriptions line up, then how it was
// set up against the baseline.
func Runs(v render.RunsView) []Line {
	names := make([]string, len(v.Runs))
	labels := make([]string, len(v.Runs))
	for i, r := range v.Runs {
		names[i], labels[i] = r.Name.Text, r.Label
	}
	width := nameWidth(names)

	lines := make([]Line, len(v.Runs))
	for i, r := range v.Runs {
		lead := fitName(r.Name.Text, width)
		if v.Numbered {
			lead = pad(r.Label, longest(labels)+1) + lead
		}
		l := Line{{Text: lead, Style: r.Name.Style, Run: r.Name.Run}}
		for j, d := range r.Details {
			if j > 0 {
				l = append(l, render.Text{Text: " · ", Style: render.Dim})
			}
			l = append(l, d)
		}
		lines[i] = l
	}
	return lines
}
