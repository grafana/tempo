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
