package render

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare/internal/comparetest"
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
	series := func(sums ...*metrics.Summary) compare.Series { return compare.Series{Summaries: sums} }
	sum := func(minV, p99, maxV float64) *metrics.Summary {
		s := comparetest.Summary(minV, minV, minV, p99, p99, p99, maxV)
		return &s
	}

	tests := []struct {
		name     string
		s        compare.Series
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
			a := NewAxis(tt.s)
			require.InDelta(t, tt.min, a.Min, 1e-9)
			require.InDelta(t, tt.max, a.Max, 1e-9)
		})
	}
}

func TestAxisTicks(t *testing.T) {
	require.Equal(t, []Tick{
		{Value: 10, Label: "10ns"}, {Value: 20, Label: "20ns"}, {Value: 30, Label: "30ns"},
	}, Axis{Min: 10, Max: 30, Step: 10}.Ticks(compare.Nanoseconds))
}

func TestNewBoxPlot(t *testing.T) {
	far := comparetest.Summary(15, 26, 32, 37, 40, 48, 1116)
	near := comparetest.Summary(14, 23, 28, 33, 36, 44, 49)
	s := compare.Series{Unit: compare.Count, Summaries: []*metrics.Summary{&far, &near, nil}}

	v := NewBoxPlot([]string{"base", "a", "gone"}, s, 1)
	require.Equal(t, compare.Count, v.Unit)
	require.Equal(t, Axis{Min: 10, Max: 50, Step: 5}, v.Axis)
	require.Len(t, v.Ticks, 9)
	require.Equal(t, Tick{Value: 50, Label: "50"}, v.Ticks[8])

	// A max past the axis is clipped, and written out for an output to show.
	require.Equal(t, BoxPlotRow{Name: Text{Text: "base", Style: RunName}, Summary: &far, Clipped: true, Max: "1.12k"}, v.Rows[0])
	require.Equal(t, BoxPlotRow{Name: Text{Text: "a", Style: BaselineName, Run: 1}, Summary: &near}, v.Rows[1])
	require.Nil(t, v.Rows[2].Summary)
}
