package compare

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnitFor(t *testing.T) {
	tests := []struct {
		metric string
		want   Unit
	}{
		{"harness.wallNs", Nanoseconds},
		{"backend.timeNs", Nanoseconds},
		{"harness.allocBytes", Bytes},
		{"backend.bytes", Bytes},
		{"response.inspectedBytes", Bytes},
		{"process.go_memstats_alloc_bytes", Bytes},
		{"process.go_memstats_alloc_bytes_total", Bytes},
		{"process.go_gc_duration_seconds_sum", Seconds},
		{"process.process_cpu_seconds_total", Seconds},
		{"process.go_gc_duration_seconds_count", Count},
		{"backend.reads", Count},
		{"harness.allocCount", Count},
		{"response.inspectedSpans", Count},
	}
	for _, tt := range tests {
		t.Run(tt.metric, func(t *testing.T) {
			require.Equal(t, tt.want, UnitFor(tt.metric))
		})
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		unit Unit
		v    float64
		want string
	}{
		{Nanoseconds, 0, "0ns"},
		{Nanoseconds, 850, "850ns"},
		{Nanoseconds, 68_541, "68.5µs"},
		{Nanoseconds, 32_336_542, "32.3ms"},
		{Nanoseconds, 10_000_000, "10ms"},
		{Nanoseconds, 1_115_700_459, "1.12s"},
		// Rounding that reaches the next suffix moves to it.
		{Nanoseconds, 999_700, "1ms"},
		// Past the largest suffix the number grows instead.
		{Nanoseconds, 12_345e9, "12300s"},
		{Seconds, 0.0031, "3.1ms"},
		{Bytes, 844, "844B"},
		{Bytes, 86_890_486, "86.9MB"},
		{Bytes, 1_500_000_000, "1.5GB"},
		{Count, 204, "204"},
		{Count, 2_031_661, "2.03M"},
		{Count, 0.1626, "0.163"},
		{Count, 0.000123, "0.000123"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			require.Equal(t, tt.want, Format(tt.unit, tt.v))
		})
	}
}
