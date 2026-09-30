package term

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare/internal/comparetest"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare/render"
	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
)

func TestAxisTicks(t *testing.T) {
	axis := render.Axis{Min: 0, Max: 50, Step: 10}
	a := charAxis{Axis: axis, width: 51}
	labels, line := a.ticks(axis.Ticks(compare.Count))

	gap := strings.Repeat(" ", 8)
	require.Equal(t, "0"+gap+"10"+gap+"20"+gap+"30"+gap+"40"+gap+"50", labels)

	seg := strings.Repeat("─", 9)
	require.Equal(t, "├"+seg+"┼"+seg+"┼"+seg+"┼"+seg+"┼"+seg+"┤", line)

	// Labels that would touch are dropped rather than overlapped, and the ends
	// are kept over the middle, since they say what the axis spans.
	a.width = 20
	labels, _ = a.ticks(axis.Ticks(compare.Nanoseconds))
	require.Equal(t, "0ns   20ns      50ns", labels)
}

func TestAxisRow(t *testing.T) {
	a := charAxis{Axis: render.Axis{Min: 0, Max: 50, Step: 10}, width: 51}
	rep := strings.Repeat

	require.Equal(t,
		rep(" ", 10)+"├"+rep("─", 9)+rep("▒", 5)+"┃"+rep("▒", 5)+rep("─", 4)+"╎"+rep("─", 4)+"┤"+rep("·", 4)+"•"+rep(" ", 5),
		a.row(comparetest.Summary(10, 20, 25, 30, 35, 40, 45), false))

	require.Equal(t,
		rep(" ", 15)+"├"+rep("─", 10)+rep("▒", 6)+"┃"+rep("▒", 5)+rep("─", 2)+"╎"+rep("─", 7)+"┤"+"·"+"▸",
		a.row(comparetest.Summary(15, 26, 32, 37, 40, 48, 1116), true))

	// A distribution narrower than a column still shows its median.
	require.Equal(t, rep(" ", 25)+"┃"+rep(" ", 25), a.row(comparetest.Summary(25, 25, 25, 25, 25, 25, 25), false))
}

func TestBoxPlot(t *testing.T) {
	far := comparetest.Summary(15, 26, 32, 37, 40, 48, 1116)
	near := comparetest.Summary(14, 23, 28, 33, 36, 44, 49)
	s := compare.Series{Unit: compare.Count, Summaries: []*metrics.Summary{&far, &near, nil}}

	p := BoxPlot(render.NewBoxPlot([]string{"base", "rb-4M", "gone"}, s, 0), 80)
	require.Len(t, p.Header, 2)
	require.Len(t, p.Rows, 3)

	indent := strings.Repeat(" ", len("rb-4M")+2)
	for _, h := range p.Header {
		require.True(t, strings.HasPrefix(h.String(), indent), "header %q is indented past the names", h)
		require.Equal(t, render.Dim, h[0].Style)
	}
	require.True(t, strings.HasPrefix(p.Rows[0].String(), "base   "))
	require.True(t, strings.HasSuffix(p.Rows[0].String(), "▸ 1.12k"), "a clipped max is written after the plot: %q", p.Rows[0])
	require.True(t, strings.HasPrefix(p.Rows[1].String(), "rb-4M  "))
	require.Equal(t, "gone   no data", p.Rows[2].String())

	// Each row is shown as its run, the baseline's in its own style.
	require.Equal(t, render.Text{Text: p.Rows[0].String(), Style: render.BaselineName, Run: 0}, p.Rows[0][0])
	require.Equal(t, render.RunName, p.Rows[1][0].Style)
	require.Equal(t, 1, p.Rows[1][0].Run)

	// Room for the clipped max is taken out of the plot, so the rows fit.
	for _, r := range p.Rows[:2] {
		require.LessOrEqual(t, utf8.RuneCountInString(r.String()), 80)
	}
	require.Equal(t, 80, utf8.RuneCountInString(p.Rows[0].String()))
}
