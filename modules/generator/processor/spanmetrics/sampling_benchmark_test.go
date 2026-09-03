package spanmetrics

import (
	"context"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/model/exemplar"
	"github.com/prometheus/prometheus/model/histogram"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/metadata"
	prometheus_storage "github.com/prometheus/prometheus/storage"

	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/modules/overrides/histograms"
	"github.com/grafana/tempo/pkg/sharedconfig"
)

// BenchmarkSpanMetricsSampling measures what per-series sampling buys, against
// a real ManagedRegistry so the label pipeline the sampler skips (sort,
// sanitize, per-label limit, hash, UTF-8 validation) is included at production
// cost. BenchmarkSpanMetricsPushSpans uses registry.TestRegistry, which
// stringifies labels into a map and would overstate the saving.
//
// The clock is driven by the benchmark rather than by time.Now so the sampler
// reaches its rate-adapted steady state instead of being measured inside its
// first window. Each push advances it by benchmarkSamplingPushInterval, which
// fixes the per-series span rate the sampler sees; rows are labelled with the
// budget they are given relative to that rate. The reported keep% is what the
// sampler actually let through, so a row whose keep% is 100 is telling you the
// budget was never binding.
func BenchmarkSpanMetricsSampling(b *testing.B) {
	// 4 resources x 100 identical spans per push, so each push is 100 spans for
	// each of 4 series. At one push per pushInterval that fixes the per-series
	// span rate the sampler observes.
	const (
		spansPerPush          = 400
		spansPerSeriesPerPush = 100
		pushInterval          = 20 * time.Millisecond
	)
	spansPerSeriesPerSecond := int(float64(spansPerSeriesPerPush) / pushInterval.Seconds())

	// "plain" is the default span-metrics config. "prod" adds the per-span costs
	// a busy tenant actually pays -- six dimensions, a dimension mapping, span
	// name sanitization and the per-label cardinality limiter -- which is both a
	// higher baseline and a larger absolute saving.
	for _, shape := range []struct {
		name      string
		tune      func(*Config)
		overrides registry.Overrides
	}{
		{name: "plain", overrides: &benchmarkSamplingOverrides{}},
		{
			name: "prod",
			tune: func(cfg *Config) {
				cfg.Dimensions = []string{
					"k8s.cluster.name",
					"k8s.namespace.name",
					"http.method",
					"http.status_code",
					"span.attr.1",
					"span.attr.2",
				}
				cfg.DimensionMappings = []sharedconfig.DimensionMappings{
					{Name: "route_key", SourceLabel: []string{"http.method", "http.route", "http.status_code"}, Join: ":"},
				}
			},
			overrides: &benchmarkSamplingOverrides{
				sanitizeSpanNames:      true,
				maxCardinalityPerLabel: 1_000_000,
			},
		},
	} {
		for _, tc := range []struct {
			name   string
			budget int
		}{
			{"off", 0},
			{"keep_1_in_10", spansPerSeriesPerSecond / 10},
			{"keep_1_in_100", spansPerSeriesPerSecond / 100},
		} {
			b.Run(shape.name+"/"+tc.name, func(b *testing.B) {
				cfg := Config{}
				cfg.RegisterFlagsAndApplyDefaults("", nil)
				if shape.tune != nil {
					shape.tune(&cfg)
				}
				cfg.MaxSpansPerSeriesPerSecond = tc.budget

				regCfg := &registry.Config{}
				regCfg.RegisterFlagsAndApplyDefaults("", nil)
				reg := registry.New(regCfg, shape.overrides, "test", &benchmarkSamplingAppender{}, log.NewNopLogger(), benchmarkSamplingLimiter{})
				defer reg.Close()

				p, err := New(
					cfg,
					reg,
					prometheus.NewCounter(prometheus.CounterOpts{}),
					prometheus.NewCounter(prometheus.CounterOpts{}),
				)
				if err != nil {
					b.Fatal(err)
				}
				defer p.Shutdown(context.Background())

				processor := p.(*Processor)
				now := time.Unix(1_700_000_000, 0)
				processor.now = func() time.Time { return now }

				request := benchmarkSpanMetricsRequest(true)
				push := func() {
					now = now.Add(pushInterval)
					p.PushSpans(context.Background(), request)
				}

				// Warm up past a few sampler windows so the measured loop runs at
				// the block size the observed rate implies, rather than inside the
				// block-doubling burst guard that covers a series' first window.
				for warmup := time.Duration(0); warmup < 3*samplerWindow; warmup += pushInterval {
					push()
				}

				var seenBefore, keptBefore uint64
				if processor.sampler != nil {
					seenBefore, keptBefore = processor.sampler.counts()
				}

				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					push()
				}
				b.StopTimer()

				b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*spansPerPush), "ns/span")
				// keep% is what the sampler actually let through. A row reporting
				// 100 means its budget was never binding, so its timing is not a
				// sampling result.
				keep := 100.0
				if processor.sampler != nil {
					seen, kept := processor.sampler.counts()
					if seen > seenBefore {
						keep = 100 * float64(kept-keptBefore) / float64(seen-seenBefore)
					}
				}
				b.ReportMetric(keep, "keep%")
			})
		}
	}
}

// benchmarkSamplingOverrides applies the registry defaults, optionally turning
// on the two features that add per-span work: span name sanitization and the
// per-label cardinality limiter.
type benchmarkSamplingOverrides struct {
	sanitizeSpanNames      bool
	maxCardinalityPerLabel uint64
}

func (*benchmarkSamplingOverrides) MetricsGeneratorMaxActiveSeries(string) uint32 { return 0 }
func (*benchmarkSamplingOverrides) MetricsGeneratorMaxActiveEntities(string) uint32 {
	return 0
}

func (*benchmarkSamplingOverrides) MetricsGeneratorCollectionInterval(string) time.Duration {
	return 15 * time.Second
}
func (*benchmarkSamplingOverrides) MetricsGeneratorDisableCollection(string) bool { return true }
func (*benchmarkSamplingOverrides) MetricsGeneratorGenerateNativeHistograms(string) histograms.HistogramMethod {
	return histograms.HistogramMethodClassic
}
func (*benchmarkSamplingOverrides) MetricsGeneratorTraceIDLabelName(string) string { return "traceID" }
func (*benchmarkSamplingOverrides) MetricsGeneratorNativeHistogramBucketFactor(string) float64 {
	return 1.1
}
func (*benchmarkSamplingOverrides) MetricsGeneratorNativeHistogramMaxBucketNumber(string) uint32 {
	return 100
}
func (*benchmarkSamplingOverrides) MetricsGeneratorNativeHistogramMinResetDuration(string) time.Duration {
	return 15 * time.Minute
}
func (o *benchmarkSamplingOverrides) MetricsGeneratorSpanNameSanitization(string) string {
	if o.sanitizeSpanNames {
		return string(registry.SpanNameSanitizationEnabled)
	}
	return string(registry.SpanNameSanitizationDisabled)
}

func (o *benchmarkSamplingOverrides) MetricsGeneratorMaxCardinalityPerLabel(string) uint64 {
	return o.maxCardinalityPerLabel
}

type benchmarkSamplingLimiter struct{}

func (benchmarkSamplingLimiter) OnAdd(hash uint64, _ uint32, lbls labels.Labels) (labels.Labels, uint64) {
	return lbls, hash
}
func (benchmarkSamplingLimiter) OnUpdate(uint64, uint32) {}
func (benchmarkSamplingLimiter) OnDelete(uint64, uint32) {}

// benchmarkSamplingAppender satisfies storage.Appendable/Appender and stores
// nothing, so registry collection never enters the measurement.
type benchmarkSamplingAppender struct{}

var (
	_ prometheus_storage.Appendable = (*benchmarkSamplingAppender)(nil)
	_ prometheus_storage.Appender   = (*benchmarkSamplingAppender)(nil)
)

func (a *benchmarkSamplingAppender) Appender(context.Context) prometheus_storage.Appender {
	return a
}

func (*benchmarkSamplingAppender) Append(prometheus_storage.SeriesRef, labels.Labels, int64, float64) (prometheus_storage.SeriesRef, error) {
	return 0, nil
}

func (*benchmarkSamplingAppender) AppendExemplar(prometheus_storage.SeriesRef, labels.Labels, exemplar.Exemplar) (prometheus_storage.SeriesRef, error) {
	return 0, nil
}

func (*benchmarkSamplingAppender) AppendHistogram(prometheus_storage.SeriesRef, labels.Labels, int64, *histogram.Histogram, *histogram.FloatHistogram) (prometheus_storage.SeriesRef, error) {
	return 0, nil
}

func (*benchmarkSamplingAppender) AppendHistogramSTZeroSample(prometheus_storage.SeriesRef, labels.Labels, int64, int64, *histogram.Histogram, *histogram.FloatHistogram) (prometheus_storage.SeriesRef, error) {
	return 0, nil
}

func (*benchmarkSamplingAppender) AppendSTZeroSample(prometheus_storage.SeriesRef, labels.Labels, int64, int64) (prometheus_storage.SeriesRef, error) {
	return 0, nil
}

func (*benchmarkSamplingAppender) UpdateMetadata(prometheus_storage.SeriesRef, labels.Labels, metadata.Metadata) (prometheus_storage.SeriesRef, error) {
	return 0, nil
}

func (*benchmarkSamplingAppender) SetOptions(*prometheus_storage.AppendOptions) {}

func (*benchmarkSamplingAppender) Commit() error   { return nil }
func (*benchmarkSamplingAppender) Rollback() error { return nil }
