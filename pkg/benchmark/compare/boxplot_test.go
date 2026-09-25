package compare

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
)

func TestNiceStep(t *testing.T) {
	tests := []struct {
		raw, want float64
	}{
		{6.6, 5},
		{9, 10},
		{1.2, 1},
		{25, 20},
		{0.16, 0.2},
		{3_300_000, 5_000_000},
		{0, 1},
		{-1, 1},
	}
	for _, tt := range tests {
		require.InDelta(t, tt.want, niceStep(tt.raw), 1e-9, "raw %v", tt.raw)
	}
}

func TestNewAxis(t *testing.T) {
	series := func(sums ...*metrics.Summary) Series { return Series{Summaries: sums} }
	sum := func(minV, p99, maxV float64) *metrics.Summary {
		s := summary(minV, minV, minV, p99, p99, p99, maxV)
		return &s
	}

	tests := []struct {
		name     string
		s        Series
		min, max float64
	}{
		{"a far max is clipped", series(sum(15, 48, 1116)), 15, 50},
		{"a close max is taken in", series(sum(10, 50, 55)), 10, 60},
		{"spans every run", series(sum(20, 30, 30), nil, sum(5, 45, 45)), 0, 50},
		{"a constant is padded", series(sum(4, 4, 4)), 3.6, 4.4},
		{"a zero is padded upwards", series(sum(0, 0, 0)), 0, 1},
		{"no data", series(nil), 0, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAxis(tt.s, 40)
			require.InDelta(t, tt.min, a.Min, 1e-9)
			require.InDelta(t, tt.max, a.Max, 1e-9)
			require.Equal(t, 40, a.Width)
		})
	}

	require.Equal(t, minPlotWidth, NewAxis(series(nil), 1).Width)
}

func TestAxisTicks(t *testing.T) {
	a := Axis{Min: 0, Max: 50, Step: 10, Width: 51}
	labels, line := a.Ticks(Count)

	gap := strings.Repeat(" ", 8)
	require.Equal(t, "0"+gap+"10"+gap+"20"+gap+"30"+gap+"40"+gap+"50", labels)

	seg := strings.Repeat("─", 9)
	require.Equal(t, "├"+seg+"┼"+seg+"┼"+seg+"┼"+seg+"┼"+seg+"┤", line)

	// Labels that would touch are dropped rather than overlapped, and the ends
	// are kept over the middle, since they say what the axis spans.
	a = Axis{Min: 0, Max: 50, Step: 10, Width: 20}
	labels, _ = a.Ticks(Nanoseconds)
	require.Equal(t, "0ns   20ns      50ns", labels)
}

func TestAxisRow(t *testing.T) {
	a := Axis{Min: 0, Max: 50, Step: 10, Width: 51}
	rep := strings.Repeat

	row, clipped := a.Row(summary(10, 20, 25, 30, 35, 40, 45))
	require.False(t, clipped)
	require.Equal(t,
		rep(" ", 10)+"├"+rep("─", 9)+rep("▒", 5)+"┃"+rep("▒", 5)+rep("─", 4)+"╎"+rep("─", 4)+"┤"+rep("·", 4)+"•"+rep(" ", 5),
		row)

	row, clipped = a.Row(summary(15, 26, 32, 37, 40, 48, 1116))
	require.True(t, clipped)
	require.Equal(t,
		rep(" ", 15)+"├"+rep("─", 10)+rep("▒", 6)+"┃"+rep("▒", 5)+rep("─", 2)+"╎"+rep("─", 7)+"┤"+"·"+"▸",
		row)

	// A distribution narrower than a column still shows its median.
	row, _ = a.Row(summary(25, 25, 25, 25, 25, 25, 25))
	require.Equal(t, rep(" ", 25)+"┃"+rep(" ", 25), row)
}

func TestBoxPlot(t *testing.T) {
	far := summary(15, 26, 32, 37, 40, 48, 1116)
	near := summary(14, 23, 28, 33, 36, 44, 49)
	s := Series{Unit: Count, Summaries: []*metrics.Summary{&far, &near, nil}}

	p := BoxPlot([]string{"base", "rb-4M", "gone"}, s, 80)
	require.Len(t, p.Header, 2)
	require.Len(t, p.Rows, 3)

	indent := strings.Repeat(" ", len("rb-4M")+2)
	for _, h := range p.Header {
		require.True(t, strings.HasPrefix(h, indent), "header %q is indented past the names", h)
	}
	require.True(t, strings.HasPrefix(p.Rows[0], "base   "))
	require.True(t, strings.HasSuffix(p.Rows[0], "▸ 1.12k"), "a clipped max is written after the plot: %q", p.Rows[0])
	require.True(t, strings.HasPrefix(p.Rows[1], "rb-4M  "))
	require.Equal(t, "gone   no data", p.Rows[2])

	// Room for the clipped max is taken out of the plot, so the rows fit.
	for _, r := range p.Rows[:2] {
		require.LessOrEqual(t, utf8.RuneCountInString(r), 80)
	}
	require.Equal(t, 80, utf8.RuneCountInString(p.Rows[0]))
}
