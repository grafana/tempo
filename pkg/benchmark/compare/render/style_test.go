package render

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChangeStyle(t *testing.T) {
	tests := []struct {
		pct  float64
		want Style
	}{
		{0, Dim},
		{1.9, Dim},
		{-1.9, Dim},
		{2, Worse},
		{-2, Better},
		{-9.9, Better},
		{10, MuchWorse},
		{-45, MuchBetter},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, changeStyle(tt.pct), "%v%%", tt.pct)
	}
}

func TestTextColor(t *testing.T) {
	_, ok := Text{Style: Plain}.Color()
	require.False(t, ok, "plain text has no colour")
	_, ok = Text{Style: Title}.Color()
	require.False(t, ok, "a title stands out by weight")

	better, _ := Text{Style: Better}.Color()
	muchBetter, _ := Text{Style: MuchBetter}.Color()
	worse, _ := Text{Style: Worse}.Color()
	require.Equal(t, better, muchBetter, "how far a change went shows in weight, not colour")
	require.NotEqual(t, better, worse)

	// Runs have a colour each, in turn, whether or not they are the baseline.
	first, _ := Text{Style: RunName, Run: 0}.Color()
	second, _ := Text{Style: RunName, Run: 1}.Color()
	baseline, _ := Text{Style: BaselineName, Run: 0}.Color()
	again, _ := Text{Style: RunName, Run: len(runColors)}.Color()
	require.NotEqual(t, first, second)
	require.Equal(t, first, baseline)
	require.Equal(t, first, again)
}

func TestStyleBold(t *testing.T) {
	for _, s := range []Style{Title, MuchBetter, MuchWorse, BaselineName} {
		require.True(t, s.Bold(), "style %d", s)
	}
	for _, s := range []Style{Plain, Dim, Warn, Better, Worse, RunName} {
		require.False(t, s.Bold(), "style %d", s)
	}
}
