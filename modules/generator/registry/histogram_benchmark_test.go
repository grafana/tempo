package registry

import (
	"testing"
	"time"

	"github.com/prometheus/prometheus/model/labels"
)

func BenchmarkRegistryHistogramObserve(b *testing.B) {
	lbls := labels.FromStrings("service", "frontend", "span_name", "GET /api")

	b.Run("classic_low_bucket", func(b *testing.B) {
		h := newHistogram("bench_histogram", []float64{0.01, 0.05, 0.1, 0.5, 1, 5}, noopLimiter, "trace_id", nil, 15*time.Minute)
		h.ObserveWithExemplar(lbls, 0.001, "trace-1", 1)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			h.ObserveWithExemplar(lbls, 0.001, "trace-1", 1)
		}
	})

	b.Run("classic_high_bucket", func(b *testing.B) {
		h := newHistogram("bench_histogram", []float64{0.01, 0.05, 0.1, 0.5, 1, 5}, noopLimiter, "trace_id", nil, 15*time.Minute)
		h.ObserveWithExemplar(lbls, 4, "trace-1", 1)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			h.ObserveWithExemplar(lbls, 4, "trace-1", 1)
		}
	})

	b.Run("native", func(b *testing.B) {
		h := newNativeHistogram("bench_native_histogram", []float64{0.01, 0.05, 0.1, 0.5, 1, 5}, noopLimiter, "trace_id", HistogramModeNative, nil, testTenant, &mockOverrides{}, 15*time.Minute)
		h.ObserveWithExemplar(lbls, 0.5, "trace-1", 1)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			h.ObserveWithExemplar(lbls, 0.5, "trace-1", 1)
		}
	})
}
