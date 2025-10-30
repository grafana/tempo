package localentitylimiter

import (
	"context"
	"iter"
	"sync"
	"time"

	"github.com/grafana/tempo/modules/generator/cardinality"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const removeStaleSeriesInterval = 5 * time.Minute

type localEntityLimiterMetrics struct {
	activeEntities *prometheus.GaugeVec
	maxEntities    *prometheus.GaugeVec
	entityDemand   *prometheus.GaugeVec
}

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
		entityDemand: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_registry_entity_demand",
			Help:      "The estimated number of entities that would be created if the max active entities were unlimited",
		}, []string{"tenant"}),
	}
}

var metrics = newMetrics(prometheus.DefaultRegisterer)

type LocalEntityLimiter struct {
	mtx               sync.Mutex
	entityLastUpdated map[uint64]time.Time
	maxEntityFunc     func(tenant string) uint32
	demand            *cardinality.Cardinality
	staleDuration     time.Duration
}

func NewLocalEntityLimiter(maxEntityFunc func(tenant string) uint32, staleDuration time.Duration) *LocalEntityLimiter {
	return &LocalEntityLimiter{
		maxEntityFunc:     maxEntityFunc,
		entityLastUpdated: make(map[uint64]time.Time),
		staleDuration:     staleDuration,
		demand:            cardinality.NewCardinality(staleDuration, removeStaleSeriesInterval),
	}
}

func (l *LocalEntityLimiter) TrackEntities(_ context.Context, tenant string, hashes iter.Seq[uint64]) (rejected iter.Seq[uint64], err error) {
	maxEntities := l.maxEntityFunc(tenant)
	shouldLimit := maxEntities != 0

	now := time.Now()

	return func(yield func(uint64) bool) {
		l.mtx.Lock()
		defer l.mtx.Unlock()

		for hash := range hashes {
			l.demand.Insert(hash)

			_, exists := l.entityLastUpdated[hash]

			if exists {
				l.entityLastUpdated[hash] = now
				continue
			}

			if shouldLimit && uint32(len(l.entityLastUpdated)) >= maxEntities {
				if !yield(hash) {
					break
				}
				continue
			}

			l.entityLastUpdated[hash] = now
		}

		metrics.activeEntities.WithLabelValues(tenant).Set(float64(len(l.entityLastUpdated)))
		metrics.maxEntities.WithLabelValues(tenant).Set(float64(maxEntities))
		metrics.entityDemand.WithLabelValues(tenant).Set(float64(l.demand.Estimate()))
	}, nil
}

func (l *LocalEntityLimiter) Prune(context.Context) {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	for hash, lastUpdated := range l.entityLastUpdated {
		if time.Since(lastUpdated) > l.staleDuration {
			delete(l.entityLastUpdated, hash)
		}
	}

	l.demand.Advance()
}
