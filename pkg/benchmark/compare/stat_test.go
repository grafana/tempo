package compare

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStat(t *testing.T) {
	sum := summary(1, 2, 3, 4, 5, 6, 7)
	require.Equal(t, []string{"p50", "p90", "p99"}, []string{P50.String(), P90.String(), P99.String()})
	require.Equal(t, []float64{3, 5, 6}, []float64{P50.of(&sum), P90.of(&sum), P99.of(&sum)})
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
