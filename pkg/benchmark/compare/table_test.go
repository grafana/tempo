package compare

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
)

func TestTable(t *testing.T) {
	base := summary(15e6, 26e6, 32.3e6, 37e6, 40.3e6, 47.8e6, 1.12e9)
	faster := summary(14e6, 23e6, 28.1e6, 33e6, 36.9e6, 44e6, 0.98e9)
	s := Series{Unit: Nanoseconds, Summaries: []*metrics.Summary{&base, &faster, nil}}

	header, rows := Table([]string{"base", "rb-4M", "gone"}, s, 0)
	require.Equal(t, "run     n     p50     p90     p99    max    Δp50   Δp99", header)
	require.Equal(t, []string{
		"base   10  32.3ms  40.3ms  47.8ms  1.12s",
		"rb-4M  10  28.1ms  36.9ms    44ms  980ms  -13.0%  -7.9%",
		"gone    –       –       –       –      –",
	}, rows)

	// Against another baseline the deltas are from it instead.
	_, rows = Table([]string{"base", "rb-4M", "gone"}, s, 1)
	require.Equal(t, "base   10  32.3ms  40.3ms  47.8ms  1.12s  +14.9%  +8.6%", rows[0])
	require.Equal(t, "rb-4M  10  28.1ms  36.9ms    44ms  980ms", rows[1])

	// A baseline without data has nothing to compare against.
	_, rows = Table([]string{"base", "rb-4M", "gone"}, s, 2)
	require.Equal(t, "base   10  32.3ms  40.3ms  47.8ms  1.12s", rows[0])

	zero := summary(0, 0, 0, 0, 0, 0, 0)
	s = Series{Unit: Count, Summaries: []*metrics.Summary{&zero, &faster}}
	_, rows = Table([]string{"a", "b"}, s, 0)
	require.Contains(t, rows[1], "n/a")
}
