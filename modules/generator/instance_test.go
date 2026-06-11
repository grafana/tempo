package generator

import (
	"context"
	"flag"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/modules/generator/processor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/prometheus/model/exemplar"
	"github.com/prometheus/prometheus/model/histogram"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/metadata"
	prometheus_storage "github.com/prometheus/prometheus/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/modules/generator/processor/hostinfo"
	"github.com/grafana/tempo/modules/generator/processor/servicegraphs"
	"github.com/grafana/tempo/modules/generator/processor/spanmetrics"
	"github.com/grafana/tempo/modules/generator/storage"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
)

func Test_instance_concurrency(t *testing.T) {
	// Both instances use the same overrides, this map will be accessed by both
	overrides := &mockOverrides{}
	overrides.processors = map[string]struct{}{
		processor.SpanMetricsName:   {},
		processor.ServiceGraphsName: {},
	}
	cfg := &Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})

	instance1, err := newInstance(cfg, "test", overrides, &noopStorage{}, log.NewNopLogger())
	assert.NoError(t, err)

	instance2, err := newInstance(cfg, "test", overrides, &noopStorage{}, log.NewNopLogger())
	assert.NoError(t, err)

	end := make(chan struct{})

	accessor := func(f func()) {
		for {
			select {
			case <-end:
				return
			default:
				f()
			}
		}
	}

	go accessor(func() {
		req := test.MakeBatch(1, nil)
		instance1.pushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*v1.ResourceSpans{req}})
	})

	go accessor(func() {
		req := test.MakeBatch(1, nil)
		instance2.pushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*v1.ResourceSpans{req}})
	})

	go accessor(func() {
		err := instance1.updateProcessors()
		assert.NoError(t, err)
	})

	go accessor(func() {
		err := instance2.updateProcessors()
		assert.NoError(t, err)
	})

	time.Sleep(100 * time.Millisecond)

	instance1.shutdown()
	instance2.shutdown()

	time.Sleep(10 * time.Millisecond)
	close(end)
}

func TestInstancePushSpansSkipProcessors(t *testing.T) {
	overrides := &mockOverrides{}
	overrides.processors = map[string]struct{}{
		processor.SpanMetricsName:   {},
		processor.ServiceGraphsName: {},
	}
	const tenantID = "skip-processors-test"

	cfg := &Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	i, err := newInstance(cfg, tenantID, overrides, &noopStorage{}, log.NewNopLogger())
	require.NoError(t, err)

	req := test.MakeBatch(1, nil)

	// Expose this series so it's present at the initial zero value even if not created/incremented by the test.
	_ = metricSkippedProcessorPushes.WithLabelValues(tenantID)

	t.Run("use metrics-generating processors", func(t *testing.T) {
		i.pushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*v1.ResourceSpans{req}})

		expectMetrics := `
# HELP tempo_metrics_generator_metrics_generation_skipped_processor_pushes_total The total number of processor pushes skipped because the request indicated that metrics should not be generated.
# TYPE tempo_metrics_generator_metrics_generation_skipped_processor_pushes_total counter
tempo_metrics_generator_metrics_generation_skipped_processor_pushes_total{tenant="skip-processors-test"} 0
`
		err := testutil.GatherAndCompare(
			prometheus.DefaultGatherer,
			strings.NewReader(expectMetrics),
			"tempo_metrics_generator_metrics_generation_skipped_processor_pushes_total",
		)
		require.NoError(t, err)
	})

	t.Run("skip metrics-generating processors", func(t *testing.T) {
		i.pushSpans(context.Background(), &tempopb.PushSpansRequest{Batches: []*v1.ResourceSpans{req}, SkipMetricsGeneration: true})

		expectMetrics := `
# HELP tempo_metrics_generator_metrics_generation_skipped_processor_pushes_total The total number of processor pushes skipped because the request indicated that metrics should not be generated.
# TYPE tempo_metrics_generator_metrics_generation_skipped_processor_pushes_total counter
tempo_metrics_generator_metrics_generation_skipped_processor_pushes_total{tenant="skip-processors-test"} 2
`
		err := testutil.GatherAndCompare(
			prometheus.DefaultGatherer,
			strings.NewReader(expectMetrics),
			"tempo_metrics_generator_metrics_generation_skipped_processor_pushes_total",
		)
		require.NoError(t, err)
	})
}

// Test_instance_preprocessSpans covers the in-place ingestion-slack filtering:
// spans outside the slack window are dropped by compacting each span slice in
// place, the stale tail is nil-cleared so dropped spans are not retained, and
// the discarded-spans counter is incremented.
func Test_instance_preprocessSpans(t *testing.T) {
	const tenantID = "preprocess-spans-test"

	inst := &instance{instanceID: tenantID}
	inst.ingestionSlackOverride.Store((30 * time.Second).Nanoseconds())

	makeSpan := func(name string, end time.Time) *v1.Span {
		return &v1.Span{Name: name, EndTimeUnixNano: uint64(end.UnixNano())}
	}

	now := time.Now()
	keepA1 := makeSpan("keep-a1", now)
	dropA2 := makeSpan("drop-a2", now.Add(-time.Hour)) // too far in the past
	keepA3 := makeSpan("keep-a3", now)
	dropA4 := makeSpan("drop-a4", now.Add(time.Hour)) // too far in the future
	dropB1 := makeSpan("drop-b1", now.Add(-time.Hour))
	dropB2 := makeSpan("drop-b2", now.Add(time.Hour))
	keepC1 := makeSpan("keep-c1", now)
	keepC2 := makeSpan("keep-c2", now)

	scopeMixed := &v1.ScopeSpans{Spans: []*v1.Span{keepA1, dropA2, keepA3, dropA4}}
	scopeAllDropped := &v1.ScopeSpans{Spans: []*v1.Span{dropB1, dropB2}}
	scopeAllKept := &v1.ScopeSpans{Spans: []*v1.Span{keepC1, keepC2}}
	scopeEmpty := &v1.ScopeSpans{}

	req := &tempopb.PushSpansRequest{Batches: []*v1.ResourceSpans{
		{ScopeSpans: []*v1.ScopeSpans{scopeMixed, scopeAllDropped}},
		{ScopeSpans: []*v1.ScopeSpans{scopeAllKept, scopeEmpty}},
	}}

	// Keep the original slice headers to verify in-place compaction and the
	// nil-cleared tails afterwards.
	originalMixed := scopeMixed.Spans
	originalAllDropped := scopeAllDropped.Spans

	discarded := metricSpansDiscarded.WithLabelValues(tenantID, reasonOutsideTimeRangeSlack, "all")
	received := metricSpansIngested.WithLabelValues(tenantID)
	discardedBefore := testutil.ToFloat64(discarded)
	receivedBefore := testutil.ToFloat64(received)

	inst.preprocessSpans(req)

	// Mixed scope: survivors compacted in place, order preserved, tail nil-cleared.
	require.Len(t, scopeMixed.Spans, 2)
	require.Same(t, keepA1, scopeMixed.Spans[0])
	require.Same(t, keepA3, scopeMixed.Spans[1])
	require.Same(t, &originalMixed[0], &scopeMixed.Spans[0], "filtering must reuse the original backing array")
	require.Nil(t, originalMixed[2], "dropped tail must be nil-cleared to release span pointers")
	require.Nil(t, originalMixed[3], "dropped tail must be nil-cleared to release span pointers")

	// All dropped: empty result, fully nil-cleared backing array.
	require.Empty(t, scopeAllDropped.Spans)
	require.Nil(t, originalAllDropped[0])
	require.Nil(t, originalAllDropped[1])

	// All kept: untouched.
	require.Len(t, scopeAllKept.Spans, 2)
	require.Same(t, keepC1, scopeAllKept.Spans[0])
	require.Same(t, keepC2, scopeAllKept.Spans[1])

	require.Equal(t, float64(4), testutil.ToFloat64(discarded)-discardedBefore, "4 spans outside the slack window")
	require.Equal(t, float64(8), testutil.ToFloat64(received)-receivedBefore, "all 8 spans count as received")

	// Re-pushing the already-filtered request drops nothing further.
	inst.preprocessSpans(req)
	require.Len(t, scopeMixed.Spans, 2)
	require.Empty(t, scopeAllDropped.Spans)
	require.Len(t, scopeAllKept.Spans, 2)
	require.Equal(t, float64(4), testutil.ToFloat64(discarded)-discardedBefore)

	// An empty request is a no-op.
	require.NotPanics(t, func() { inst.preprocessSpans(&tempopb.PushSpansRequest{}) })
}

func Test_instance_updateProcessors(t *testing.T) {
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	overrides := mockOverrides{}

	instance, err := newInstance(&cfg, "test", &overrides, &noopStorage{}, logger)
	assert.NoError(t, err)

	// stop the update goroutine
	close(instance.shutdownCh)

	// no processors should be present initially
	assert.Len(t, instance.processors, 0)

	t.Run("add servicegraphs processors", func(t *testing.T) {
		overrides.processors = map[string]struct{}{
			processor.ServiceGraphsName: {},
		}
		err := instance.updateProcessors()
		assert.NoError(t, err)

		assert.Len(t, instance.processors, 1)
		assert.Equal(t, instance.processors[processor.ServiceGraphsName].Name(), processor.ServiceGraphsName)
	})

	t.Run("ignore unknown processor", func(t *testing.T) {
		overrides.processors = map[string]struct{}{
			"span-metricsss": {}, // typo in the overrides
		}
		err := instance.updateProcessors()
		assert.NoError(t, err)

		// unknown processors are ignored and therefore the desired processor set is empty.
		assert.Len(t, instance.processors, 0)
	})

	t.Run("ignore removed local-blocks processor and keep valid processors", func(t *testing.T) {
		overrides.processors = map[string]struct{}{
			"local-blocks":              {},
			processor.ServiceGraphsName: {},
		}
		err := instance.updateProcessors()
		assert.NoError(t, err)

		assert.Len(t, instance.processors, 1)
		assert.Equal(t, processor.ServiceGraphsName, instance.processors[processor.ServiceGraphsName].Name())
	})

	t.Run("add spanmetrics processor", func(t *testing.T) {
		overrides.processors = map[string]struct{}{
			processor.ServiceGraphsName: {},
			processor.SpanMetricsName:   {},
		}
		err := instance.updateProcessors()
		assert.NoError(t, err)

		assert.Len(t, instance.processors, 2)
		assert.Equal(t, instance.processors[processor.ServiceGraphsName].Name(), processor.ServiceGraphsName)
		assert.Equal(t, instance.processors[processor.SpanMetricsName].Name(), processor.SpanMetricsName)
	})

	t.Run("replace spanmetrics processor", func(t *testing.T) {
		overrides.processors = map[string]struct{}{
			processor.ServiceGraphsName: {},
			processor.SpanMetricsName:   {},
		}
		overrides.spanMetricsDimensions = []string{"namespace"}
		overrides.spanMetricsIntrinsicDimensions = map[string]bool{"status_message": true}

		err := instance.updateProcessors()
		assert.NoError(t, err)

		var expectedConfig spanmetrics.Config
		expectedConfig.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
		expectedConfig.Dimensions = []string{"namespace"}
		expectedConfig.IntrinsicDimensions.StatusMessage = true

		assert.Equal(t, expectedConfig, instance.processors[processor.SpanMetricsName].(*spanmetrics.Processor).Cfg)
	})

	t.Run("add hostinfo processor", func(t *testing.T) {
		overrides.processors = map[string]struct{}{
			processor.ServiceGraphsName: {},
			processor.SpanMetricsName:   {},
			processor.HostInfoName:      {},
		}
		err := instance.updateProcessors()
		assert.NoError(t, err)

		assert.Len(t, instance.processors, 3)
		assert.Equal(t, instance.processors[processor.ServiceGraphsName].Name(), processor.ServiceGraphsName)
		assert.Equal(t, instance.processors[processor.SpanMetricsName].Name(), processor.SpanMetricsName)
		assert.Equal(t, instance.processors[processor.HostInfoName].Name(), processor.HostInfoName)
	})

	t.Run("replace hostinfo processor", func(t *testing.T) {
		overrides.processors = map[string]struct{}{
			processor.ServiceGraphsName: {},
			processor.SpanMetricsName:   {},
			processor.HostInfoName:      {},
		}
		overrides.hostInfoHostIdentifiers = []string{"host.id"}

		overrides.hostInfoMetricName = "sample_traces_host_info"
		err := instance.updateProcessors()
		assert.NoError(t, err)

		var expectedConfig hostinfo.Config
		expectedConfig.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
		expectedConfig.HostIdentifiers = []string{"host.id"}
		expectedConfig.MetricName = "sample_traces_host_info"

		assert.Equal(t, expectedConfig, instance.processors[processor.HostInfoName].(*hostinfo.Processor).Cfg)
	})

	t.Run("remove processor", func(t *testing.T) {
		overrides.processors = nil
		err := instance.updateProcessors()
		assert.NoError(t, err)

		assert.Len(t, instance.processors, 0)
	})

	t.Run("add span-latency subprocessor", func(t *testing.T) {
		overrides.processors = map[string]struct{}{
			processor.ServiceGraphsName:  {},
			spanmetrics.Latency.String(): {},
		}
		err := instance.updateProcessors()
		assert.NoError(t, err)

		var expectedConfig spanmetrics.Config
		expectedConfig.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
		expectedConfig.Dimensions = []string{"namespace"}
		expectedConfig.IntrinsicDimensions.StatusMessage = true
		expectedConfig.HistogramBuckets = prometheus.ExponentialBuckets(0.002, 2, 14)
		expectedConfig.Subprocessors[spanmetrics.Latency] = true
		expectedConfig.Subprocessors[spanmetrics.Count] = false
		expectedConfig.Subprocessors[spanmetrics.Size] = false

		assert.Equal(t, expectedConfig, instance.processors[processor.SpanMetricsName].(*spanmetrics.Processor).Cfg)

		expectedProcessors := []string{processor.ServiceGraphsName, processor.SpanMetricsName}
		actualProcessors := make([]string, 0, len(instance.processors))

		for name := range instance.processors {
			actualProcessors = append(actualProcessors, name)
		}

		sort.Strings(actualProcessors)

		assert.Equal(t, expectedProcessors, actualProcessors)
	})

	t.Run("replace span-latency subprocessor with span-count", func(t *testing.T) {
		overrides.processors = map[string]struct{}{
			processor.ServiceGraphsName: {},
			spanmetrics.Count.String():  {},
		}
		err := instance.updateProcessors()
		assert.NoError(t, err)

		var expectedConfig spanmetrics.Config
		expectedConfig.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
		expectedConfig.Dimensions = []string{"namespace"}
		expectedConfig.IntrinsicDimensions.StatusMessage = true
		expectedConfig.HistogramBuckets = nil
		expectedConfig.Subprocessors[spanmetrics.Latency] = false
		expectedConfig.Subprocessors[spanmetrics.Count] = true
		expectedConfig.Subprocessors[spanmetrics.Size] = false

		assert.Equal(t, expectedConfig, instance.processors[processor.SpanMetricsName].(*spanmetrics.Processor).Cfg)

		expectedProcessors := []string{processor.ServiceGraphsName, processor.SpanMetricsName}
		actualProcessors := make([]string, 0, len(instance.processors))

		for name := range instance.processors {
			actualProcessors = append(actualProcessors, name)
		}

		sort.Strings(actualProcessors)

		assert.Equal(t, expectedProcessors, actualProcessors)
	})

	t.Run("use all three subprocessors at once", func(t *testing.T) {
		overrides.processors = map[string]struct{}{
			processor.ServiceGraphsName:  {},
			spanmetrics.Count.String():   {},
			spanmetrics.Latency.String(): {},
			spanmetrics.Size.String():    {},
		}
		err := instance.updateProcessors()
		assert.NoError(t, err)

		var expectedConfig spanmetrics.Config
		expectedConfig.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
		expectedConfig.Dimensions = []string{"namespace"}
		expectedConfig.IntrinsicDimensions.StatusMessage = true
		expectedConfig.HistogramBuckets = prometheus.ExponentialBuckets(0.002, 2, 14)
		expectedConfig.Subprocessors[spanmetrics.Latency] = true
		expectedConfig.Subprocessors[spanmetrics.Count] = true
		expectedConfig.Subprocessors[spanmetrics.Size] = true

		assert.Equal(t, expectedConfig, instance.processors[processor.SpanMetricsName].(*spanmetrics.Processor).Cfg)

		expectedProcessors := []string{processor.ServiceGraphsName, processor.SpanMetricsName}
		actualProcessors := make([]string, 0, len(instance.processors))

		for name := range instance.processors {
			actualProcessors = append(actualProcessors, name)
		}

		sort.Strings(actualProcessors)

		assert.Equal(t, expectedProcessors, actualProcessors)
	})

	t.Run("replace subprocessors with span-metrics processor", func(t *testing.T) {
		overrides.processors = map[string]struct{}{
			processor.ServiceGraphsName: {},
			processor.SpanMetricsName:   {},
		}
		err := instance.updateProcessors()
		assert.NoError(t, err)

		var expectedConfig spanmetrics.Config
		expectedConfig.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
		expectedConfig.Dimensions = []string{"namespace"}
		expectedConfig.IntrinsicDimensions.StatusMessage = true
		expectedConfig.HistogramBuckets = prometheus.ExponentialBuckets(0.002, 2, 14)
		expectedConfig.Subprocessors[spanmetrics.Latency] = true
		expectedConfig.Subprocessors[spanmetrics.Count] = true

		assert.Equal(t, expectedConfig, instance.processors[processor.SpanMetricsName].(*spanmetrics.Processor).Cfg)

		expectedProcessors := []string{processor.ServiceGraphsName, processor.SpanMetricsName}
		actualProcessors := make([]string, 0, len(instance.processors))

		for name := range instance.processors {
			actualProcessors = append(actualProcessors, name)
		}

		sort.Strings(actualProcessors)

		assert.Equal(t, expectedProcessors, actualProcessors)
	})

	t.Run("replace span-metrics and servicegraphs processors when histograms impementation changes", func(t *testing.T) {
		overrides.nativeHistograms = "native"
		overrides.processors = map[string]struct{}{
			processor.ServiceGraphsName: {},
			processor.SpanMetricsName:   {},
		}
		err := instance.updateProcessors()
		assert.NoError(t, err)

		assertHistogramsReload := func(t *testing.T) {
			desiredProcessors := instance.overrides.MetricsGeneratorProcessors(instance.instanceID)
			desiredCfg, copyErr := instance.cfg.Processor.copyWithOverrides(instance.overrides, instance.instanceID)
			assert.NoError(t, copyErr)
			toAdd, toRemove, toReplace, diffErr := instance.diffProcessors(desiredProcessors, desiredCfg)
			assert.NoError(t, diffErr)
			assert.Empty(t, toAdd)
			assert.Empty(t, toRemove)

			sort.Strings(toReplace)
			assert.Equal(t, []string{processor.ServiceGraphsName, processor.SpanMetricsName}, toReplace)
		}

		assertHistogramsNoChange := func(t *testing.T) {
			desiredProcessors := instance.overrides.MetricsGeneratorProcessors(instance.instanceID)
			desiredCfg, copyErr := instance.cfg.Processor.copyWithOverrides(instance.overrides, instance.instanceID)
			assert.NoError(t, copyErr)
			toAdd, toRemove, toReplace, diffErr := instance.diffProcessors(desiredProcessors, desiredCfg)
			assert.NoError(t, diffErr)
			assert.Empty(t, toAdd)
			assert.Empty(t, toRemove)
			assert.Empty(t, toReplace)
		}

		// Downgrade to classic
		overrides.nativeHistograms = "classic"
		assertHistogramsReload(t)

		err = instance.updateProcessors()
		assert.NoError(t, err)
		assertHistogramsNoChange(t)

		// Upgrade to both native and classic
		overrides.nativeHistograms = "both"
		assertHistogramsReload(t)

		err = instance.updateProcessors()
		assert.NoError(t, err)
		assertHistogramsNoChange(t)

		// Upgrade back to native
		overrides.nativeHistograms = "native"
		assertHistogramsReload(t)

		err = instance.updateProcessors()
		assert.NoError(t, err)
		assertHistogramsNoChange(t)
	})

	t.Run("add service-graphs-connection-info subprocessor only", func(t *testing.T) {
		overrides.processors = map[string]struct{}{
			servicegraphs.ConnectionInfo.String(): {},
		}
		err := instance.updateProcessors()
		require.NoError(t, err)

		// service-graphs sub-name should be folded into the bare service-graphs processor entry.
		_, hasBare := instance.processors[processor.ServiceGraphsName]
		require.True(t, hasBare, "expected bare service-graphs processor to be registered")
		_, hasSubName := instance.processors[processor.ServiceGraphsConnectionInfoName]
		require.False(t, hasSubName, "sub-name should not appear in the processors map")

		cfg := instance.processors[processor.ServiceGraphsName].(*servicegraphs.Processor).Cfg
		require.False(t, cfg.Subprocessors[servicegraphs.Request], "Request should be disabled when only connection-info requested")
		require.False(t, cfg.Subprocessors[servicegraphs.Latency], "Latency should be disabled when only connection-info requested")
		require.True(t, cfg.Subprocessors[servicegraphs.ConnectionInfo], "ConnectionInfo should be enabled")
	})

	t.Run("bare service-graphs keeps connection-info opt-in", func(t *testing.T) {
		overrides.processors = map[string]struct{}{
			processor.ServiceGraphsName: {},
		}
		err := instance.updateProcessors()
		require.NoError(t, err)

		cfg := instance.processors[processor.ServiceGraphsName].(*servicegraphs.Processor).Cfg
		require.True(t, cfg.Subprocessors[servicegraphs.Request])
		require.True(t, cfg.Subprocessors[servicegraphs.Latency])
		require.False(t, cfg.Subprocessors[servicegraphs.ConnectionInfo], "ConnectionInfo must remain off under bare service-graphs")
	})

	t.Run("service-graphs + connection-info sub-name combines RED and connection_info", func(t *testing.T) {
		overrides.processors = map[string]struct{}{
			processor.ServiceGraphsName:           {},
			servicegraphs.ConnectionInfo.String(): {},
		}
		err := instance.updateProcessors()
		require.NoError(t, err)

		cfg := instance.processors[processor.ServiceGraphsName].(*servicegraphs.Processor).Cfg
		require.True(t, cfg.Subprocessors[servicegraphs.Request])
		require.True(t, cfg.Subprocessors[servicegraphs.Latency])
		require.True(t, cfg.Subprocessors[servicegraphs.ConnectionInfo])
	})

	t.Run("service-graphs-request subprocessor only", func(t *testing.T) {
		overrides.processors = map[string]struct{}{
			servicegraphs.Request.String(): {},
		}
		err := instance.updateProcessors()
		require.NoError(t, err)

		cfg := instance.processors[processor.ServiceGraphsName].(*servicegraphs.Processor).Cfg
		require.True(t, cfg.Subprocessors[servicegraphs.Request])
		require.False(t, cfg.Subprocessors[servicegraphs.Latency])
		require.False(t, cfg.Subprocessors[servicegraphs.ConnectionInfo])

		_, hasSubName := instance.processors[processor.ServiceGraphsRequestName]
		require.False(t, hasSubName, "sub-name should be folded into the bare service-graphs entry")
	})

	t.Run("service-graphs-latency subprocessor only", func(t *testing.T) {
		overrides.processors = map[string]struct{}{
			servicegraphs.Latency.String(): {},
		}
		err := instance.updateProcessors()
		require.NoError(t, err)

		cfg := instance.processors[processor.ServiceGraphsName].(*servicegraphs.Processor).Cfg
		require.False(t, cfg.Subprocessors[servicegraphs.Request])
		require.True(t, cfg.Subprocessors[servicegraphs.Latency])
		require.False(t, cfg.Subprocessors[servicegraphs.ConnectionInfo])
	})

	t.Run("transition between service-graphs subprocessor subsets triggers replace", func(t *testing.T) {
		// Start with bare service-graphs (Request + Latency).
		overrides.processors = map[string]struct{}{
			processor.ServiceGraphsName: {},
		}
		err := instance.updateProcessors()
		require.NoError(t, err)

		// Now request connection-info only. diffProcessors should mark the service-graphs
		// processor for replacement (config changed) rather than removal.
		overrides.processors = map[string]struct{}{
			servicegraphs.ConnectionInfo.String(): {},
		}

		desiredProcessors := instance.filterSupportedProcessors(instance.overrides.MetricsGeneratorProcessors(instance.instanceID))
		desiredCfg, err := instance.cfg.Processor.copyWithOverrides(instance.overrides, instance.instanceID)
		require.NoError(t, err)
		desiredProcessors, desiredCfg = instance.updateSubprocessors(desiredProcessors, desiredCfg)

		toAdd, toRemove, toReplace, err := instance.diffProcessors(desiredProcessors, desiredCfg)
		require.NoError(t, err)
		require.Empty(t, toAdd, "service-graphs already exists; no add")
		require.Empty(t, toRemove, "service-graphs should not be removed; sub-name folds into it")
		require.Equal(t, []string{processor.ServiceGraphsName}, toReplace, "config diff (Request+Latency -> ConnectionInfo only) must trigger replace")

		err = instance.updateProcessors()
		require.NoError(t, err)

		cfg := instance.processors[processor.ServiceGraphsName].(*servicegraphs.Processor).Cfg
		require.False(t, cfg.Subprocessors[servicegraphs.Request])
		require.False(t, cfg.Subprocessors[servicegraphs.Latency])
		require.True(t, cfg.Subprocessors[servicegraphs.ConnectionInfo])
	})
}

type noopStorage struct{}

var _ storage.Storage = (*noopStorage)(nil)

func (m noopStorage) Appender(context.Context) prometheus_storage.Appender {
	return &noopAppender{}
}

func (m noopStorage) Close() error { return nil }

type noopAppender struct{}

var _ prometheus_storage.Appender = (*noopAppender)(nil)

func (n noopAppender) Append(prometheus_storage.SeriesRef, labels.Labels, int64, float64) (prometheus_storage.SeriesRef, error) {
	return 0, nil
}

func (n noopAppender) AppendExemplar(prometheus_storage.SeriesRef, labels.Labels, exemplar.Exemplar) (prometheus_storage.SeriesRef, error) {
	return 0, nil
}

func (n noopAppender) AppendHistogram(prometheus_storage.SeriesRef, labels.Labels, int64, *histogram.Histogram, *histogram.FloatHistogram) (prometheus_storage.SeriesRef, error) {
	return 0, nil
}

func (n noopAppender) Commit() error { return nil }

func (n noopAppender) Rollback() error                                { return nil }
func (n noopAppender) SetOptions(_ *prometheus_storage.AppendOptions) {}

func (n noopAppender) UpdateMetadata(prometheus_storage.SeriesRef, labels.Labels, metadata.Metadata) (prometheus_storage.SeriesRef, error) {
	return 0, nil
}

func (n noopAppender) AppendCTZeroSample(_ prometheus_storage.SeriesRef, _ labels.Labels, _, _ int64) (prometheus_storage.SeriesRef, error) {
	return 0, nil
}

func (n *noopAppender) AppendHistogramCTZeroSample(_ prometheus_storage.SeriesRef, _ labels.Labels, _, _ int64, _ *histogram.Histogram, _ *histogram.FloatHistogram) (prometheus_storage.SeriesRef, error) {
	return 0, nil
}

func (n noopAppender) AppendSTZeroSample(_ prometheus_storage.SeriesRef, _ labels.Labels, _, _ int64) (prometheus_storage.SeriesRef, error) {
	return 0, nil
}

func (n *noopAppender) AppendHistogramSTZeroSample(_ prometheus_storage.SeriesRef, _ labels.Labels, _, _ int64, _ *histogram.Histogram, _ *histogram.FloatHistogram) (prometheus_storage.SeriesRef, error) {
	return 0, nil
}
