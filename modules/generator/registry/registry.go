package registry

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/prometheus/storage"
	"go.uber.org/atomic"
)

var (
	metricActiveSeries = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_registry_active_series",
		Help:      "The active series per tenant",
	}, []string{"tenant"})
	metricTotalSeriesAdded = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_registry_series_added_total",
		Help:      "The total amount of series created per tenant",
	}, []string{"tenant"})
	metricTotalSeriesRemoved = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_registry_series_removed_total",
		Help:      "The total amount of series removed after they have become stale per tenant",
	}, []string{"tenant"})
	metricTotalSeriesLimited = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_registry_series_limited_total",
		Help:      "The total amount of series not created because of limits per tenant",
	}, []string{"tenant"})
	metricTotalScrapes = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_registry_scrapes_total",
		Help:      "The total amount of registry scrapes per tenant",
	}, []string{"tenant"})
	metricFailedScrapes = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_registry_scrapes_failed_total",
		Help:      "The total amount of failed registry scrapes per tenant",
	}, []string{"tenant"})
)

type ManagedRegistry struct {
	onShutdown func()

	cfg            *Config
	overrides      Overrides
	tenant         string
	externalLabels map[string]string

	metricsMtx sync.RWMutex
	// TODO we should not allow duplicate metrics, make this map[name]metric?
	metrics      []*metric
	activeSeries atomic.Uint32

	appendable storage.Appendable

	logger                   log.Logger
	metricActiveSeries       prometheus.Gauge
	metricTotalSeriesAdded   prometheus.Counter
	metricTotalSeriesRemoved prometheus.Counter
	metricTotalSeriesLimited prometheus.Counter
	metricTotalScrapes       prometheus.Counter
	metricFailedScrapes      prometheus.Counter
}

var _ Registry = (*ManagedRegistry)(nil)

// New creates a ManagedRegistry. This Registry will scrape itself, write samples into an appender
// and remove stale series.
func New(cfg *Config, overrides Overrides, tenant string, appendable storage.Appendable, logger log.Logger) *ManagedRegistry {
	instanceCtx, cancel := context.WithCancel(context.Background())

	externalLabels := make(map[string]string)
	for k, v := range cfg.ExternalLabels {
		externalLabels[k] = v
	}
	hostname, _ := os.Hostname()
	externalLabels["instance"] = hostname

	r := &ManagedRegistry{
		onShutdown: cancel,

		cfg:            cfg,
		overrides:      overrides,
		tenant:         tenant,
		externalLabels: externalLabels,

		appendable: appendable,

		logger:                   logger,
		metricActiveSeries:       metricActiveSeries.WithLabelValues(tenant),
		metricTotalSeriesAdded:   metricTotalSeriesAdded.WithLabelValues(tenant),
		metricTotalSeriesRemoved: metricTotalSeriesRemoved.WithLabelValues(tenant),
		metricTotalSeriesLimited: metricTotalSeriesLimited.WithLabelValues(tenant),
		metricTotalScrapes:       metricTotalScrapes.WithLabelValues(tenant),
		metricFailedScrapes:      metricFailedScrapes.WithLabelValues(tenant),
	}

	go job(instanceCtx, r.scrape, r.scrapeInterval)
	go job(instanceCtx, r.removeStaleSeries, constantInterval(5*time.Minute))

	return r
}

func (r *ManagedRegistry) onAddMetricSeries() bool {
	maxActiveSeries := r.overrides.MetricsGeneratorMaxActiveSeries(r.tenant)
	if maxActiveSeries != 0 && r.activeSeries.Load() >= maxActiveSeries {
		r.metricTotalSeriesLimited.Inc()
		level.Warn(r.logger).Log("msg", "reached max active series", "active_series", len(r.metrics), "max_active_series", maxActiveSeries)
		return false
	}

	r.activeSeries.Inc()

	r.metricTotalSeriesAdded.Inc()
	r.metricActiveSeries.Inc()
	return true
}

func (r *ManagedRegistry) onRemoveMetricSeries() {
	r.activeSeries.Dec()

	r.metricTotalSeriesRemoved.Inc()
	r.metricActiveSeries.Dec()
}

func (r *ManagedRegistry) scrape(ctx context.Context) {
	r.metricsMtx.RLock()
	defer r.metricsMtx.RUnlock()

	var err error
	defer func() {
		r.metricTotalScrapes.Inc()
		if err != nil {
			level.Error(r.logger).Log("msg", "scraping registry failed", "err", err)
			r.metricFailedScrapes.Inc()
		}
	}()

	var activeSeries uint32

	appender := r.appendable.Appender(ctx)
	scrapeTimeMs := time.Now().UnixMilli()

	for _, m := range r.metrics {
		active, err := m.scrape(appender, scrapeTimeMs, r.externalLabels)
		if err != nil {
			return
		}
		activeSeries += uint32(active)
	}

	// set active series in case there is drift
	r.activeSeries.Store(activeSeries)
	r.metricActiveSeries.Set(float64(activeSeries))

	err = appender.Commit()
	if err != nil {
		return
	}

	level.Info(r.logger).Log("msg", "scraped registry", "active_series", activeSeries)
}

func (r *ManagedRegistry) scrapeInterval() time.Duration {
	interval := r.overrides.MetricsGeneratorScrapeInterval(r.tenant)
	if interval != 0 {
		return interval
	}
	return r.cfg.ScrapeInterval
}

func (r *ManagedRegistry) removeStaleSeries(_ context.Context) {
	r.metricsMtx.RLock()
	defer r.metricsMtx.RUnlock()

	timeMs := time.Now().Add(-1 * r.cfg.StaleDuration).UnixMilli()

	for _, m := range r.metrics {
		m.removeStaleSeries(timeMs)
	}

	level.Info(r.logger).Log("msg", "deleted stale series", "active_series", r.activeSeries.Load())
}

func (r *ManagedRegistry) Close() {
	level.Info(r.logger).Log("msg", "closing registry")
	r.onShutdown()
}

type counter struct {
	m *metric
}

var _ Counter = (*counter)(nil)

func (r *ManagedRegistry) NewCounter(name string, labels []string) Counter {
	c := &counter{
		m: newMetric(name, labels, r.onAddMetricSeries, r.onRemoveMetricSeries),
	}

	r.metricsMtx.Lock()
	defer r.metricsMtx.Unlock()

	r.metrics = append(r.metrics, c.m)

	return c
}

func (c counter) Inc(labelValues []string, value float64) {
	if value < 0 {
		panic("counter can only increase")
	}
	c.m.add(labelValues, value)
}

type histogram struct {
	// TODO this is not ideal, we are manageing series for every individual metric while they all share the same label values
	//  instead of updating, scraping and removing stale series from each metric, we can do them all at once
	sum          *metric
	count        *metric
	bucketLabels map[float64]string
	buckets      map[float64]*metric
	bucketInf    *metric
}

var _ Histogram = (*histogram)(nil)

func (r *ManagedRegistry) NewHistogram(name string, labels []string, buckets []float64) Histogram {
	h := &histogram{
		sum:          newMetric(fmt.Sprintf("%s_sum", name), labels, r.onAddMetricSeries, r.onRemoveMetricSeries),
		count:        newMetric(fmt.Sprintf("%s_count", name), labels, r.onAddMetricSeries, r.onRemoveMetricSeries),
		bucketLabels: make(map[float64]string),
		buckets:      make(map[float64]*metric),
		bucketInf:    nil,
	}

	nameBucket := fmt.Sprintf("%s_bucket", name)
	labelsWithLe := append(labels, "le")

	for _, bucket := range buckets {
		h.bucketLabels[bucket] = formatFloat(bucket)
		h.buckets[bucket] = newMetric(nameBucket, labelsWithLe, r.onAddMetricSeries, r.onRemoveMetricSeries)
	}

	h.bucketInf = newMetric(nameBucket, labelsWithLe, r.onAddMetricSeries, r.onRemoveMetricSeries)

	r.metricsMtx.Lock()
	defer r.metricsMtx.Unlock()

	r.metrics = append(r.metrics, h.sum, h.count, h.bucketInf)
	for _, m := range h.buckets {
		r.metrics = append(r.metrics, m)
	}

	return h
}

func (h histogram) Observe(labelValues []string, value float64) {
	h.sum.add(labelValues, value)
	h.count.add(labelValues, 1)

	labelValuesWithLe := append(labelValues, "")
	leIndex := len(labelValuesWithLe) - 1

	for bucket, m := range h.buckets {
		labelValuesWithLe[leIndex] = h.bucketLabels[bucket]
		m.add(labelValuesWithLe, isLe(value, bucket))
	}

	labelValuesWithLe[leIndex] = "+Inf"
	h.bucketInf.add(labelValuesWithLe, 1)
}

func isLe(value, bucket float64) float64 {
	if value <= bucket {
		return 1.0
	}
	return 0.0
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
