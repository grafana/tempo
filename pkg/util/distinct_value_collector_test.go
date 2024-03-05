package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDistinctValueCollectorDiff(t *testing.T) {
	d := NewDistinctValueCollector[string](0, func(s string) int { return len(s) })

	d.Collect("123")
	d.Collect("4567")

	require.Equal(t, []string{"123", "4567"}, d.Diff())
	require.Equal(t, []string{}, d.Diff())

	d.Collect("123")
	d.Collect("890")

	require.Equal(t, []string{"890"}, d.Diff())
	require.Equal(t, []string{}, d.Diff())
}
