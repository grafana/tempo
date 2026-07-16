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

	// The path spanmetrics uses on the hot path: raw trace ID bytes plus a
	// precomputed label hash carried by BorrowedLabels (hand-constructed here
	// from owned labels, as documented on the type). This differs from
	// classic_low_bucket in several variables at once (trace ID form, hash
	// source, timestamp source), so compare the pair as a whole-path
	// before/after; don't attribute a delta to any single variable.
	b.Run("classic_trace_id_bytes_with_hash", func(b *testing.B) {
		h := newHistogram("bench_histogram", []float64{0.01, 0.05, 0.1, 0.5, 1, 5}, noopLimiter, "trace_id", nil, 15*time.Minute)
		traceID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}
		borrowed := &BorrowedLabels{Labels: lbls, Hash: lbls.Hash()}
		timeMs := time.Now().UnixMilli()
		h.ObserveBorrowed(borrowed, 0.001, traceID, 1, timeMs)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			h.ObserveBorrowed(borrowed, 0.001, traceID, 1, timeMs)
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
		// A zero bucket factor would disable sparse (native) buckets and fall
		// back to classic DefBuckets, so configure real native parameters.
		overrides := &mockOverrides{
			nativeHistogramBucketFactor:     1.5,
			nativeHistogramMaxBucketNumber:  100,
			nativeHistogramMinResetDuration: 15 * time.Minute,
		}
		h := newNativeHistogram("bench_native_histogram", []float64{0.01, 0.05, 0.1, 0.5, 1, 5}, noopLimiter, "trace_id", HistogramModeNative, nil, testTenant, overrides, 15*time.Minute)
		h.ObserveWithExemplar(lbls, 0.5, "trace-1", 1)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			h.ObserveWithExemplar(lbls, 0.5, "trace-1", 1)
		}
	})
}
