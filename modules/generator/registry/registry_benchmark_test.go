package registry

import (
	"fmt"
	"runtime"
	"runtime/debug"
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

// BenchmarkLabelBuilderPoolsMultiTenant measures how the per-registry pools
// scale with tenant count. One benchmark operation covers every tenant once.
// Run with -cpu to expose sync.Pool's GOMAXPROCS-sized local-cache cost, for
// example:
//
//	go test ./modules/generator/registry -run '^$' \
//		-bench BenchmarkLabelBuilderPoolsMultiTenant -benchmem -cpu 5
func BenchmarkLabelBuilderPoolsMultiTenant(b *testing.B) {
	type benchmarkLabel struct {
		name  string
		value string
	}

	newPools := func(tenants int) []*labelBuilderPools {
		pools := make([]*labelBuilderPools, tenants)
		for i := range pools {
			pools[i] = newLabelBuilderPools()
		}
		return pools
	}

	buildLabels := func(b *testing.B, pools *labelBuilderPools, benchmarkLabels []benchmarkLabel) {
		builder := pools.newLabelBuilder(0, 0, noopSanitizer{}, noopLabelLimiter{})
		for _, label := range benchmarkLabels {
			builder.Add(label.name, label.value)
		}
		borrowed, ok := builder.CloseAndBorrowLabels()
		if !ok {
			b.Fatal("expected valid labels")
		}
		borrowed.Release()
	}

	makeLabels := func(count int) []benchmarkLabel {
		benchmarkLabels := make([]benchmarkLabel, count)
		for i := range benchmarkLabels {
			benchmarkLabels[i] = benchmarkLabel{
				name:  fmt.Sprintf("label_%03d", i),
				value: fmt.Sprintf("value_%03d", i),
			}
		}
		return benchmarkLabels
	}

	defaultLabels := []benchmarkLabel{
		{name: "service", value: "frontend"},
		{name: "span_name", value: "GET /api/users"},
		{name: "status_code", value: "200"},
	}
	tests := []struct {
		name    string
		tenants int
		labels  []benchmarkLabel
	}{
		{name: "tenants=1", tenants: 1, labels: defaultLabels},
		{name: "tenants=1000", tenants: 1_000, labels: defaultLabels},
		{name: "tenants=10000", tenants: 10_000, labels: defaultLabels},
		{name: "tenants=500,labels=32", tenants: 500, labels: makeLabels(32)},
		{name: "tenants=100,labels=256", tenants: 100, labels: makeLabels(256)},
	}

	for _, test := range tests {
		b.Run(test.name, func(b *testing.B) {
			b.Run("first_touch", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					pools := newPools(test.tenants)
					for _, pool := range pools {
						buildLabels(b, pool, test.labels)
					}
					runtime.KeepAlive(pools)
				}
				b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(test.tenants), "ns/tenant")
			})

			b.Run("gc_cycle_and_retouch", func(b *testing.B) {
				pools := newPools(test.tenants)
				for _, pool := range pools {
					buildLabels(b, pool, test.labels)
				}

				// Include the complete periodic cost: GC moves cached objects to
				// victim caches and clears primary pool locals, then the first
				// tenant pass recreates primary locals while reusing victims.
				b.ReportAllocs()
				for b.Loop() {
					runtime.GC()
					for _, pool := range pools {
						buildLabels(b, pool, test.labels)
					}
				}
				runtime.KeepAlive(pools)
				b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(test.tenants), "ns/tenant")
			})

			b.Run("steady_state_no_gc", func(b *testing.B) {
				// gc_cycle_and_retouch covers the periodic GC cost. Disable GC
				// before warmup here to isolate pool reuse between GC cycles.
				gcPercent := debug.SetGCPercent(-1)
				defer debug.SetGCPercent(gcPercent)

				pools := newPools(test.tenants)
				for _, pool := range pools {
					buildLabels(b, pool, test.labels)
				}

				b.ReportAllocs()
				for b.Loop() {
					for _, pool := range pools {
						buildLabels(b, pool, test.labels)
					}
				}
				runtime.KeepAlive(pools)
				b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(test.tenants), "ns/tenant")
			})

			if len(test.labels) > initialLabelCapacity {
				b.Run("default_after_high_water_no_gc", func(b *testing.B) {
					gcPercent := debug.SetGCPercent(-1)
					defer debug.SetGCPercent(gcPercent)

					pools := newPools(test.tenants)
					for _, pool := range pools {
						buildLabels(b, pool, test.labels)
					}

					b.ReportAllocs()
					for b.Loop() {
						for _, pool := range pools {
							buildLabels(b, pool, defaultLabels)
						}
					}
					runtime.KeepAlive(pools)
					b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(test.tenants), "ns/tenant")
				})
			}
		})
	}
}
