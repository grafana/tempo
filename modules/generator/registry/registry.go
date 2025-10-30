package registry

import (
	"context"
	"fmt"
	"iter"
	"maps"
	"os"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/prometheus/storage"
	"go.uber.org/atomic"

	localentitylimiter "github.com/grafana/tempo/modules/generator/localentitylimiter"
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
	metricSeriesDemand = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_registry_active_series_demand_estimate",
		Help:      "The active series demand estimate per tenant",
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

	metricsMtx    sync.RWMutex
	metrics       map[string]metric
	activeSeries  atomic.Uint32
	entityLimiter EntityLimiter

	updatedSinceLastCollectionMtx sync.RWMutex
	updatedSinceLastCollection    map[uint64]struct{}

	appendable storage.Appendable

	logger                log.Logger
	limitLogger           *tempo_log.RateLimitedLogger
	metricActiveSeries    prometheus.Gauge
	metricMaxActiveSeries prometheus.Gauge
	metricSeriesDemand    prometheus.Gauge

	metricTotalSeriesAdded   prometheus.Counter
	metricTotalSeriesRemoved prometheus.Counter
	metricTotalSeriesLimited prometheus.Counter
	metricTotalCollections   prometheus.Counter
	metricFailedCollections  prometheus.Counter
}

// metric is the interface for a metric that is managed by ManagedRegistry.
type metric interface {
	name() string
	collectMetrics(appender storage.Appender, timeMs int64) error
	countActiveSeries() int
	// countSeriesDemand estimates the number of active series that would be created if the maxActiveSeries were unlimited.
	countSeriesDemand() int
	removeStaleSeries(staleTimeMs int64)
	deleteByHash(hash uint64)
}

type entityLifecycler interface {
	onAddEntity(entityHash uint64, count uint32) bool
	onUpdateEntity(entityHash uint64)
	onRemoveEntity(count uint32)
}

const highestAggregationInterval = 1 * time.Minute

const removeStaleSeriesInterval = 5 * time.Minute

var _ Registry = (*ManagedRegistry)(nil)

// EntityLimiter is used to limit the number of entities that can be tracked by the registry.
type EntityLimiter interface {
	// TrackEntities tracks the given entities by their hash. If any entities
	// cannot be tracked, their hashes are returned. The happy case of no
	// limiting results in an empty response and nil error.
	TrackEntities(ctx context.Context, tenant string, hashes iter.Seq[uint64]) (rejected iter.Seq[uint64], err error)

	// Prune gives the limiter a chance to perform any periodic cleanup
	// necessary, such as removing expired entities. This method is called
	// periodically by the registry. Note this method is optional and may be a
	// no-op.
	Prune(ctx context.Context)
}

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

		appendable:    appendable,
		entityLimiter: localentitylimiter.NewLocalEntityLimiter(overrides.MetricsGeneratorMaxActiveEntities, cfg.StaleDuration),

		logger:                     logger,
		limitLogger:                tempo_log.NewRateLimitedLogger(1, level.Warn(logger)),
		metricActiveSeries:         metricActiveSeries.WithLabelValues(tenant),
		metricSeriesDemand:         metricSeriesDemand.WithLabelValues(tenant),
		metricMaxActiveSeries:      metricMaxActiveSeries.WithLabelValues(tenant),
		metricTotalSeriesAdded:     metricTotalSeriesAdded.WithLabelValues(tenant),
		metricTotalSeriesRemoved:   metricTotalSeriesRemoved.WithLabelValues(tenant),
		metricTotalSeriesLimited:   metricTotalSeriesLimited.WithLabelValues(tenant),
		metricTotalCollections:     metricTotalCollections.WithLabelValues(tenant),
		metricFailedCollections:    metricFailedCollections.WithLabelValues(tenant),
		updatedSinceLastCollection: make(map[uint64]struct{}),
	}

	go job(instanceCtx, r.CollectMetrics, r.collectionInterval)
	go job(instanceCtx, r.removeStaleSeries, constantInterval(removeStaleSeriesInterval))

	return r
}

func (r *ManagedRegistry) NewLabelValueCombo(labels []string, values []string) *LabelValueCombo {
	if len(labels) != len(values) {
		panic(fmt.Sprintf("length of given label values does not match with labels, labels: %v, label values: %v", labels, values))
	}
	return newLabelValueComboWithMax(labels, values, r.cfg.MaxLabelNameLength, r.cfg.MaxLabelValueLength)
}

func (r *ManagedRegistry) NewCounter(name string) Counter {
	c := newCounter(name, r, r.externalLabels, r.cfg.StaleDuration)
	r.registerMetric(c)
	return c
}

func (r *ManagedRegistry) NewHistogram(name string, buckets []float64, histogramOverride HistogramMode) (h Histogram) {
	traceIDLabelName := r.overrides.MetricsGenerationTraceIDLabelName(r.tenant)

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

func (r *ManagedRegistry) onAddEntity(hash uint64, count uint32) bool {
	r.onUpdateEntity(hash)

	maxActiveSeries := r.overrides.MetricsGeneratorMaxActiveSeries(r.tenant)
	if maxActiveSeries != 0 && r.activeSeries.Load()+count > maxActiveSeries {
		r.metricTotalSeriesLimited.Add(float64(count))
		r.limitLogger.Log("msg", "reached max active series", "active_series", r.activeSeries.Load(), "max_active_series", maxActiveSeries)
		return false
	}

	r.activeSeries.Add(count)

	r.metricTotalSeriesAdded.Add(float64(count))
	return true
}

func (r *ManagedRegistry) onUpdateEntity(hash uint64) {
	r.updatedSinceLastCollectionMtx.Lock()
	r.updatedSinceLastCollection[hash] = struct{}{}
	r.updatedSinceLastCollectionMtx.Unlock()
}

func (r *ManagedRegistry) onRemoveEntity(count uint32) {
	r.activeSeries.Sub(count)
	r.metricTotalSeriesRemoved.Add(float64(count))
}

func (r *ManagedRegistry) CollectMetrics(ctx context.Context) {
	r.updatedSinceLastCollectionMtx.RLock()
	hashes := maps.Clone(r.updatedSinceLastCollection)
	clear(r.updatedSinceLastCollection)
	r.updatedSinceLastCollectionMtx.RUnlock()

	r.metricsMtx.RLock()
	defer r.metricsMtx.RUnlock()

	rejected, err := r.entityLimiter.TrackEntities(ctx, r.tenant, maps.Keys(hashes))
	if err != nil {
		r.limitLogger.Log("msg", "tracking entities failed", "err", err)
	}

	numRejected := 0
	for hash := range rejected {
		numRejected++
		for _, m := range r.metrics {
			m.deleteByHash(hash)
		}
	}

	if numRejected > 0 {
		r.limitLogger.Log("msg", "max active entities reached", "rejected", numRejected, "updated_since_last_collection", len(hashes))
	}

	var activeSeries int
	var seriesDemand int

	for _, m := range r.metrics {
		seriesDemand += m.countSeriesDemand()
		activeSeries += m.countActiveSeries()
	}

	r.activeSeries.Store(uint32(activeSeries))
	r.metricActiveSeries.Set(float64(activeSeries))
	maxActiveSeries := r.overrides.MetricsGeneratorMaxActiveSeries(r.tenant)
	r.metricMaxActiveSeries.Set(float64(maxActiveSeries))

	r.metricSeriesDemand.Set(float64(seriesDemand))

	if r.overrides.MetricsGeneratorDisableCollection(r.tenant) {
		return
	}

	defer func() {
		r.metricTotalCollections.Inc()
		if err != nil {
			level.Error(r.logger).Log("msg", "collecting metrics failed", "err", err)
			r.metricFailedCollections.Inc()
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

func (r *ManagedRegistry) removeStaleSeries(ctx context.Context) {
	r.entityLimiter.Prune(ctx)

	r.metricsMtx.RLock()
	defer r.metricsMtx.RUnlock()

	timeMs := time.Now().Add(-1 * r.cfg.StaleDuration).UnixMilli()

	for _, m := range r.metrics {
		m.removeStaleSeries(staleTimeMs)
	}

	level.Info(r.logger).Log("msg", "deleted stale series", "active_series", r.activeSeries.Load())
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
