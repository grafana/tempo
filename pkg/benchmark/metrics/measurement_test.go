package metrics

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSummarize(t *testing.T) {
	require.Equal(t, Summary{}, summarize(nil))

	s := summarize([]float64{5, 1, 4, 2, 3})
	require.Equal(t, 5, s.Count)
	require.Equal(t, 1.0, s.Min)
	require.Equal(t, 3.0, s.P50)
	require.Equal(t, 5.0, s.Max)
	require.InDelta(t, 3.0, s.Mean, 0.001)

	// Quartiles are here so a box plot needs nothing else.
	require.Equal(t, 2.0, s.P25)
	require.Equal(t, 4.0, s.P75)

	// Percentiles are nearest rank, so they are values that were measured.
	one := summarize([]float64{42})
	require.Equal(t, 42.0, one.Min)
	require.Equal(t, 42.0, one.P99)
	require.Zero(t, one.StdDev)
}
