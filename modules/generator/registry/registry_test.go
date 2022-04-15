package registry

import (
	"context"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
)

func TestManagedRegistry_concurrency(t *testing.T) {
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
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	go accessor(func() {
		s := make([]rune, 6)
		for i := range s {
			s[i] = letters[rand.Intn(len(letters))]
		}
		registry.NewCounter(string(s), nil)
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

	counter := registry.NewCounter("my_counter", []string{"label"})

	counter.Inc(NewLabelValues([]string{"value-1"}), 1.0)

	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1", "instance": mustGetHostname()}, 0, 1.0),
	}
	collectRegistryMetricsAndAssert(t, registry, appender, expectedSamples)
}

func TestManagedRegistry_histogram(t *testing.T) {
	appender := &capturingAppender{}

	registry := New(&Config{}, &mockOverrides{}, "test", appender, log.NewNopLogger())
	defer registry.Close()

	histogram := registry.NewHistogram("histogram", []string{"label"}, []float64{1.0, 2.0})

	histogram.ObserveWithExemplar(NewLabelValues([]string{"value-1"}), 1.0, "")

	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "histogram_count", "label": "value-1", "instance": mustGetHostname()}, 0, 1.0),
		newSample(map[string]string{"__name__": "histogram_sum", "label": "value-1", "instance": mustGetHostname()}, 0, 1.0),
		newSample(map[string]string{"__name__": "histogram_bucket", "label": "value-1", "instance": mustGetHostname(), "le": "1"}, 0, 1.0),
		newSample(map[string]string{"__name__": "histogram_bucket", "label": "value-1", "instance": mustGetHostname(), "le": "2"}, 0, 1.0),
		newSample(map[string]string{"__name__": "histogram_bucket", "label": "value-1", "instance": mustGetHostname(), "le": "+Inf"}, 0, 1.0),
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

	counter1 := registry.NewCounter("metric_1", nil)
	counter2 := registry.NewCounter("metric_2", nil)

	counter1.Inc(nil, 1)
	counter2.Inc(nil, 2)

	registry.removeStaleSeries(context.Background())

	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "metric_1", "instance": mustGetHostname()}, 0, 1),
		newSample(map[string]string{"__name__": "metric_2", "instance": mustGetHostname()}, 0, 2),
	}
	collectRegistryMetricsAndAssert(t, registry, appender, expectedSamples)

	appender.samples = nil

	time.Sleep(50 * time.Millisecond)
	counter2.Inc(nil, 2)
	time.Sleep(50 * time.Millisecond)

	registry.removeStaleSeries(context.Background())

	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "metric_2", "instance": mustGetHostname()}, 0, 4),
	}
	collectRegistryMetricsAndAssert(t, registry, appender, expectedSamples)
}

func TestManagedRegistry_externalLabels(t *testing.T) {
	appender := &capturingAppender{}

	cfg := &Config{
		ExternalLabels: map[string]string{
			"foo": "bar",
		},
	}
	registry := New(cfg, &mockOverrides{}, "test", appender, log.NewNopLogger())
	defer registry.Close()

	counter := registry.NewCounter("my_counter", nil)
	counter.Inc(nil, 1.0)

	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_counter", "instance": mustGetHostname(), "foo": "bar"}, 0, 1),
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

	counter1 := registry.NewCounter("metric_1", []string{"label"})
	counter2 := registry.NewCounter("metric_2", nil)

	counter1.Inc(NewLabelValues([]string{"value-1"}), 1.0)
	// these series should be discarded
	counter1.Inc(NewLabelValues([]string{"value-2"}), 1.0)
	counter2.Inc(nil, 1.0)

	assert.Equal(t, uint32(1), registry.activeSeries.Load())
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "metric_1", "label": "value-1", "instance": mustGetHostname()}, 0, 1),
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

	counter := registry.NewCounter("metric_1", nil)
	counter.Inc(nil, 1.0)

	// active series are still tracked
	assert.Equal(t, uint32(1), registry.activeSeries.Load())
	// but no samples are collected and sent out
	registry.collectMetrics(context.Background())
	assert.Empty(t, appender.samples)
	assert.Empty(t, appender.exemplars)
}

func collectRegistryMetricsAndAssert(t *testing.T, r *ManagedRegistry, appender *capturingAppender, expectedSamples []sample) {
	assert.Equal(t, uint32(len(expectedSamples)), r.activeSeries.Load())

	collectionTimeMs := time.Now().UnixMilli()
	r.collectMetrics(context.Background())

	for i := range expectedSamples {
		expectedSamples[i].t = collectionTimeMs
	}

	assert.Equal(t, true, appender.isCommitted)
	assert.Equal(t, false, appender.isRolledback)
	assert.ElementsMatch(t, expectedSamples, appender.samples)
}

type mockOverrides struct {
	maxActiveSeries   uint32
	disableCollection bool
}

var _ Overrides = (*mockOverrides)(nil)

func (m *mockOverrides) MetricsGeneratorMaxActiveSeries(userID string) uint32 {
	return m.maxActiveSeries
}

func (m *mockOverrides) MetricsGeneratorCollectionInterval(userID string) time.Duration {
	return 15 * time.Second
}

func (m *mockOverrides) MetricsGeneratorDisableCollection(userID string) bool {
	return m.disableCollection
}

func mustGetHostname() string {
	hostname, _ := os.Hostname()
	return hostname
}
