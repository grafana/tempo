package registry

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagedRegistry_concurrency(*testing.T) {
	cfg := &Config{
		StaleDuration: 1 * time.Millisecond,
	}
	registry := New(cfg, &mockOverrides{}, "test", &noopAppender{}, log.NewNopLogger())
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
		registry.collectMetrics(context.Background())
	})

	go accessor(func() {
		registry.removeStaleSeries(context.Background())
	})

	time.Sleep(200 * time.Millisecond)
	close(end)
}

func TestManagedRegistry_counter(t *testing.T) {
	appender := &capturingAppender{}

	registry := New(&Config{}, &mockOverrides{}, "test", appender, log.NewNopLogger())
	defer registry.Close()

	counter := registry.NewCounter("my_counter")

	counter.Inc(newLabelValueCombo([]string{"label"}, []string{"value-1"}), 1.0)

	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1", "__metrics_gen_instance": mustGetHostname()}, 0, 0.0),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1", "__metrics_gen_instance": mustGetHostname()}, 0, 1.0),
	}
	collectRegistryMetricsAndAssert(t, registry, appender, expectedSamples)
}

func TestManagedRegistry_histogram(t *testing.T) {
	appender := &capturingAppender{}

	registry := New(&Config{}, &mockOverrides{}, "test", appender, log.NewNopLogger())
	defer registry.Close()

	histogram := registry.NewHistogram("histogram", []float64{1.0, 2.0})

	histogram.ObserveWithExemplar(newLabelValueCombo([]string{"label"}, []string{"value-1"}), 1.0, "", 1.0)

	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "histogram_count", "label": "value-1", "__metrics_gen_instance": mustGetHostname()}, 0, 1.0),
		newSample(map[string]string{"__name__": "histogram_sum", "label": "value-1", "__metrics_gen_instance": mustGetHostname()}, 0, 1.0),
		newSample(map[string]string{"__name__": "histogram_bucket", "label": "value-1", "__metrics_gen_instance": mustGetHostname(), "le": "1"}, 0, 1.0),
		newSample(map[string]string{"__name__": "histogram_bucket", "label": "value-1", "__metrics_gen_instance": mustGetHostname(), "le": "2"}, 0, 1.0),
		newSample(map[string]string{"__name__": "histogram_bucket", "label": "value-1", "__metrics_gen_instance": mustGetHostname(), "le": "+Inf"}, 0, 1.0),
	}
	collectRegistryMetricsAndAssert(t, registry, appender, expectedSamples)
}

func TestManagedRegistry_removeStaleSeries(t *testing.T) {
	appender := &capturingAppender{}

	cfg := &Config{
		StaleDuration: 75 * time.Millisecond,
	}
	registry := New(cfg, &mockOverrides{}, "test", appender, log.NewNopLogger())
	defer registry.Close()

	counter1 := registry.NewCounter("metric_1")
	counter2 := registry.NewCounter("metric_2")

	counter1.Inc(nil, 1)
	counter2.Inc(nil, 2)

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
	counter2.Inc(nil, 2)
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
	registry := New(cfg, &mockOverrides{}, "test", appender, log.NewNopLogger())
	defer registry.Close()

	counter := registry.NewCounter("my_counter")
	counter.Inc(nil, 1.0)

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
	registry := New(cfg, &mockOverrides{}, "test", appender, log.NewNopLogger())
	defer registry.Close()

	counter := registry.NewCounter("my_counter")
	counter.Inc(nil, 1.0)

	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_counter", "__metrics_gen_instance": mustGetHostname(), "__tempo_tenant": "test"}, 0, 0),
		newSample(map[string]string{"__name__": "my_counter", "__metrics_gen_instance": mustGetHostname(), "__tempo_tenant": "test"}, 0, 1),
	}
	collectRegistryMetricsAndAssert(t, registry, appender, expectedSamples)
}

func TestManagedRegistry_maxSeries(t *testing.T) {
	appender := &capturingAppender{}

	overrides := &mockOverrides{
		maxActiveSeries: 1,
	}
	registry := New(&Config{}, overrides, "test", appender, log.NewNopLogger())
	defer registry.Close()

	counter1 := registry.NewCounter("metric_1")
	counter2 := registry.NewCounter("metric_2")

	counter1.Inc(newLabelValueCombo([]string{"label"}, []string{"value-1"}), 1.0)
	// these series should be discarded
	counter1.Inc(newLabelValueCombo(nil, []string{"value-2"}), 1.0)
	counter2.Inc(nil, 1.0)

	assert.Equal(t, uint32(1), registry.activeSeries.Load())
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "metric_1", "label": "value-1", "__metrics_gen_instance": mustGetHostname()}, 0, 0),
		newSample(map[string]string{"__name__": "metric_1", "label": "value-1", "__metrics_gen_instance": mustGetHostname()}, 0, 1),
	}
	collectRegistryMetricsAndAssert(t, registry, appender, expectedSamples)
}

func TestManagedRegistry_disableCollection(t *testing.T) {
	appender := &capturingAppender{}

	overrides := &mockOverrides{
		disableCollection: true,
	}
	registry := New(&Config{}, overrides, "test", appender, log.NewNopLogger())
	defer registry.Close()

	counter := registry.NewCounter("metric_1")
	counter.Inc(nil, 1.0)

	// active series are still tracked
	assert.Equal(t, uint32(1), registry.activeSeries.Load())
	// but no samples are collected and sent out
	registry.collectMetrics(context.Background())
	assert.Empty(t, appender.samples)
	assert.Empty(t, appender.exemplars)
}

func TestManagedRegistry_maxLabelNameLength(t *testing.T) {
	appender := &capturingAppender{}

	cfg := &Config{
		MaxLabelNameLength:  8,
		MaxLabelValueLength: 5,
	}
	registry := New(cfg, &mockOverrides{}, "test", appender, log.NewNopLogger())
	defer registry.Close()

	counter := registry.NewCounter("counter")
	histogram := registry.NewHistogram("histogram", []float64{1.0})

	counter.Inc(registry.NewLabelValueCombo([]string{"very_lengthy_label"}, []string{"very_length_value"}), 1.0)
	histogram.ObserveWithExemplar(registry.NewLabelValueCombo([]string{"another_very_lengthy_label"}, []string{"another_very_lengthy_value"}), 1.0, "", 1.0)

	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "counter", "very_len": "very_", "__metrics_gen_instance": mustGetHostname()}, 0, 0.0),
		newSample(map[string]string{"__name__": "counter", "very_len": "very_", "__metrics_gen_instance": mustGetHostname()}, 0, 1.0),
		newSample(map[string]string{"__name__": "histogram_count", "another_": "anoth", "__metrics_gen_instance": mustGetHostname()}, 0, 1.0),
		newSample(map[string]string{"__name__": "histogram_sum", "another_": "anoth", "__metrics_gen_instance": mustGetHostname()}, 0, 1.0),
		newSample(map[string]string{"__name__": "histogram_bucket", "another_": "anoth", "__metrics_gen_instance": mustGetHostname(), "le": "1"}, 0, 1.0),
		newSample(map[string]string{"__name__": "histogram_bucket", "another_": "anoth", "__metrics_gen_instance": mustGetHostname(), "le": "+Inf"}, 0, 1.0),
	}
	collectRegistryMetricsAndAssert(t, registry, appender, expectedSamples)
}

func TestValidLabelValueCombo(t *testing.T) {
	appender := &capturingAppender{}

	registry := New(&Config{}, &mockOverrides{}, "test", appender, log.NewNopLogger())
	defer registry.Close()

	assert.Panics(t, func() {
		registry.NewLabelValueCombo([]string{"one-label"}, []string{"one-value", "two-value"})
	})
}

func collectRegistryMetricsAndAssert(t *testing.T, r *ManagedRegistry, appender *capturingAppender, expectedSamples []sample) {
	collectionTimeMs := time.Now().UnixMilli()
	r.collectMetrics(context.Background())

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
	maxActiveSeries   uint32
	disableCollection bool
}

var _ Overrides = (*mockOverrides)(nil)

func (m *mockOverrides) MetricsGeneratorMaxActiveSeries(string) uint32 {
	return m.maxActiveSeries
}

func (m *mockOverrides) MetricsGeneratorCollectionInterval(string) time.Duration {
	return 15 * time.Second
}

func (m *mockOverrides) MetricsGeneratorDisableCollection(string) bool {
	return m.disableCollection
}

func (m *mockOverrides) MetricsGenerationTraceIDLabelName(string) string {
	return ""
}

func mustGetHostname() string {
	hostname, _ := os.Hostname()
	return hostname
}
