package localentitylimiter

import (
	"sync"

	"github.com/grafana/tempo/modules/generator/registry"
	tempo_log "github.com/grafana/tempo/pkg/util/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type localEntityLimiterMetrics struct {
	activeEntities *prometheus.GaugeVec
	maxEntities    *prometheus.GaugeVec
}

var _ registry.Limiter = (*LocalEntityLimiter)(nil)

func newMetrics(reg prometheus.Registerer) localEntityLimiterMetrics {
	return localEntityLimiterMetrics{
		activeEntities: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_registry_active_entities",
			Help:      "The number of active entities in the metrics generator registry",
		}, []string{"tenant"}),
		maxEntities: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_registry_max_active_entities",
			Help:      "The maximum number of entities allowed to be active in the metrics generator registry",
		}, []string{"tenant"}),
	}
}

var metrics = newMetrics(prometheus.DefaultRegisterer)

type LocalEntityLimiter struct {
	tenant               string
	entityLastUpdated    map[uint64]uint32
	mtx                  sync.Mutex
	maxEntityFunc        func(tenant string) uint32
	limitLogger          *tempo_log.RateLimitedLogger
	metricActiveEntities prometheus.Gauge
	metricMaxEntities    prometheus.Gauge
}

func New(maxEntityFunc func(tenant string) uint32, tenant string, limitLogger *tempo_log.RateLimitedLogger) *LocalEntityLimiter {
	return &LocalEntityLimiter{
		tenant:               tenant,
		entityLastUpdated:    make(map[uint64]uint32),
		maxEntityFunc:        maxEntityFunc,
		limitLogger:          limitLogger,
		metricActiveEntities: metrics.activeEntities.WithLabelValues(tenant),
		metricMaxEntities:    metrics.maxEntities.WithLabelValues(tenant),
	}
}

func (l *LocalEntityLimiter) OnAdd(labelHash uint64, seriesCount uint32) bool {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	activeSeries, ok := l.entityLastUpdated[labelHash]
	if ok {
		activeSeries += seriesCount
		l.entityLastUpdated[labelHash] = activeSeries
		return true
	}

	maxEntities := l.maxEntityFunc(l.tenant)
	if maxEntities != 0 && uint32(len(l.entityLastUpdated))+1 > maxEntities {
		l.limitLogger.Log("msg", "reached max active entities", "active_entities", len(l.entityLastUpdated), "max_active_entities", maxEntities)
		return false
	}

	l.entityLastUpdated[labelHash] = seriesCount

	l.metricActiveEntities.Set(float64(len(l.entityLastUpdated)))
	l.metricMaxEntities.Set(float64(maxEntities))
	return true
}

func (l *LocalEntityLimiter) OnDelete(labelHash uint64, seriesCount uint32) {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	activeSeries, ok := l.entityLastUpdated[labelHash]
	if ok {
		activeSeries -= seriesCount
		if activeSeries <= 0 {
			delete(l.entityLastUpdated, labelHash)
		} else {
			l.entityLastUpdated[labelHash] = activeSeries
		}
	}

	l.metricActiveEntities.Set(float64(len(l.entityLastUpdated)))
}
