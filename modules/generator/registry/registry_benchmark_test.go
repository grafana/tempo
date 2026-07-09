package registry

import (
	"testing"
	"time"

	"github.com/go-kit/log"
)

// BenchmarkManagedRegistryBorrowPath measures the production label-building hot
// path end to end: NewLabelBuilder -> Add -> CloseAndBorrowLabels (sort +
// sanitize + per-label limit + hash + utf8 check) -> hash-keyed metric update
// -> Release, against a real ManagedRegistry with its per-tenant pools. This is
// the path spanmetrics drives per span and the one the borrow API optimizes.
// BenchmarkSpanMetricsPushSpans uses registry.TestRegistry, which ignores the
// precomputed hash and stringifies labels, so it does not exercise this path.
//
// The variants toggle the sanitizer and per-label limiter so their per-span
// cost is visible: the sanitizer strings.Clone's the span name on every
// Sanitize call, while the limiter only clones a label name the first time it
// is seen. The label set is identical every iteration, so after the first
// iteration this measures the steady-state existing-series path (map hit, no
// new-series allocation).
func BenchmarkManagedRegistryBorrowPath(b *testing.B) {
	variants := []struct {
		name      string
		overrides *mockOverrides
	}{
		{"plain", &mockOverrides{}},
		{"sanitizer", &mockOverrides{spanNameSanitization: SpanNameSanitizationEnabled}},
		{"limiter", &mockOverrides{maxCardinalityPerLabel: 1_000_000}},
		{"sanitizer_and_limiter", &mockOverrides{
			spanNameSanitization:   SpanNameSanitizationEnabled,
			maxCardinalityPerLabel: 1_000_000,
		}},
	}

	traceID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}

	for _, v := range variants {
		b.Run(v.name, func(b *testing.B) {
			// Apply production defaults (e.g. MaxLabelNameLength/ValueLength) so
			// the measured path includes the label-length truncation checks.
			cfg := &Config{}
			cfg.RegisterFlagsAndApplyDefaults("", nil)
			reg := New(cfg, v.overrides, "test", &noopAppender{}, log.NewNopLogger(), noopLimiter)
			defer reg.Close()

			counter := reg.NewCounter("bench_counter")
			histogram := reg.NewHistogram("bench_histogram", []float64{0.1, 0.5, 1, 5}, HistogramModeClassic)
			ts := time.Now().UnixMilli()

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				builder := reg.NewLabelBuilder()
				builder.Add("service", "frontend")
				builder.Add("span_name", "GET /api/users")
				builder.Add("status_code", "200")
				borrowed, ok := builder.CloseAndBorrowLabels()
				if !ok {
					b.Fatal("expected valid labels")
				}
				counter.IncBorrowed(borrowed, 1, ts)
				histogram.ObserveBorrowed(borrowed, 0.42, traceID, 1, ts)
				borrowed.Release()
			}
		})
	}
}
