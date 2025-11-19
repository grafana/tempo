package registry

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/modules/overrides/histograms"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockLimiter struct {
	onAddFunc    func(labelHash uint64, seriesCount uint32) bool
	onUpdateFunc func(labelHash uint64, seriesCount uint32)
	onDeleteFunc func(labelHash uint64, seriesCount uint32)
}

var noopLimiter Limiter = &mockLimiter{}

var _ Limiter = (*mockLimiter)(nil)

func (m *mockLimiter) OnAdd(labelHash uint64, seriesCount uint32) bool {
	if m.onAddFunc == nil {
		return true
	}
	return m.onAddFunc(labelHash, seriesCount)
}

func (m *mockLimiter) OnUpdate(labelHash uint64, seriesCount uint32) {
	if m.onUpdateFunc == nil {
		return
	}
	m.onUpdateFunc(labelHash, seriesCount)
}

func (m *mockLimiter) OnDelete(labelHash uint64, seriesCount uint32) {
	if m.onDeleteFunc == nil {
		return
	}
	m.onDeleteFunc(labelHash, seriesCount)
}

func buildTestLabels(names []string, values []string) labels.Labels {
	builder := NewLabelBuilder(0, 0)
	for i := range names {
		builder.Add(names[i], values[i])
	}
	lbls, _ := builder.CloseAndBuildLabels()
	return lbls
}

// TODO: rewrite tests to use mocked limiter instead of this
func (r *ManagedRegistry) activeSeries() uint32 {
	r.metricsMtx.RLock()
	defer r.metricsMtx.RUnlock()
	output := uint32(0)
	for _, m := range r.metrics {
		output += uint32(m.countActiveSeries())
	}
	return output
}

func TestManagedRegistry_concurrency(*testing.T) {
	cfg := &Config{
		StaleDuration: 1 * time.Millisecond,
	}
	registry := New(cfg, &mockOverrides{}, "test", &noopAppender{}, log.NewNopLogger(), noopLimiter)
	defer registry.Close()

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

	// this goroutine constantly creates new counters
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	go accessor(func() {
		s := make([]rune, 6)
		for i := range s {
			s[i] = letters[rand.Intn(len(letters))]
		}
		registry.NewCounter(string(s))
	})

	go accessor(func() {
		registry.CollectMetrics(context.Background())
	})

	go accessor(func() {
		registry.removeStaleSeries(context.Background())
	})

	time.Sleep(200 * time.Millisecond)
	close(end)
}

func TestManagedRegistry_counter(t *testing.T) {
	appender := &capturingAppender{}

	registry := New(&Config{}, &mockOverrides{}, "test", appender, log.NewNopLogger(), noopLimiter)
	defer registry.Close()

	counter := registry.NewCounter("my_counter")

	counter.Inc(buildTestLabels([]string{"label", "label"}, []string{"repeated-value", "value-1"}), 1.0)

	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1", "__metrics_gen_instance": mustGetHostname()}, 0, 0.0),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1", "__metrics_gen_instance": mustGetHostname()}, 0, 1.0),
	}
	collectRegistryMetricsAndAssert(t, registry, appender, expectedSamples)
}

func TestManagedRegistry_histogram(t *testing.T) {
	appender := &capturingAppender{}

	registry := New(&Config{}, &mockOverrides{}, "test", appender, log.NewNopLogger(), noopLimiter)
	defer registry.Close()

	histogram := registry.NewHistogram("histogram", []float64{1.0, 2.0}, HistogramModeClassic)

	histogram.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.0, "", 1.0)

	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "histogram_count", "label": "value-1", "__metrics_gen_instance": mustGetHostname()}, 0, 0),
		newSample(map[string]string{"__name__": "histogram_count", "label": "value-1", "__metrics_gen_instance": mustGetHostname()}, 1, 1.0),
		newSample(map[string]string{"__name__": "histogram_sum", "label": "value-1", "__metrics_gen_instance": mustGetHostname()}, 1, 1.0),
		newSample(map[string]string{"__name__": "histogram_bucket", "label": "value-1", "__metrics_gen_instance": mustGetHostname(), "le": "1"}, 0, 0),
		newSample(map[string]string{"__name__": "histogram_bucket", "label": "value-1", "__metrics_gen_instance": mustGetHostname(), "le": "1"}, 1, 1.0),
		newSample(map[string]string{"__name__": "histogram_bucket", "label": "value-1", "__metrics_gen_instance": mustGetHostname(), "le": "2"}, 0, 0),
		newSample(map[string]string{"__name__": "histogram_bucket", "label": "value-1", "__metrics_gen_instance": mustGetHostname(), "le": "2"}, 1, 1.0),
		newSample(map[string]string{"__name__": "histogram_bucket", "label": "value-1", "__metrics_gen_instance": mustGetHostname(), "le": "+Inf"}, 0, 0),
		newSample(map[string]string{"__name__": "histogram_bucket", "label": "value-1", "__metrics_gen_instance": mustGetHostname(), "le": "+Inf"}, 1, 1.0),
	}
	collectRegistryMetricsAndAssert(t, registry, appender, expectedSamples)
}

func TestManagedRegistry_removeStaleSeries(t *testing.T) {
	appender := &capturingAppender{}

	cfg := &Config{
		StaleDuration: 75 * time.Millisecond,
	}
	registry := New(cfg, &mockOverrides{}, "test", appender, log.NewNopLogger(), noopLimiter)
	defer registry.Close()

	counter1 := registry.NewCounter("metric_1")
	counter2 := registry.NewCounter("metric_2")

	counter1.Inc(labels.New(), 1)
	counter2.Inc(labels.New(), 2)

	registry.removeStaleSeries(context.Background())

	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "metric_1", "__metrics_gen_instance": mustGetHostname()}, 0, 0),
		newSample(map[string]string{"__name__": "metric_1", "__metrics_gen_instance": mustGetHostname()}, 0, 1),
		newSample(map[string]string{"__name__": "metric_2", "__metrics_gen_instance": mustGetHostname()}, 0, 0),
		newSample(map[string]string{"__name__": "metric_2", "__metrics_gen_instance": mustGetHostname()}, 0, 2),
	}
	collectRegistryMetricsAndAssert(t, registry, appender, expectedSamples)

	appender.samples = nil

	time.Sleep(50 * time.Millisecond)
	counter2.Inc(labels.New(), 2)
	time.Sleep(50 * time.Millisecond)

	registry.removeStaleSeries(context.Background())

	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "metric_2", "__metrics_gen_instance": mustGetHostname()}, 0, 4),
	}
	collectRegistryMetricsAndAssert(t, registry, appender, expectedSamples)
}

func TestManagedRegistry_externalLabels(t *testing.T) {
	appender := &capturingAppender{}

	cfg := &Config{
		ExternalLabels: map[string]string{
			"__foo": "bar",
		},
	}
	registry := New(cfg, &mockOverrides{}, "test", appender, log.NewNopLogger(), noopLimiter)
	defer registry.Close()

	counter := registry.NewCounter("my_counter")
	counter.Inc(labels.New(), 1.0)

	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_counter", "__metrics_gen_instance": mustGetHostname(), "__foo": "bar"}, 0, 0),
		newSample(map[string]string{"__name__": "my_counter", "__metrics_gen_instance": mustGetHostname(), "__foo": "bar"}, 0, 1),
	}
	collectRegistryMetricsAndAssert(t, registry, appender, expectedSamples)
}

func TestManagedRegistry_injectTenantIDAs(t *testing.T) {
	appender := &capturingAppender{}

	cfg := &Config{
		InjectTenantIDAs: "__tempo_tenant",
	}
	registry := New(cfg, &mockOverrides{}, "test", appender, log.NewNopLogger(), noopLimiter)
	defer registry.Close()

	counter := registry.NewCounter("my_counter")
	counter.Inc(labels.New(), 1.0)

	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_counter", "__metrics_gen_instance": mustGetHostname(), "__tempo_tenant": "test"}, 0, 0),
		newSample(map[string]string{"__name__": "my_counter", "__metrics_gen_instance": mustGetHostname(), "__tempo_tenant": "test"}, 0, 1),
	}
	collectRegistryMetricsAndAssert(t, registry, appender, expectedSamples)
}

func TestManagedRegistry_limited(t *testing.T) {
	appender := &capturingAppender{}

	overrides := &mockOverrides{}
	atLimit := false
	limiter := &mockLimiter{
		onAddFunc: func(uint64, uint32) bool {
			return !atLimit
		},
	}
	registry := New(&Config{}, overrides, "test", appender, log.NewNopLogger(), limiter)
	defer registry.Close()

	counter1 := registry.NewCounter("metric_1")
	counter2 := registry.NewCounter("metric_2")

	counter1.Inc(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.0)
	atLimit = true
	// these series should be discarded
	counter1.Inc(buildTestLabels([]string{"label"}, []string{"value-2"}), 1.0)
	counter2.Inc(labels.New(), 1.0)

	assert.Equal(t, uint32(1), registry.activeSeries())
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "metric_1", "label": "value-1", "__metrics_gen_instance": mustGetHostname()}, 0, 0),
		newSample(map[string]string{"__name__": "metric_1", "label": "value-1", "__metrics_gen_instance": mustGetHostname()}, 0, 1),
	}
	collectRegistryMetricsAndAssert(t, registry, appender, expectedSamples)
}

func TestManagedRegistry_maxEntities(t *testing.T) {
	appender := &capturingAppender{}

	atLimit := false
	limiter := &mockLimiter{
		onAddFunc: func(uint64, uint32) bool {
			return !atLimit
		},
	}
	overrides := &mockOverrides{}
	registry := New(&Config{}, overrides, "test", appender, log.NewNopLogger(), limiter)
	defer registry.Close()

	counter1 := registry.NewCounter("metric_1")
	counter2 := registry.NewCounter("metric_2")

	entity1 := buildTestLabels([]string{"label"}, []string{"value-1"})
	entity2 := buildTestLabels([]string{"label"}, []string{"value-2"})
	counter1.Inc(entity1, 1.0)
	counter2.Inc(entity1, 1.0)
	atLimit = true
	counter1.Inc(entity2, 1.0)
	counter2.Inc(entity2, 1.0)

	// At this point, we will have only series for entity1
	assert.Equal(t, uint32(2), registry.activeSeries())

	// The specific series which are discarded is not guaranteed, but it should be consistent within a single entity.
	entityCount := collectRegistryMetricsAndCountEntities(registry, appender)

	// After collection, the series should be removed
	assert.Equal(t, 1, entityCount)
	assert.Equal(t, uint32(2), registry.activeSeries())
}

func TestManagedRegistry_disableCollection(t *testing.T) {
	appender := &capturingAppender{}

	overrides := &mockOverrides{
		disableCollection: true,
	}
	registry := New(&Config{}, overrides, "test", appender, log.NewNopLogger(), noopLimiter)
	defer registry.Close()

	counter := registry.NewCounter("metric_1")
	counter.Inc(labels.New(), 1.0)

	// active series are still tracked
	assert.Equal(t, uint32(1), registry.activeSeries())
	// but no samples are collected and sent out
	registry.CollectMetrics(context.Background())
	assert.Empty(t, appender.samples)
	assert.Empty(t, appender.exemplars)
}

func TestManagedRegistry_maxLabelNameLength(t *testing.T) {
	appender := &capturingAppender{}

	cfg := &Config{
		MaxLabelNameLength:  8,
		MaxLabelValueLength: 5,
	}
	registry := New(cfg, &mockOverrides{}, "test", appender, log.NewNopLogger(), noopLimiter)
	defer registry.Close()

	counter := registry.NewCounter("counter")
	histogram := registry.NewHistogram("histogram", []float64{1.0}, HistogramModeClassic)

	builder := registry.NewLabelBuilder()
	builder.Add("very_lengthy_label", "very_length_value")
	lbls, ok := builder.CloseAndBuildLabels()
	require.True(t, ok)
	counter.Inc(lbls, 1.0)
	builder = registry.NewLabelBuilder()
	builder.Add("another_very_lengthy_label", "another_very_lengthy_value")
	lbls, ok = builder.CloseAndBuildLabels()
	require.True(t, ok)
	histogram.ObserveWithExemplar(lbls, 1.0, "", 1.0)

	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "counter", "very_len": "very_", "__metrics_gen_instance": mustGetHostname()}, 0, 0.0),
		newSample(map[string]string{"__name__": "counter", "very_len": "very_", "__metrics_gen_instance": mustGetHostname()}, 1, 1.0),
		newSample(map[string]string{"__name__": "histogram_count", "another_": "anoth", "__metrics_gen_instance": mustGetHostname()}, 0, 0),
		newSample(map[string]string{"__name__": "histogram_count", "another_": "anoth", "__metrics_gen_instance": mustGetHostname()}, 1, 1.0),
		newSample(map[string]string{"__name__": "histogram_sum", "another_": "anoth", "__metrics_gen_instance": mustGetHostname()}, 1, 1.0),
		newSample(map[string]string{"__name__": "histogram_bucket", "another_": "anoth", "__metrics_gen_instance": mustGetHostname(), "le": "1"}, 0, 0),
		newSample(map[string]string{"__name__": "histogram_bucket", "another_": "anoth", "__metrics_gen_instance": mustGetHostname(), "le": "1"}, 1, 1.0),
		newSample(map[string]string{"__name__": "histogram_bucket", "another_": "anoth", "__metrics_gen_instance": mustGetHostname(), "le": "+Inf"}, 0, 0),
		newSample(map[string]string{"__name__": "histogram_bucket", "another_": "anoth", "__metrics_gen_instance": mustGetHostname(), "le": "+Inf"}, 1, 1.0),
	}
	collectRegistryMetricsAndAssert(t, registry, appender, expectedSamples)
}

func TestHistogramOverridesConfig(t *testing.T) {
	cases := []struct {
		name                string
		nativeHistogramMode HistogramMode
		typeOfHistogram     interface{}
	}{
		{
			"classic",
			HistogramModeClassic,
			&histogram{},
		},
		{
			"native",
			HistogramModeNative,
			&nativeHistogram{},
		},
		{
			"both",
			HistogramModeBoth,
			&nativeHistogram{},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			appender := &capturingAppender{}
			overrides := &mockOverrides{}
			registry := New(&Config{}, overrides, "test", appender, log.NewNopLogger(), noopLimiter)
			defer registry.Close()

			tt := registry.NewHistogram("histogram", []float64{1.0, 2.0}, c.nativeHistogramMode)
			require.IsType(t, c.typeOfHistogram, tt)
		})
	}
}

func collectRegistryMetricsAndCountEntities(r *ManagedRegistry, appender *capturingAppender) int {
	r.CollectMetrics(context.Background())

	entityMap := make(map[uint64]struct{})
	var hashBuf [1024]byte

	for _, sample := range appender.samples {
		hash, _ := sample.l.HashWithoutLabels(hashBuf[:0], "__name__")
		entityMap[hash] = struct{}{}
	}

	return len(entityMap)
}

func collectRegistryMetricsAndAssert(t *testing.T, r *ManagedRegistry, appender *capturingAppender, expectedSamples []sample) {
	collectionTimeMs := time.Now().UnixMilli()
	r.CollectMetrics(context.Background())

	// Validate that there are no duplicate label names in any sample
	for i, sample := range appender.samples {
		if duplicateLabel, hasDuplicate := sample.l.HasDuplicateLabelNames(); hasDuplicate {
			t.Errorf("Sample %d has duplicate label name %q. Full labels: %v",
				i, duplicateLabel, sample.l)
		}
	}

	// Ignore the collection time on expected samples, since we won't know when the collection will actually take place.
	for i := range expectedSamples {
		expectedSamples[i].t = collectionTimeMs
	}

	// Ignore the collection time on the collected samples.  Initial counter values will be offset from the collection time.
	for i := range appender.samples {
		appender.samples[i].t = collectionTimeMs
	}

	assert.Equal(t, true, appender.isCommitted)
	assert.Equal(t, false, appender.isRolledback)

	actualSamples := appender.samples
	require.Equal(t, len(expectedSamples), len(actualSamples))

	// Ensure that both slices are ordered consistently.
	for _, slice := range [][]sample{expectedSamples, actualSamples} {
		sort.Slice(slice, func(i, j int) bool {
			this := slice[i]
			next := slice[j]

			// The actual order doesn't matter, the only thing that matters is that it is consistent.
			return this.String() < next.String()
		})
	}

	for i, expected := range expectedSamples {
		actual := actualSamples[i]

		assert.Equal(t, expected.t, actual.t)
		assert.Equal(t, expected.v, actual.v)
		// Rely on the fact that Go prints map keys in sorted order.
		// See https://tip.golang.org/doc/go1.12#fmt.
		assert.Equal(t, fmt.Sprint(expected.l.Map()), fmt.Sprint(actual.l.Map()))
	}
}

type mockOverrides struct {
	maxActiveSeries                 uint32
	maxActiveEntities               uint32
	disableCollection               bool
	generateNativeHistograms        histograms.HistogramMethod
	nativeHistogramMaxBucketNumber  uint32
	nativeHistogramBucketFactor     float64
	nativeHistogramMinResetDuration time.Duration
}

var _ Overrides = (*mockOverrides)(nil)

func (m *mockOverrides) MetricsGeneratorMaxActiveSeries(string) uint32 {
	return m.maxActiveSeries
}

func (m *mockOverrides) MetricsGeneratorMaxActiveEntities(string) uint32 {
	return m.maxActiveEntities
}

func (m *mockOverrides) MetricsGeneratorCollectionInterval(string) time.Duration {
	return 15 * time.Second
}

func (m *mockOverrides) MetricsGeneratorDisableCollection(string) bool {
	return m.disableCollection
}

func (m *mockOverrides) MetricsGeneratorGenerateNativeHistograms(_ string) histograms.HistogramMethod {
	return m.generateNativeHistograms
}

func (m *mockOverrides) MetricsGeneratorTraceIDLabelName(string) string {
	return ""
}

func (m *mockOverrides) MetricsGeneratorNativeHistogramBucketFactor(string) float64 {
	return m.nativeHistogramBucketFactor
}

func (m *mockOverrides) MetricsGeneratorNativeHistogramMaxBucketNumber(string) uint32 {
	return m.nativeHistogramMaxBucketNumber
}

func (m *mockOverrides) MetricsGeneratorNativeHistogramMinResetDuration(string) time.Duration {
	return m.nativeHistogramMinResetDuration
}

func mustGetHostname() string {
	hostname, _ := os.Hostname()
	return hostname
}

func TestManagedRegistry_demandTracking(t *testing.T) {
	appender := &capturingAppender{}

	cfg := &Config{
		StaleDuration: 15 * time.Minute,
	}
	registry := New(cfg, &mockOverrides{}, "test", appender, log.NewNopLogger(), noopLimiter)
	defer registry.Close()

	counter := registry.NewCounter("my_counter")

	// Add series with unique label combinations
	for i := 0; i < 100; i++ {
		lbls := buildTestLabels([]string{"label"}, []string{fmt.Sprintf("value-%d", i)})
		counter.Inc(lbls, 1.0)
	}

	// Collect metrics to update demand tracking
	registry.CollectMetrics(context.Background())

	// Check that active series matches expected
	activeSeries := registry.activeSeries()
	assert.Equal(t, uint32(100), activeSeries)

	// Access the internal metrics to verify demand tracking
	// The demand should be approximately equal to active series (within HLL error)
	var totalDemand int
	registry.metricsMtx.RLock()
	for _, m := range registry.metrics {
		totalDemand += m.countSeriesDemand()
	}
	registry.metricsMtx.RUnlock()

	// HLL with precision 10 has ~3.25% error, so we allow 10% tolerance
	diff := float64(totalDemand-int(activeSeries)) / float64(activeSeries)
	assert.Less(t, math.Abs(diff), 0.1, "demand estimate should be within 10%% of actual series")
}

func TestManagedRegistry_demandExceedsMax(t *testing.T) {
	appender := &capturingAppender{}

	cfg := &Config{
		StaleDuration: 15 * time.Minute,
	}
	rejectLimiter := &mockLimiter{
		onAddFunc: func(uint64, uint32) bool {
			return false
		},
	}
	overrides := &mockOverrides{
		maxActiveSeries: 50,
	}
	registry := New(cfg, overrides, "test", appender, log.NewNopLogger(), rejectLimiter)
	defer registry.Close()

	counter := registry.NewCounter("my_counter")

	// Add series which should all be rejected
	for i := 0; i < 100; i++ {
		lbls := buildTestLabels([]string{"label"}, []string{fmt.Sprintf("value-%d", i)})
		counter.Inc(lbls, 1.0)
	}

	// Collect metrics
	registry.CollectMetrics(context.Background())

	// Active series should be capped at max
	activeSeries := registry.activeSeries()
	assert.Equal(t, uint32(0), activeSeries)

	// Demand tracking should show all attempted series (including rejected ones)
	var totalDemand int
	registry.metricsMtx.RLock()
	for _, m := range registry.metrics {
		totalDemand += m.countSeriesDemand()
	}
	registry.metricsMtx.RUnlock()

	// Demand should be higher than active series since we tried to add 100 series
	assert.Greater(t, totalDemand, int(activeSeries), "demand should track all attempted series")
	assert.Greater(t, totalDemand, 80, "demand should show most of the 100 attempted series")
}

func TestManagedRegistry_demandDecaysOverTime(t *testing.T) {
	appender := &capturingAppender{}

	cfg := &Config{
		StaleDuration: 15 * time.Minute,
	}
	registry := New(cfg, &mockOverrides{}, "test", appender, log.NewNopLogger(), noopLimiter)
	defer registry.Close()

	counter := registry.NewCounter("my_counter")

	// Add series
	for i := 0; i < 50; i++ {
		lbls := buildTestLabels([]string{"label"}, []string{fmt.Sprintf("value-%d", i)})
		counter.Inc(lbls, 1.0)
	}

	registry.CollectMetrics(context.Background())

	// Get initial demand
	var initialDemand int
	registry.metricsMtx.RLock()
	for _, m := range registry.metrics {
		initialDemand += m.countSeriesDemand()
	}
	registry.metricsMtx.RUnlock()

	assert.Greater(t, initialDemand, 0, "initial demand should be non-zero")

	// Advance the cardinality tracker multiple times to evict old data
	registry.metricsMtx.RLock()
	for _, m := range registry.metrics {
		// Advance enough times to clear the sliding window
		for i := 0; i < 5; i++ {
			m.removeStaleSeries(time.Now().Add(time.Hour).UnixMilli())
		}
	}
	registry.metricsMtx.RUnlock()

	registry.CollectMetrics(context.Background())

	// Get demand after decay
	var finalDemand int
	registry.metricsMtx.RLock()
	for _, m := range registry.metrics {
		finalDemand += m.countSeriesDemand()
	}
	registry.metricsMtx.RUnlock()

	// Demand should have decreased significantly or be zero
	assert.Less(t, finalDemand, initialDemand, "demand should decay after advancing the window")
}

func TestManagedRegistry_entityDemandTracking(t *testing.T) {
	appender := &capturingAppender{}

	cfg := &Config{
		StaleDuration: 15 * time.Minute,
	}
	registry := New(cfg, &mockOverrides{}, "test", appender, log.NewNopLogger(), noopLimiter)
	defer registry.Close()

	counter := registry.NewCounter("my_counter")

	// Add series with unique label combinations (entities)
	for i := 0; i < 100; i++ {
		lbls := buildTestLabels([]string{"label"}, []string{fmt.Sprintf("value-%d", i)})
		counter.Inc(lbls, 1.0)
	}

	// Collect metrics to update demand tracking
	registry.CollectMetrics(context.Background())

	// Check entity demand estimate
	entityDemand := registry.entityDemand.Estimate()

	// HLL with precision 10 has ~3.25% error, so we allow 10% tolerance
	// We expect ~100 entities since each label combo is unique
	diff := float64(entityDemand-100) / 100.0
	assert.Less(t, math.Abs(diff), 0.1, "entity demand estimate should be within 10%% of actual entities")
	assert.Greater(t, entityDemand, uint64(80), "entity demand should show most entities")
}

func TestManagedRegistry_entityDemandExceedsMax(t *testing.T) {
	appender := &capturingAppender{}

	cfg := &Config{
		StaleDuration: 15 * time.Minute,
	}
	rejectLimiter := &mockLimiter{
		onAddFunc: func(uint64, uint32) bool {
			return false
		},
	}
	registry := New(cfg, &mockOverrides{}, "test", appender, log.NewNopLogger(), rejectLimiter)
	defer registry.Close()

	counter := registry.NewCounter("my_counter")

	// Add series which should all be rejected
	for i := 0; i < 100; i++ {
		lbls := buildTestLabels([]string{"label"}, []string{fmt.Sprintf("value-%d", i)})
		counter.Inc(lbls, 1.0)
	}

	// Collect metrics
	registry.CollectMetrics(context.Background())

	// Active series should be 0 since all were rejected
	activeSeries := registry.activeSeries()
	assert.Equal(t, uint32(0), activeSeries)

	// Entity demand tracking should NOT show attempted entities since OnUpdate is only called for accepted entities
	entityDemand := registry.entityDemand.Estimate()

	// HLL with precision 10 has ~3.25% error, so we allow 10% tolerance
	// We expect ~100 entities since each label combo is unique
	diff := float64(entityDemand-100) / 100.0
	assert.Less(t, math.Abs(diff), 0.1, "entity demand estimate should be within 10%% of actual entities")
	assert.Greater(t, entityDemand, uint64(80), "entity demand should show most entities")
}

func TestManagedRegistry_entityDemandDecaysOverTime(t *testing.T) {
	appender := &capturingAppender{}

	cfg := &Config{
		StaleDuration: 15 * time.Minute,
	}
	registry := New(cfg, &mockOverrides{}, "test", appender, log.NewNopLogger(), noopLimiter)
	defer registry.Close()

	counter := registry.NewCounter("my_counter")

	// Add entities
	for i := 0; i < 50; i++ {
		lbls := buildTestLabels([]string{"label"}, []string{fmt.Sprintf("value-%d", i)})
		counter.Inc(lbls, 1.0)
	}

	registry.CollectMetrics(context.Background())

	// Get initial entity demand
	initialEntityDemand := registry.entityDemand.Estimate()
	assert.Greater(t, initialEntityDemand, uint64(0), "initial entity demand should be non-zero")

	// Advance the entity demand cardinality tracker multiple times to evict old data
	for i := 0; i < int(removeStaleSeriesInterval/time.Minute)+1; i++ {
		registry.entityDemand.Advance()
	}

	registry.CollectMetrics(context.Background())

	// Get demand after decay
	finalEntityDemand := registry.entityDemand.Estimate()

	// Demand should have decreased significantly or be zero
	assert.Less(t, finalEntityDemand, initialEntityDemand, "entity demand should decay after advancing the window")
}

func TestManagedRegistry_entityDemandWithMultipleMetrics(t *testing.T) {
	appender := &capturingAppender{}

	cfg := &Config{
		StaleDuration: 15 * time.Minute,
	}
	registry := New(cfg, &mockOverrides{}, "test", appender, log.NewNopLogger(), noopLimiter)
	defer registry.Close()

	counter1 := registry.NewCounter("counter_1")
	counter2 := registry.NewCounter("counter_2")
	histogram := registry.NewHistogram("histogram_1", []float64{1.0, 2.0}, HistogramModeClassic)

	// Add the same entity across multiple metrics
	// Since entity demand is based on label hash (not metric name), same labels should count as one entity
	for i := 0; i < 50; i++ {
		lbls := buildTestLabels([]string{"label"}, []string{fmt.Sprintf("value-%d", i)})
		counter1.Inc(lbls, 1.0)
		counter2.Inc(lbls, 2.0)
		histogram.ObserveWithExemplar(lbls, 1.5, "", 1.0)
	}

	registry.CollectMetrics(context.Background())

	// Entity demand should be ~50, not 150, since same label combinations are used across metrics
	entityDemand := registry.entityDemand.Estimate()

	// Allow for HLL estimation error
	diff := float64(entityDemand-50) / 50.0
	assert.Less(t, math.Abs(diff), 0.15, "entity demand should be ~50 since same entities used across metrics")
	assert.Less(t, entityDemand, uint64(70), "entity demand should not triple-count entities across metrics")
}
