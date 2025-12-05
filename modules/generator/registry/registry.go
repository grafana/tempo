package registry

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/prometheus/storage"

	tempo_log "github.com/grafana/tempo/pkg/util/log"
)

var (
	metricEntityDemand = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_registry_entity_demand_estimate",
		Help:      "The entity demand estimate per tenant",
	}, []string{"tenant"})
	metricSeriesDemand = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_registry_active_series_demand_estimate",
		Help:      "The active series demand estimate per tenant",
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
	}, []string{"tenant", "error_type"})
)

type ManagedRegistry struct {
	onShutdown func()

	cfg            *Config
	overrides      Overrides
	tenant         string
	externalLabels map[string]string

	metricsMtx   sync.RWMutex
	metrics      map[string]metric
	entityDemand *Cardinality
	limiter      Limiter

	appendable storage.Appendable

	logger             log.Logger
	limitLogger        *tempo_log.RateLimitedLogger
	metricSeriesDemand prometheus.Gauge
	metricEntityDemand prometheus.Gauge

	metricTotalSeriesAdded   prometheus.Counter
	metricTotalSeriesRemoved prometheus.Counter
	metricTotalSeriesLimited prometheus.Counter
	metricTotalCollections   prometheus.Counter
}

// metric is the interface for a metric that is managed by ManagedRegistry.
type metric interface {
	name() string
	collectMetrics(appender storage.Appender, timeMs int64) error
	countActiveSeries() int
	// countSeriesDemand estimates the number of active series that would be created if the maxActiveSeries were unlimited.
	countSeriesDemand() int
	removeStaleSeries(staleTimeMs int64)
}

const highestAggregationInterval = 1 * time.Minute

const removeStaleSeriesInterval = 5 * time.Minute

var _ Registry = (*ManagedRegistry)(nil)

// Limiter is used to limit the memory consumption of the registry.
type Limiter interface {
	// OnAdd is called when a new entity is created. It returns true if the entity can be created, false otherwise.
	// LabelHash is a hash of all non-constant labels.
	OnAdd(labelHash uint64, seriesCount uint32) bool
	// OnUpdate is called when an entity is updated.
	// LabelHash is a hash of all non-constant labels.
	OnUpdate(labelHash uint64, seriesCount uint32)
	// OnDelete is called when an entity is deleted.
	// LabelHash is a hash of all non-constant labels.
	OnDelete(labelHash uint64, seriesCount uint32)
}

// New creates a ManagedRegistry. This Registry will scrape itself, write samples into an appender
// and remove stale series.
func New(cfg *Config, overrides Overrides, tenant string, appendable storage.Appendable, logger log.Logger, limiter Limiter) *ManagedRegistry {
	instanceCtx, cancel := context.WithCancel(context.Background())

	externalLabels := make(map[string]string)
	for k, v := range cfg.ExternalLabels {
		externalLabels[k] = v
	}
	hostname, _ := os.Hostname()
	externalLabels["__metrics_gen_instance"] = hostname

	if cfg.InjectTenantIDAs != "" {
		externalLabels[cfg.InjectTenantIDAs] = tenant
	}

	r := &ManagedRegistry{
		onShutdown: cancel,

		cfg:            cfg,
		overrides:      overrides,
		tenant:         tenant,
		externalLabels: externalLabels,

		metrics: map[string]metric{},

		appendable:   appendable,
		limiter:      limiter,
		entityDemand: NewCardinality(cfg.StaleDuration, removeStaleSeriesInterval),

		logger:                 logger,
		limitLogger:            tempo_log.NewRateLimitedLogger(1, level.Warn(logger)),
		metricEntityDemand:     metricEntityDemand.WithLabelValues(tenant),
		metricSeriesDemand:     metricSeriesDemand.WithLabelValues(tenant),
		metricTotalCollections: metricTotalCollections.WithLabelValues(tenant),
	}

	go job(instanceCtx, r.CollectMetrics, r.collectionInterval)
	go job(instanceCtx, r.removeStaleSeries, constantInterval(removeStaleSeriesInterval))

	return r
}

func (r *ManagedRegistry) NewLabelBuilder() LabelBuilder {
	return NewLabelBuilder(r.cfg.MaxLabelNameLength, r.cfg.MaxLabelValueLength)
}

func (r *ManagedRegistry) OnAdd(labelHash uint64, seriesCount uint32) bool {
	r.entityDemand.Insert(labelHash)
	return r.limiter.OnAdd(labelHash, seriesCount)
}

func (r *ManagedRegistry) OnUpdate(labelHash uint64, seriesCount uint32) {
	r.entityDemand.Insert(labelHash)
	r.limiter.OnUpdate(labelHash, seriesCount)
}

func (r *ManagedRegistry) OnDelete(labelHash uint64, seriesCount uint32) {
	r.limiter.OnDelete(labelHash, seriesCount)
}

func (r *ManagedRegistry) NewCounter(name string) Counter {
	c := newCounter(name, r, r.externalLabels, r.cfg.StaleDuration)
	r.registerMetric(c)
	return c
}

func (r *ManagedRegistry) NewHistogram(name string, buckets []float64, histogramOverride HistogramMode) (h Histogram) {
	traceIDLabelName := r.overrides.MetricsGeneratorTraceIDLabelName(r.tenant)

	// TODO: Temporary switch: use the old implementation when native histograms
	// are disabled, eventually the new implementation can handle all cases

	if hasNativeHistograms(histogramOverride) {
		h = newNativeHistogram(name, buckets, r, traceIDLabelName, histogramOverride, r.externalLabels, r.tenant, r.overrides, r.cfg.StaleDuration)
	} else {
		h = newHistogram(name, buckets, r, traceIDLabelName, r.externalLabels, r.cfg.StaleDuration)
	}

	r.registerMetric(h)
	return h
}

func (r *ManagedRegistry) NewGauge(name string) Gauge {
	g := newGauge(name, r, r.externalLabels, r.cfg.StaleDuration)
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

func (r *ManagedRegistry) CollectMetrics(ctx context.Context) {
	r.metricsMtx.RLock()
	defer r.metricsMtx.RUnlock()

	var err error
	var seriesDemand int

	for _, m := range r.metrics {
		seriesDemand += m.countSeriesDemand()
	}

	r.metricSeriesDemand.Set(float64(seriesDemand))
	r.metricEntityDemand.Set(float64(r.entityDemand.Estimate()))

	if r.overrides.MetricsGeneratorDisableCollection(r.tenant) {
		return
	}

	defer func() {
		r.metricTotalCollections.Inc()
		if err != nil {
			errT := getErrType(err)
			level.Error(r.logger).Log("msg", "collecting metrics failed", "err", err)
			metricFailedCollections.WithLabelValues(r.tenant, errT).Inc()
		}
	}()

	appender := r.appendable.Appender(ctx)
	collectionTimeMs := time.Now().UnixMilli()

	for _, m := range r.metrics {
		if err = m.collectMetrics(appender, collectionTimeMs); err != nil {
			return
		}
	}

	// Try to avoid committing after we have started the shutdown process.
	if ctx.Err() != nil { // shutdown
		return
	}

	// If the shutdown has started here, a "file already closed" error will be
	// observed here.
	err = appender.Commit()
	if err != nil {
		return
	}
}

func (r *ManagedRegistry) collectionInterval() time.Duration {
	interval := r.overrides.MetricsGeneratorCollectionInterval(r.tenant)
	if interval != 0 {
		return interval
	}
	return r.cfg.CollectionInterval
}

func (r *ManagedRegistry) removeStaleSeries(context.Context) {
	r.metricsMtx.RLock()
	defer r.metricsMtx.RUnlock()

	timeMs := time.Now().Add(-1 * r.cfg.StaleDuration).UnixMilli()

	remainingSeries := 0
	for _, m := range r.metrics {
		m.removeStaleSeries(timeMs)
		remainingSeries += m.countActiveSeries()
	}
	r.entityDemand.Advance()

	level.Info(r.logger).Log("msg", "deleted stale series", "active_series", remainingSeries)
}

func (r *ManagedRegistry) Close() {
	level.Info(r.logger).Log("msg", "closing registry")
	r.onShutdown()
}

func hasNativeHistograms(s HistogramMode) bool {
	return s == HistogramModeNative || s == HistogramModeBoth
}

func hasClassicHistograms(s HistogramMode) bool {
	return s == HistogramModeClassic || s == HistogramModeBoth
}

func getEndOfLastMinuteMs(timeMs int64) int64 {
	return time.UnixMilli(timeMs).Truncate(highestAggregationInterval).Add(-1 * time.Second).UnixMilli()
}
