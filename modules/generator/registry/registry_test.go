package registry

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/prometheus/model/exemplar"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		lbls := labels.FromMap(map[string]string{
			fmt.Sprintf("name-%d", i): fmt.Sprintf("value-%d", i),
		})

		go accessor(func() {
			registry.metricsMtx.Lock()
			registry.incrementMetric(lbls, 1.0)
			registry.metricsMtx.Unlock()
		})
	}

	go accessor(func() {
		registry.scrape(context.Background())
	})

	go accessor(func() {
		registry.removeStaleMetrics(context.Background())
	})

	time.Sleep(200 * time.Millisecond)
	close(end)
}

func TestManagedRegistry_counter(t *testing.T) {
	capturingAppender := &capturingAppender{}

	registry := New(&Config{}, &mockOverrides{}, "test", capturingAppender, log.NewNopLogger())
	defer registry.Close()

	counter := registry.NewCounter("counter")

	lbls := labels.FromMap(map[string]string{
		"foo": "foo-value",
		"bar": "bar-value",
	})

	counter.Inc(lbls, 1.0)
	counter.Inc(lbls, 2.0)
	counter.Inc(lbls, 1.5)

	scrapeTimeMs := time.Now().UnixMilli()
	registry.scrape(context.Background())

	assert.Equal(t, true, capturingAppender.isCommitted)
	assert.Equal(t, false, capturingAppender.isRolledback)

	require.Len(t, capturingAppender.samples, 1)

	expectedLbls := labels.NewBuilder(lbls).
		Set("__name__", "counter").
		Set("instance", mustGetHostname()).
		Labels()
	assert.Equal(t, expectedLbls, capturingAppender.samples[0].l)
	assert.Equal(t, scrapeTimeMs, capturingAppender.samples[0].t)
	assert.Equal(t, 4.5, capturingAppender.samples[0].v)
}

func TestManagedRegistry_histogram(t *testing.T) {
	capturingAppender := &capturingAppender{}

	registry := New(&Config{}, &mockOverrides{}, "test", capturingAppender, log.NewNopLogger())
	defer registry.Close()

	histogram := registry.NewHistogram("histogram", []float64{1.0, 2.0})

	lbls := labels.FromMap(map[string]string{
		"foo": "foo-value",
		"bar": "bar-value",
	})

	histogram.Observe(lbls, 1.0)
	histogram.Observe(lbls, 2.0)
	histogram.Observe(lbls, 2.5)

	scrapeTimeMs := time.Now().UnixMilli()
	registry.scrape(context.Background())

	assert.Equal(t, true, capturingAppender.isCommitted)
	assert.Equal(t, false, capturingAppender.isRolledback)

	require.Len(t, capturingAppender.samples, 5)

	expectedLbls := labels.NewBuilder(lbls).Set("instance", mustGetHostname())

	expectedSamples := []sample{
		{
			l: expectedLbls.Set("__name__", "histogram_count").Labels(),
			t: scrapeTimeMs,
			v: 3.0,
		},
		{
			l: expectedLbls.Set("__name__", "histogram_sum").Labels(),
			t: scrapeTimeMs,
			v: 5.5,
		},
		{
			l: expectedLbls.Set("__name__", "histogram_bucket").Set("le", "1").Labels(),
			t: scrapeTimeMs,
			v: 1.0,
		},
		{
			l: expectedLbls.Set("__name__", "histogram_bucket").Set("le", "2").Labels(),
			t: scrapeTimeMs,
			v: 2.0,
		},
		{
			l: expectedLbls.Set("__name__", "histogram_bucket").Set("le", "+Inf").Labels(),
			t: scrapeTimeMs,
			v: 3.0,
		},
	}
	for _, expectedSample := range expectedSamples {
		assert.Contains(t, capturingAppender.samples, expectedSample)
	}
}

func TestManagedRegistry_staleSeries(t *testing.T) {
	cfg := &Config{
		StaleDuration: 75 * time.Millisecond,
	}
	registry := New(cfg, &mockOverrides{}, "test", &noopAppender{}, log.NewNopLogger())
	defer registry.Close()

	lbls1 := labels.FromMap(map[string]string{"__name__": "metric-1"})
	lbls2 := labels.FromMap(map[string]string{"__name__": "metric-2"})

	registry.incrementMetric(lbls1, 1.0)
	registry.incrementMetric(lbls2, 1.0)

	registry.removeStaleMetrics(context.Background())
	assert.Len(t, registry.metrics, 2)

	time.Sleep(50 * time.Millisecond)

	registry.incrementMetric(lbls2, 1.0)

	time.Sleep(50 * time.Millisecond)

	registry.removeStaleMetrics(context.Background())
	assert.Len(t, registry.metrics, 1)
}

func TestManagedRegistry_externalLabels(t *testing.T) {
	capturingAppender := capturingAppender{}

	cfg := &Config{
		ExternalLabels: map[string]string{
			"foo": "bar",
		},
	}
	registry := New(cfg, &mockOverrides{}, "test", &capturingAppender, log.NewNopLogger())
	defer registry.Close()

	lbls := labels.FromMap(map[string]string{"__name__": "metric-1"})

	registry.incrementMetric(lbls, 1.0)

	scrapeTimeMs := time.Now().UnixMilli()
	registry.scrape(context.Background())

	expectedLbls := labels.NewBuilder(lbls).
		Set("__name__", "metric-1").
		Set("foo", "bar").
		Set("instance", mustGetHostname()).
		Labels()
	assert.Equal(t, expectedLbls, capturingAppender.samples[0].l)
	assert.Equal(t, scrapeTimeMs, capturingAppender.samples[0].t)
	assert.Equal(t, 1.0, capturingAppender.samples[0].v)
}

func TestManagedRegistry_maxSeries(t *testing.T) {
	overrides := &mockOverrides{
		maxActiveSeries: 1,
	}
	registry := New(&Config{}, overrides, "test", &noopAppender{}, log.NewNopLogger())
	defer registry.Close()

	lbls1 := labels.FromMap(map[string]string{"__name__": "metric-1"})
	lbls2 := labels.FromMap(map[string]string{"__name__": "metric-2"})

	registry.incrementMetric(lbls1, 1.0)
	registry.incrementMetric(lbls2, 1.0)

	assert.Len(t, registry.metrics, 1)
}

type mockOverrides struct {
	maxActiveSeries int
}

var _ Overrides = (*mockOverrides)(nil)

func (m *mockOverrides) MetricsGeneratorMaxActiveSeries(userID string) int {
	return m.maxActiveSeries
}

func (m *mockOverrides) MetricsGeneratorScrapeInterval(userID string) time.Duration {
	return 15 * time.Second
}

type noopAppender struct{}

var _ storage.Appendable = (*noopAppender)(nil)

var _ storage.Appender = (*noopAppender)(nil)

func (n *noopAppender) Appender(ctx context.Context) storage.Appender {
	return n
}

func (n noopAppender) Append(ref storage.SeriesRef, l labels.Labels, t int64, v float64) (storage.SeriesRef, error) {
	return 0, nil
}

func (n noopAppender) AppendExemplar(ref storage.SeriesRef, l labels.Labels, e exemplar.Exemplar) (storage.SeriesRef, error) {
	return 0, nil
}

func (n noopAppender) Commit() error {
	return nil
}

func (n noopAppender) Rollback() error {
	return nil
}

type capturingAppender struct {
	samples      []sample
	isCommitted  bool
	isRolledback bool
}

type sample struct {
	l labels.Labels
	t int64
	v float64
}

func (s sample) String() string {
	return fmt.Sprintf("%s %d %g", s.l, s.t, s.v)
}

var _ storage.Appendable = (*capturingAppender)(nil)

var _ storage.Appender = (*capturingAppender)(nil)

func (c *capturingAppender) Appender(ctx context.Context) storage.Appender {
	return c
}

func (c *capturingAppender) Append(ref storage.SeriesRef, l labels.Labels, t int64, v float64) (storage.SeriesRef, error) {
	c.samples = append(c.samples, sample{l, t, v})
	return ref, nil
}

func (c *capturingAppender) Commit() error {
	c.isCommitted = true
	return nil
}

func (c *capturingAppender) Rollback() error {
	c.isRolledback = true
	return nil
}

func (c *capturingAppender) AppendExemplar(ref storage.SeriesRef, l labels.Labels, e exemplar.Exemplar) (storage.SeriesRef, error) {
	panic("AppendExemplar is not supported")
}

func mustGetHostname() string {
	hostname, _ := os.Hostname()
	return hostname
}
