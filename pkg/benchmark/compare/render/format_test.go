package render

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
)

func TestFormat(t *testing.T) {
	tests := []struct {
		unit compare.Unit
		v    float64
		want string
	}{
		{compare.Nanoseconds, 0, "0ns"},
		{compare.Nanoseconds, 850, "850ns"},
		{compare.Nanoseconds, 68_541, "68.5µs"},
		{compare.Nanoseconds, 32_336_542, "32.3ms"},
		{compare.Nanoseconds, 10_000_000, "10ms"},
		{compare.Nanoseconds, 1_115_700_459, "1.12s"},
		// Rounding that reaches the next suffix moves to it.
		{compare.Nanoseconds, 999_700, "1ms"},
		// Past the largest suffix the number grows instead.
		{compare.Nanoseconds, 12_345e9, "12300s"},
		{compare.Seconds, 0.0031, "3.1ms"},
		{compare.Bytes, 844, "844B"},
		{compare.Bytes, 86_890_486, "86.9MB"},
		{compare.Bytes, 1_500_000_000, "1.5GB"},
		{compare.Count, 204, "204"},
		{compare.Count, 2_031_661, "2.03M"},
		{compare.Count, 0.1626, "0.163"},
		{compare.Count, 0.000123, "0.000123"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			require.Equal(t, tt.want, Format(tt.unit, tt.v))
		})
	}
}

func TestFormatChange(t *testing.T) {
	require.Equal(t, "-13.0%", formatChange(-13.04))
	require.Equal(t, "+4.0%", formatChange(4))
	require.Equal(t, "0%", formatChange(0), "no change is not a signed zero")
}
