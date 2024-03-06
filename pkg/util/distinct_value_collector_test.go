package util

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDistinctValueCollectorDiff(t *testing.T) {
	d := NewDistinctValueCollector[string](0, func(s string) int { return len(s) })

	d.Collect("123")
	d.Collect("4567")

	stringsSlicesEqual(t, []string{"123", "4567"}, d.Diff())
	stringsSlicesEqual(t, []string{}, d.Diff())

	d.Collect("123")
	d.Collect("890")

	stringsSlicesEqual(t, []string{"890"}, d.Diff())
	stringsSlicesEqual(t, []string{}, d.Diff())
}

func stringsSlicesEqual(t *testing.T, a, b []string) {
	sort.Strings(a)
	sort.Strings(b)
	require.Equal(t, a, b)
}
