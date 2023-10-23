package registry

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/prometheus/storage"
	"go.uber.org/atomic"

	tempo_log "github.com/grafana/tempo/pkg/util/log"
)

var (
	metricActiveSeries = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_registry_active_series",
		Help:      "The active series per tenant",
	}, []string{"tenant"})
	metricMaxActiveSeries = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_registry_max_active_series",
		Help:      "The maximum active series per tenant",
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
	metricTotalCollections = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_registry_collections_total",
		Help:      "The total amount of metrics collections per tenant",
	}, []string{"tenant"})
	metricFailedCollections = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_registry_collections_failed_total",
		Help:      "The total amount of failed metrics collections per tenant",
	}, []string{"tenant"})
)

type ManagedRegistry struct {
	onShutdown func()

	cfg            *Config
	overrides      Overrides
	tenant         string
	externalLabels map[string]string

	metricsMtx   sync.RWMutex
	metrics      map[string]metric
	activeSeries atomic.Uint32

	appendable storage.Appendable

	logger                   log.Logger
	limitLogger              *tempo_log.RateLimitedLogger
	metricActiveSeries       prometheus.Gauge
	metricMaxActiveSeries    prometheus.Gauge
	metricTotalSeriesAdded   prometheus.Counter
	metricTotalSeriesRemoved prometheus.Counter
	metricTotalSeriesLimited prometheus.Counter
	metricTotalCollections   prometheus.Counter
	metricFailedCollections  prometheus.Counter
}

// metric is the interface for a metric that is managed by ManagedRegistry.
type metric interface {
	name() string
	collectMetrics(appender storage.Appender, timeMs int64, externalLabels map[string]string) (activeSeries int, err error)
	removeStaleSeries(staleTimeMs int64)
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
	externalLabels["__metrics_gen_instance"] = hostname

	r := &ManagedRegistry{
		onShutdown: cancel,

		cfg:            cfg,
		overrides:      overrides,
		tenant:         tenant,
		externalLabels: externalLabels,

		metrics: map[string]metric{},

		appendable: appendable,

		logger:                   logger,
		limitLogger:              tempo_log.NewRateLimitedLogger(1, level.Warn(logger)),
		metricActiveSeries:       metricActiveSeries.WithLabelValues(tenant),
		metricMaxActiveSeries:    metricMaxActiveSeries.WithLabelValues(tenant),
		metricTotalSeriesAdded:   metricTotalSeriesAdded.WithLabelValues(tenant),
		metricTotalSeriesRemoved: metricTotalSeriesRemoved.WithLabelValues(tenant),
		metricTotalSeriesLimited: metricTotalSeriesLimited.WithLabelValues(tenant),
		metricTotalCollections:   metricTotalCollections.WithLabelValues(tenant),
		metricFailedCollections:  metricFailedCollections.WithLabelValues(tenant),
	}

	go job(instanceCtx, r.collectMetrics, r.collectionInterval)
	go job(instanceCtx, r.removeStaleSeries, constantInterval(5*time.Minute))

	return r
}

func (r *ManagedRegistry) NewLabelValueCombo(labels []string, values []string) *LabelValueCombo {
	if len(labels) != len(values) {
		panic(fmt.Sprintf("length of given label values does not match with labels, labels: %v, label values: %v", labels, values))
	}
	return newLabelValueComboWithMax(labels, values, r.cfg.MaxLabelNameLength, r.cfg.MaxLabelValueLength)
}

func (r *ManagedRegistry) NewCounter(name string) Counter {
	c := newCounter(name, r.onAddMetricSeries, r.onRemoveMetricSeries)
	r.registerMetric(c)
	return c
}

func (r *ManagedRegistry) NewHistogram(name string, buckets []float64) Histogram {
	h := newHistogram(name, buckets, r.onAddMetricSeries, r.onRemoveMetricSeries)
	r.registerMetric(h)
	return h
}

func (r *ManagedRegistry) NewGauge(name string) Gauge {
	g := newGauge(name, r.onAddMetricSeries, r.onRemoveMetricSeries)
	r.registerMetric(g)
	return g
}

func (r *ManagedRegistry) registerMetric(m metric) {
	r.metricsMtx.Lock()
	defer r.metricsMtx.Unlock()

	if _, ok := r.metrics[m.name()]; ok {
		level.Info(r.logger).Log("msg", "replacing metric, counters will be reset", "metric", m.name())
	}
	r.metrics[m.name()] = m
}

func (r *ManagedRegistry) onAddMetricSeries(count uint32) bool {
	maxActiveSeries := r.overrides.MetricsGeneratorMaxActiveSeries(r.tenant)
	if maxActiveSeries != 0 && r.activeSeries.Load()+count > maxActiveSeries {
		r.metricTotalSeriesLimited.Inc()
		r.limitLogger.Log("msg", "reached max active series", "active_series", r.activeSeries.Load(), "max_active_series", maxActiveSeries)
		return false
	}

	r.activeSeries.Add(count)

	r.metricTotalSeriesAdded.Add(float64(count))
	r.metricActiveSeries.Add(float64(count))
	return true
}

func (r *ManagedRegistry) onRemoveMetricSeries(count uint32) {
	r.activeSeries.Sub(count)

	r.metricTotalSeriesRemoved.Add(float64(count))
	r.metricActiveSeries.Sub(float64(count))
}

func (r *ManagedRegistry) collectMetrics(ctx context.Context) {
	if r.overrides.MetricsGeneratorDisableCollection(r.tenant) {
		return
	}

	r.metricsMtx.RLock()
	defer r.metricsMtx.RUnlock()

	var err error
	defer func() {
		r.metricTotalCollections.Inc()
		if err != nil {
			level.Error(r.logger).Log("msg", "collecting metrics failed", "err", err)
			r.metricFailedCollections.Inc()
		}
	}()

	var activeSeries uint32

	appender := r.appendable.Appender(ctx)
	collectionTimeMs := time.Now().UnixMilli()

	for _, m := range r.metrics {
		active, err := m.collectMetrics(appender, collectionTimeMs, r.externalLabels)
		if err != nil {
			return
		}
		activeSeries += uint32(active)
	}

	// set active series in case there is drift
	r.activeSeries.Store(activeSeries)
	r.metricActiveSeries.Set(float64(activeSeries))

	maxActiveSeries := r.overrides.MetricsGeneratorMaxActiveSeries(r.tenant)
	r.metricMaxActiveSeries.Set(float64(maxActiveSeries))

	err = appender.Commit()
	if err != nil {
		return
	}

	level.Info(r.logger).Log("msg", "collecting metrics", "active_series", activeSeries)
}

func (r *ManagedRegistry) collectionInterval() time.Duration {
	interval := r.overrides.MetricsGeneratorCollectionInterval(r.tenant)
	if interval != 0 {
		return interval
	}
	return r.cfg.CollectionInterval
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
