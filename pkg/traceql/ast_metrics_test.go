package traceql

import "testing"

func BenchmarkMetricsFilter_compare(b *testing.B) {
	m := newMetricsFilter(OpGreater, 100)

	b.ResetTimer()
	for i := range b.N {
		m.compare(float64(i))
	}
}
