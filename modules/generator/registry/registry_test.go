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
	"github.com/grafana/tempo/modules/overrides"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagedRegistry_concurrency(*testing.T) {
	cfg := &Config{
		StaleDuration:      1 * time.Millisecond,
		CollectionInterval: 1 * time.Second,
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
		registry.CollectMetrics(context.Background())
	})

	time.Sleep(200 * time.Millisecond)
	close(end)
}

func TestManagedRegistry_counter(t *testing.T) {
	appender := newCapturingAppender()
	trigger := make(chan struct{})
	instanceCtx, cancel := context.WithCancel(context.Background())
	registry := newWithTrigger(&Config{
		CollectionInterval: 1 * time.Second,
		StaleDuration:      10 * time.Second,
	}, &mockOverrides{}, "test", appender, log.NewNopLogger(), trigger, instanceCtx, cancel)
	defer registry.Close()

	counter := registry.NewCounter("my_counter")

	counter.Inc(newLabelValueCombo([]string{"label"}, []string{"value-1"}), 1.0)

	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1", "__metrics_gen_instance": mustGetHostname()}, 0, 0.0),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1", "__metrics_gen_instance": mustGetHostname()}, 0, 1.0),
	}
	ctx, cncl := context.WithTimeout(context.Background(), 10*time.Second)
	defer cncl()
	collectRegistryMetricsAndAssert(t, trigger, ctx, registry, appender, expectedSamples)
}

func TestManagedRegistry_histogram(t *testing.T) {
	appender := newCapturingAppender()
	trigger := make(chan struct{})
	instanceCtx, cancel := context.WithCancel(context.Background())
	registry := newWithTrigger(&Config{
		CollectionInterval: 1 * time.Second,
		StaleDuration:      10 * time.Second,
	}, &mockOverrides{}, "test", appender, log.NewNopLogger(), trigger, instanceCtx, cancel)
	defer registry.Close()

	histogram := registry.NewHistogram("histogram", []float64{1.0, 2.0}, HistogramModeClassic)

	histogram.ObserveWithExemplar(newLabelValueCombo([]string{"label"}, []string{"value-1"}), 1.0, "", 1.0)

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
	ctx, cncl := context.WithTimeout(context.Background(), 10*time.Second)
	defer cncl()
	collectRegistryMetricsAndAssert(t, trigger, ctx, registry, appender, expectedSamples)
}

func TestManagedRegistry_removeStaleSeries(t *testing.T) {
	appender := newCapturingAppender()

	cfg := &Config{
		StaleDuration:      4 * time.Second,
		CollectionInterval: 2 * time.Second,
	}
	trigger := make(chan struct{})
	instanceCtx, cancel := context.WithCancel(context.Background())
	registry := newWithTrigger(cfg, &mockOverrides{}, "test", appender, log.NewNopLogger(), trigger, instanceCtx, cancel)
	defer registry.Close()

	counter1 := registry.NewCounter("metric_1")
	counter2 := registry.NewCounter("metric_2")

	counter1.Inc(nil, 1)
	counter2.Inc(nil, 2)

	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "metric_1", "__metrics_gen_instance": mustGetHostname()}, 0, 0),
		newSample(map[string]string{"__name__": "metric_1", "__metrics_gen_instance": mustGetHostname()}, 0, 1),
		newSample(map[string]string{"__name__": "metric_2", "__metrics_gen_instance": mustGetHostname()}, 0, 0),
		newSample(map[string]string{"__name__": "metric_2", "__metrics_gen_instance": mustGetHostname()}, 0, 2),
	}
	ctx, cncl := context.WithTimeout(context.Background(), 10*time.Second)
	defer cncl()
	collectRegistryMetricsAndAssert(t, trigger, ctx, registry, appender, expectedSamples)
	time.Sleep(5 * time.Second)
	counter2.Inc(nil, 2)
	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "metric_1", "__metrics_gen_instance": mustGetHostname()}, 0, staleMarker()),
		newSample(map[string]string{"__name__": "metric_2", "__metrics_gen_instance": mustGetHostname()}, 0, 4),
	}
	collectRegistryMetricsAndAssert(t, trigger, ctx, registry, appender, expectedSamples)
}

func TestManagedRegistry_externalLabels(t *testing.T) {
	appender := newCapturingAppender()

	trigger := make(chan struct{})
	instanceCtx, cancel := context.WithCancel(context.Background())
	registry := newWithTrigger(&Config{
		CollectionInterval: 1 * time.Second,
		StaleDuration:      1 * time.Second,
		ExternalLabels: map[string]string{
			"__foo": "bar",
		},
	}, &mockOverrides{}, "test", appender, log.NewNopLogger(), trigger, instanceCtx, cancel)
	defer registry.Close()

	counter := registry.NewCounter("my_counter")
	counter.Inc(nil, 1.0)

	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_counter", "__metrics_gen_instance": mustGetHostname(), "__foo": "bar"}, 0, 0),
		newSample(map[string]string{"__name__": "my_counter", "__metrics_gen_instance": mustGetHostname(), "__foo": "bar"}, 0, 1),
	}
	ctx, cncl := context.WithTimeout(context.Background(), 10*time.Second)
	defer cncl()
	collectRegistryMetricsAndAssert(t, trigger, ctx, registry, appender, expectedSamples)
}

func TestManagedRegistry_injectTenantIDAs(t *testing.T) {
	appender := newCapturingAppender()
	trigger := make(chan struct{})
	instanceCtx, cancel := context.WithCancel(context.Background())
	registry := newWithTrigger(&Config{
		CollectionInterval: 1 * time.Second,
		StaleDuration:      10 * time.Second,
		InjectTenantIDAs:   "__tempo_tenant",
	}, &mockOverrides{}, "test", appender, log.NewNopLogger(), trigger, instanceCtx, cancel)
	defer registry.Close()

	counter := registry.NewCounter("my_counter")
	counter.Inc(nil, 1.0)

	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_counter", "__metrics_gen_instance": mustGetHostname(), "__tempo_tenant": "test"}, 0, 0),
		newSample(map[string]string{"__name__": "my_counter", "__metrics_gen_instance": mustGetHostname(), "__tempo_tenant": "test"}, 0, 1),
	}
	ctx, cncl := context.WithTimeout(context.Background(), 10*time.Second)
	defer cncl()
	collectRegistryMetricsAndAssert(t, trigger, ctx, registry, appender, expectedSamples)
}

func TestManagedRegistry_maxSeries(t *testing.T) {
	appender := newCapturingAppender()
	trigger := make(chan struct{})
	instanceCtx, cancel := context.WithCancel(context.Background())
	registry := newWithTrigger(&Config{
		CollectionInterval: 1 * time.Second,
		StaleDuration:      10 * time.Second,
	}, &mockOverrides{
		maxActiveSeries: 1,
	}, "test", appender, log.NewNopLogger(), trigger, instanceCtx, cancel)
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
	ctx, cncl := context.WithTimeout(context.Background(), 10*time.Second)
	defer cncl()
	collectRegistryMetricsAndAssert(t, trigger, ctx, registry, appender, expectedSamples)
}

func TestManagedRegistry_disableCollection(t *testing.T) {
	appender := newCapturingAppender()
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
	registry.CollectMetrics(context.Background())
	assert.Empty(t, appender.samples)
	assert.Empty(t, appender.exemplars)
}

func TestManagedRegistry_maxLabelNameLength(t *testing.T) {
	appender := newCapturingAppender()
	trigger := make(chan struct{})
	instanceCtx, cancel := context.WithCancel(context.Background())
	registry := newWithTrigger(&Config{
		MaxLabelNameLength:  8,
		MaxLabelValueLength: 5,
		CollectionInterval:  1 * time.Second,
		StaleDuration:       10 * time.Second,
	}, &mockOverrides{}, "test", appender, log.NewNopLogger(), trigger, instanceCtx, cancel)
	defer registry.Close()

	counter := registry.NewCounter("counter")
	histogram := registry.NewHistogram("histogram", []float64{1.0}, HistogramModeClassic)

	counter.Inc(registry.NewLabelValueCombo([]string{"very_lengthy_label"}, []string{"very_length_value"}), 1.0)
	histogram.ObserveWithExemplar(registry.NewLabelValueCombo([]string{"another_very_lengthy_label"}, []string{"another_very_lengthy_value"}), 1.0, "", 1.0)

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
	ctx, cncl := context.WithTimeout(context.Background(), 10*time.Second)
	defer cncl()
	collectRegistryMetricsAndAssert(t, trigger, ctx, registry, appender, expectedSamples)
}

func TestValidLabelValueCombo(t *testing.T) {
	appender := newCapturingAppender()

	registry := New(&Config{}, &mockOverrides{}, "test", appender, log.NewNopLogger())
	defer registry.Close()

	assert.Panics(t, func() {
		registry.NewLabelValueCombo([]string{"one-label"}, []string{"one-value", "two-value"})
	})
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
			appender := newCapturingAppender()
			overrides := &mockOverrides{}
			registry := New(&Config{}, overrides, "test", appender, log.NewNopLogger())
			defer registry.Close()

			tt := registry.NewHistogram("histogram", []float64{1.0, 2.0}, c.nativeHistogramMode)
			require.IsType(t, c.typeOfHistogram, tt)
		})
	}
}

func collectRegistryMetricsAndAssert(t *testing.T, trigger chan struct{}, ctx context.Context, _ *ManagedRegistry, appender *capturingAppender, expectedSamples []sample) {
	collectionTimeMs := time.Now().UnixMilli()

	var actualSamples []sample

	// This will trigger a collection.
	trigger <- struct{}{}

	actualSamples = <-appender.onCommit

	// Ignore the collection time on expected samples, since we won't know when the collection will actually take place.
	for i := range expectedSamples {
		expectedSamples[i].t = collectionTimeMs
	}

	// Ignore the collection time on the collected samples.  Initial counter values will be offset from the collection time.
	for i := range actualSamples {
		actualSamples[i].t = collectionTimeMs
	}

	require.Equal(t, true, appender.isCommitted)
	require.Equal(t, false, appender.isRolledback)

	for i := range actualSamples {
		println(actualSamples[i].String())
	}
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

		require.Equal(t, expected.t, actual.t)
		// Silly nan checks.
		if math.IsNaN(expected.v) {
			require.True(t, math.IsNaN(actual.v))
		} else {
			require.Equal(t, expected.v, actual.v)
		}
		// Rely on the fact that Go prints map keys in sorted order.
		// See https://tip.golang.org/doc/go1.12#fmt.
		require.Equal(t, fmt.Sprint(expected.l.Map()), fmt.Sprint(actual.l.Map()))
	}
}

type mockOverrides struct {
	maxActiveSeries          uint32
	disableCollection        bool
	generateNativeHistograms overrides.HistogramMethod
}

var _ Overrides = (*mockOverrides)(nil)

func (m *mockOverrides) MetricsGeneratorMaxActiveSeries(string) uint32 {
	return m.maxActiveSeries
}

func (m *mockOverrides) MetricsGeneratorCollectionInterval(string) time.Duration {
	return 1 * time.Second
}

func (m *mockOverrides) MetricsGeneratorDisableCollection(string) bool {
	return m.disableCollection
}

func (m *mockOverrides) MetricsGeneratorGenerateNativeHistograms(_ string) overrides.HistogramMethod {
	return m.generateNativeHistograms
}

func (m *mockOverrides) MetricsGenerationTraceIDLabelName(string) string {
	return ""
}

func mustGetHostname() string {
	hostname, _ := os.Hostname()
	return hostname
}
