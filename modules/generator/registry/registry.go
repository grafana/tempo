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
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
)

var (
	metricActiveSeries = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_registry_active_series",
		Help:      "The active series per tenant",
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

	metricsMtx sync.Mutex
	metrics    map[uint64]metric

	appendable storage.Appendable

	logger                   log.Logger
	metricActiveSeries       prometheus.Gauge
	metricTotalSeriesLimited prometheus.Counter
	metricTotalScrapes       prometheus.Counter
	metricFailedScrapes      prometheus.Counter
}

type metric struct {
	lbls       labels.Labels
	value      float64
	lastUpdate time.Time
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

		metrics: make(map[uint64]metric),

		appendable: appendable,

		logger:                   logger,
		metricActiveSeries:       metricActiveSeries.WithLabelValues(tenant),
		metricTotalSeriesLimited: metricTotalSeriesLimited.WithLabelValues(tenant),
		metricTotalScrapes:       metricTotalScrapes.WithLabelValues(tenant),
		metricFailedScrapes:      metricFailedScrapes.WithLabelValues(tenant),
	}

	go job(instanceCtx, r.scrape, r.scrapeInterval)
	go job(instanceCtx, r.removeStaleMetrics, constantInterval(5*time.Minute))

	return r
}

func (r *ManagedRegistry) NewCounter(name string) Counter {
	return &counter{
		name:     name,
		registry: r,
	}
}

func (r *ManagedRegistry) NewHistogram(name string, buckets []float64) Histogram {
	return &histogram{
		nameSum:    fmt.Sprintf("%s_sum", name),
		nameCount:  fmt.Sprintf("%s_count", name),
		nameBucket: fmt.Sprintf("%s_bucket", name),
		buckets:    buckets,
		registry:   r,
	}
}

// incrementMetric adds value to the metric identified by lbls.
// Must be called under lock.
func (r *ManagedRegistry) incrementMetric(lbls labels.Labels, value float64) {
	hash := lbls.Hash()

	s, ok := r.metrics[hash]
	if ok {
		s.value += value
		s.lastUpdate = time.Now()
		r.metrics[hash] = s
		return
	}

	// TODO divide by the amount of instances
	maxActiveSeries := r.overrides.MetricsGeneratorMaxActiveSeries(r.tenant)
	if maxActiveSeries != 0 && len(r.metrics)+1 > maxActiveSeries {
		r.metricTotalSeriesLimited.Inc()
		level.Warn(r.logger).Log("msg", "reached max active series", "active_series", len(r.metrics), "max_active_series", maxActiveSeries)
		return
	}

	r.metrics[hash] = metric{
		lbls:       lbls,
		value:      value,
		lastUpdate: time.Now(),
	}
	r.metricActiveSeries.Inc()
}

func (r *ManagedRegistry) scrape(ctx context.Context) {
	r.metricsMtx.Lock()
	defer r.metricsMtx.Unlock()

	level.Info(r.logger).Log("msg", "scraping registry", "active_series", len(r.metrics))

	var err error
	defer func() {
		r.metricTotalScrapes.Inc()
		if err != nil {
			level.Error(r.logger).Log("msg", "scraping registry failed", "err", err)
			r.metricFailedScrapes.Inc()
		}
	}()

	appender := r.appendable.Appender(ctx)
	scrapeTimeMs := time.Now().UnixMilli()

	for _, s := range r.metrics {
		lb := labels.NewBuilder(s.lbls)
		for k, v := range r.externalLabels {
			lb.Set(k, v)
		}
		lbls := lb.Labels()

		_, err = appender.Append(0, lbls, scrapeTimeMs, s.value)
		if err != nil {
			return
		}
		//// TODO support exemplars
		//_, err = appender.AppendExemplar(ref, lbls, exemplar.Exemplar{
		//	Labels: s.exemplarLabels,
		//	Value:  s.exemplarValue,
		//	HasTs:  true,
		//	Ts:     scrapeTimeMs,
		//})
		//if err != nil {
		//	return
		//}
	}

	err = appender.Commit()
}

func (r *ManagedRegistry) scrapeInterval() time.Duration {
	interval := r.overrides.MetricsGeneratorScrapeInterval(r.tenant)
	if interval != 0 {
		return interval
	}
	return r.cfg.ScrapeInterval
}

func (r *ManagedRegistry) removeStaleMetrics(_ context.Context) {
	r.metricsMtx.Lock()
	defer r.metricsMtx.Unlock()

	lenBefore := len(r.metrics)

	for hash, serie := range r.metrics {
		if time.Since(serie.lastUpdate) > r.cfg.StaleDuration {
			delete(r.metrics, hash)
			r.metricActiveSeries.Dec()
		}
	}

	level.Info(r.logger).Log("msg", "deleted stale series", "active_series", len(r.metrics), "deleted", lenBefore-len(r.metrics))
}

func (r *ManagedRegistry) Close() {
	level.Info(r.logger).Log("msg", "closing registry")
	r.onShutdown()
}

type counter struct {
	name     string
	registry *ManagedRegistry
}

var _ Counter = (*counter)(nil)

func (c counter) Inc(lbls labels.Labels, value float64) {
	if value < 0 {
		panic("counter can only increase")
	}

	lb := labels.NewBuilder(lbls).Set(labels.MetricName, c.name)

	c.registry.metricsMtx.Lock()
	defer c.registry.metricsMtx.Unlock()

	c.registry.incrementMetric(lb.Labels(), value)
}

type histogram struct {
	nameSum    string
	nameCount  string
	nameBucket string
	buckets    []float64
	registry   *ManagedRegistry
}

var _ Histogram = (*histogram)(nil)

func (h histogram) Observe(lbls labels.Labels, value float64) {
	lb := labels.NewBuilder(lbls)

	h.registry.metricsMtx.Lock()
	defer h.registry.metricsMtx.Unlock()

	lb.Set(labels.MetricName, h.nameCount)
	h.registry.incrementMetric(lb.Labels(), 1)

	lb.Set(labels.MetricName, h.nameSum)
	h.registry.incrementMetric(lb.Labels(), value)

	lb.Set(labels.MetricName, h.nameBucket)
	for _, bucket := range h.buckets {
		lb.Set(labels.BucketLabel, formatFloat(bucket))
		h.registry.incrementMetric(lb.Labels(), isLe(value, bucket))
	}

	lb.Set(labels.BucketLabel, "+Inf")
	h.registry.incrementMetric(lb.Labels(), 1.0)
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
