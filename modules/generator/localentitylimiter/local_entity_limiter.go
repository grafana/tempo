package localentitylimiter

import (
	"sync"

	"github.com/grafana/tempo/modules/generator/registry"
	tempo_log "github.com/grafana/tempo/pkg/util/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type localEntityLimiterMetrics struct {
	totalEntitiesLimited *prometheus.CounterVec
	activeEntities       *prometheus.GaugeVec
	maxActiveEntities    *prometheus.GaugeVec
	totalEntitiesAdded   *prometheus.CounterVec
	totalEntitiesRemoved *prometheus.CounterVec
}

var _ registry.Limiter = (*LocalEntityLimiter)(nil)

func newMetrics(reg prometheus.Registerer) localEntityLimiterMetrics {
	return localEntityLimiterMetrics{
		totalEntitiesLimited: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_registry_entities_limited_total",
			Help:      "The total amount of entities not created because of limits per tenant",
		}, []string{"tenant"}),
		activeEntities: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_registry_active_entities",
			Help:      "The number of active entities in the metrics generator registry",
		}, []string{"tenant"}),
		maxActiveEntities: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_registry_max_active_entities",
			Help:      "The maximum number of entities allowed to be active in the metrics generator registry",
		}, []string{"tenant"}),
		totalEntitiesAdded: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_registry_entities_added_total",
			Help:      "The total amount of entities created per tenant",
		}, []string{"tenant"}),
		totalEntitiesRemoved: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_registry_entities_removed_total",
			Help:      "The total amount of entities removed after they have become stale per tenant",
		}, []string{"tenant"}),
	}
}

var metrics = newMetrics(prometheus.DefaultRegisterer)

type LocalEntityLimiter struct {
	tenant             string
	entityActiveSeries map[uint64]uint32
	mtx                sync.Mutex
	maxEntityFunc      func(tenant string) uint32
	limitLogger        *tempo_log.RateLimitedLogger

	metricTotalEntitiesLimited prometheus.Counter
	metricActiveEntities       prometheus.Gauge
	metricMaxActiveEntities    prometheus.Gauge
	metricTotalEntitiesAdded   prometheus.Counter
	metricTotalEntitiesRemoved prometheus.Counter
}

func New(maxEntityFunc func(tenant string) uint32, tenant string, limitLogger *tempo_log.RateLimitedLogger) *LocalEntityLimiter {
	return &LocalEntityLimiter{
		tenant:             tenant,
		entityActiveSeries: make(map[uint64]uint32),
		maxEntityFunc:      maxEntityFunc,
		limitLogger:        limitLogger,

		metricTotalEntitiesLimited: metrics.totalEntitiesLimited.WithLabelValues(tenant),
		metricActiveEntities:       metrics.activeEntities.WithLabelValues(tenant),
		metricMaxActiveEntities:    metrics.maxActiveEntities.WithLabelValues(tenant),
		metricTotalEntitiesAdded:   metrics.totalEntitiesAdded.WithLabelValues(tenant),
		metricTotalEntitiesRemoved: metrics.totalEntitiesRemoved.WithLabelValues(tenant),
	}
}

func (l *LocalEntityLimiter) OnAdd(labelHash uint64, seriesCount uint32) bool {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	activeSeries, ok := l.entityActiveSeries[labelHash]
	if ok {
		activeSeries += seriesCount
		l.entityActiveSeries[labelHash] = activeSeries
		return true
	}

	maxEntities := l.maxEntityFunc(l.tenant)
	if maxEntities != 0 && uint32(len(l.entityActiveSeries))+1 > maxEntities {
		l.limitLogger.Log("msg", "reached max active entities", "active_entities", len(l.entityActiveSeries), "max_active_entities", maxEntities)
		l.metricTotalEntitiesLimited.Add(float64(1))
		return false
	}

	l.entityActiveSeries[labelHash] = seriesCount

	l.metricActiveEntities.Set(float64(len(l.entityActiveSeries)))
	l.metricMaxActiveEntities.Set(float64(maxEntities))
	l.metricTotalEntitiesAdded.Add(float64(1))
	return true
}

func (l *LocalEntityLimiter) OnUpdate(uint64, uint32) {
	// No-op, we rely on OnDelete to clean up
}

func (l *LocalEntityLimiter) OnDelete(labelHash uint64, seriesCount uint32) {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	activeSeries, ok := l.entityActiveSeries[labelHash]
	if !ok {
		return
	}

	// Guard against accidental overflow. This is a programming error, but we
	// should be defensive.
	if seriesCount > activeSeries {
		seriesCount = activeSeries
	}
	activeSeries -= seriesCount
	if activeSeries == 0 {
		delete(l.entityActiveSeries, labelHash)
	} else {
		l.entityActiveSeries[labelHash] = activeSeries
	}

	l.metricActiveEntities.Set(float64(len(l.entityActiveSeries)))
	l.metricTotalEntitiesRemoved.Add(1)
}
