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

func TestStyleBold(t *testing.T) {
	for _, s := range []Style{Title, MuchBetter, MuchWorse, BaselineName} {
		require.True(t, s.Bold(), "style %d", s)
	}
	for _, s := range []Style{Plain, Dim, Warn, Better, Worse, RunName} {
		require.False(t, s.Bold(), "style %d", s)
	}
}
