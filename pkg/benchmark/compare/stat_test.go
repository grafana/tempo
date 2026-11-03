package compare

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStat(t *testing.T) {
	sum := summary(1, 2, 3, 4, 5, 6, 7)
	var names []string
	var values []float64
	for _, s := range Stats {
		names = append(names, s.String())
		values = append(values, s.Of(&sum))
	}
	require.Equal(t, []string{"min", "p25", "p50", "p75", "p90", "p99", "max"}, names)
	require.Equal(t, []float64{1, 2, 3, 4, 5, 6, 7}, values)
}

func TestParseStat(t *testing.T) {
	for _, s := range Stats {
		got, err := ParseStat(s.String())
		require.NoError(t, err)
		require.Equal(t, s, got)
	}
	_, err := ParseStat("p42")
	require.Error(t, err)
}
