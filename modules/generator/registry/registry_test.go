package registry

import (
	"context"
	"fmt"
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

	for i := 0; i < 4; i++ {
		counter := registry.NewCounter(fmt.Sprintf("counter_%d", i), []string{"label"})
		go accessor(func() {
			counter.Inc([]string{"value-1"}, 1.0)
			counter.Inc([]string{"value-2"}, 2.0)
		})
	}

	for i := 0; i < 4; i++ {
		histogram := registry.NewHistogram(fmt.Sprintf("counter_%d", i), []string{"label"}, []float64{1.0, 2.0})
		go accessor(func() {
			histogram.Observe([]string{"value-1"}, 1.0)
		})
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
		registry.scrape(context.Background())
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

	counter.Inc([]string{"value-1"}, 0.5)
	counter.Inc([]string{"value-1"}, 0.5)
	counter.Inc([]string{"value-2"}, 2.0)

	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1", "instance": mustGetHostname()}, 0, 1),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2", "instance": mustGetHostname()}, 0, 2),
	}
	scrapeRegistryAndAssert(t, registry, appender, expectedSamples)
}

func TestManagedRegistry_histogram(t *testing.T) {
	appender := &capturingAppender{}

	registry := New(&Config{}, &mockOverrides{}, "test", appender, log.NewNopLogger())
	defer registry.Close()

	histogram := registry.NewHistogram("histogram", []string{"label"}, []float64{1.0, 2.0})

	histogram.Observe([]string{"value-1"}, 1.0)
	histogram.Observe([]string{"value-1"}, 2.0)
	histogram.Observe([]string{"value-2"}, 2.5)

	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "histogram_count", "label": "value-1", "instance": mustGetHostname()}, 0, 2),
		newSample(map[string]string{"__name__": "histogram_sum", "label": "value-1", "instance": mustGetHostname()}, 0, 3),
		newSample(map[string]string{"__name__": "histogram_bucket", "label": "value-1", "instance": mustGetHostname(), "le": "1"}, 0, 1),
		newSample(map[string]string{"__name__": "histogram_bucket", "label": "value-1", "instance": mustGetHostname(), "le": "2"}, 0, 2),
		newSample(map[string]string{"__name__": "histogram_bucket", "label": "value-1", "instance": mustGetHostname(), "le": "+Inf"}, 0, 2),
		newSample(map[string]string{"__name__": "histogram_count", "label": "value-2", "instance": mustGetHostname()}, 0, 1),
		newSample(map[string]string{"__name__": "histogram_sum", "label": "value-2", "instance": mustGetHostname()}, 0, 2.5),
		// 0 values are omitted
		//newSample(map[string]string{"__name__": "histogram_bucket", "label": "value-2", "instance": mustGetHostname(), "le": "1"}, 0, 0),
		//newSample(map[string]string{"__name__": "histogram_bucket", "label": "value-2", "instance": mustGetHostname(), "le": "2"}, 0, 0),
		newSample(map[string]string{"__name__": "histogram_bucket", "label": "value-2", "instance": mustGetHostname(), "le": "+Inf"}, 0, 1),
	}
	scrapeRegistryAndAssert(t, registry, appender, expectedSamples)
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

	assert.Equal(t, registry.activeSeries.Load(), uint32(2))
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "metric_1", "instance": mustGetHostname()}, 0, 1),
		newSample(map[string]string{"__name__": "metric_2", "instance": mustGetHostname()}, 0, 2),
	}
	scrapeRegistryAndAssert(t, registry, appender, expectedSamples)

	appender.samples = nil

	time.Sleep(50 * time.Millisecond)
	counter2.Inc(nil, 2)
	time.Sleep(50 * time.Millisecond)

	registry.removeStaleSeries(context.Background())

	assert.Equal(t, registry.activeSeries.Load(), uint32(1))
	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "metric_2", "instance": mustGetHostname()}, 0, 4),
	}
	scrapeRegistryAndAssert(t, registry, appender, expectedSamples)
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
	scrapeRegistryAndAssert(t, registry, appender, expectedSamples)
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

	counter1.Inc([]string{"value-1"}, 1.0)
	// these series should be discarded
	counter1.Inc([]string{"value-2"}, 1.0)
	counter2.Inc(nil, 1.0)

	assert.Equal(t, registry.activeSeries.Load(), uint32(1))
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "metric_1", "label": "value-1", "instance": mustGetHostname()}, 0, 1),
	}
	scrapeRegistryAndAssert(t, registry, appender, expectedSamples)
}

func scrapeRegistryAndAssert(t *testing.T, r *ManagedRegistry, appender *capturingAppender, expectedSamples []sample) {
	scrapeTimeMs := time.Now().UnixMilli()
	r.scrape(context.Background())

	for i := range expectedSamples {
		expectedSamples[i].t = scrapeTimeMs
	}

	assert.Equal(t, true, appender.isCommitted)
	assert.Equal(t, false, appender.isRolledback)
	assert.ElementsMatch(t, expectedSamples, appender.samples)
}

type mockOverrides struct {
	maxActiveSeries uint32
}

var _ Overrides = (*mockOverrides)(nil)

func (m *mockOverrides) MetricsGeneratorMaxActiveSeries(userID string) uint32 {
	return m.maxActiveSeries
}

func (m *mockOverrides) MetricsGeneratorScrapeInterval(userID string) time.Duration {
	return 15 * time.Second
}

func mustGetHostname() string {
	hostname, _ := os.Hostname()
	return hostname
}
