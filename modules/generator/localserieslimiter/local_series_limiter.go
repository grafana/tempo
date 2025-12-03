package localserieslimiter

import (
	"go.uber.org/atomic"

	"github.com/grafana/tempo/modules/generator/registry"
	tempo_log "github.com/grafana/tempo/pkg/util/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type localSeriesLimiterMetrics struct {
	totalSeriesLimited *prometheus.CounterVec
	activeSeries       *prometheus.GaugeVec
	maxActiveSeries    *prometheus.GaugeVec
	totalSeriesAdded   *prometheus.CounterVec
	totalSeriesRemoved *prometheus.CounterVec
}

func newMetrics(reg prometheus.Registerer) localSeriesLimiterMetrics {
	return localSeriesLimiterMetrics{
		totalSeriesLimited: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_registry_series_limited_total",
			Help:      "The total amount of series not created because of limits per tenant",
		}, []string{"tenant"}),
		activeSeries: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_registry_active_series",
			Help:      "The active series per tenant",
		}, []string{"tenant"}),
		maxActiveSeries: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_registry_max_active_series",
			Help:      "The maximum active series per tenant",
		}, []string{"tenant"}),
		totalSeriesAdded: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_registry_series_added_total",
			Help:      "The total amount of series created per tenant",
		}, []string{"tenant"}),
		totalSeriesRemoved: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_registry_series_removed_total",
			Help:      "The total amount of series removed after they have become stale per tenant",
		}, []string{"tenant"}),
	}
}

var metrics = newMetrics(prometheus.DefaultRegisterer)

type LocalSeriesLimiter struct {
	tenant                   string
	activeSeries             atomic.Uint32
	maxSeriesFunc            func(tenant string) uint32
	limitLogger              *tempo_log.RateLimitedLogger
	metricTotalSeriesLimited prometheus.Counter
	metricActiveSeries       prometheus.Gauge
	metricMaxActiveSeries    prometheus.Gauge
	metricTotalSeriesAdded   prometheus.Counter
	metricTotalSeriesRemoved prometheus.Counter
}

var _ registry.Limiter = (*LocalSeriesLimiter)(nil)

func New(maxSeriesFunc func(tenant string) uint32, tenant string, limitLogger *tempo_log.RateLimitedLogger) *LocalSeriesLimiter {
	return &LocalSeriesLimiter{
		tenant:                   tenant,
		maxSeriesFunc:            maxSeriesFunc,
		limitLogger:              limitLogger,
		metricTotalSeriesLimited: metrics.totalSeriesLimited.WithLabelValues(tenant),
		metricActiveSeries:       metrics.activeSeries.WithLabelValues(tenant),
		metricMaxActiveSeries:    metrics.maxActiveSeries.WithLabelValues(tenant),
		metricTotalSeriesAdded:   metrics.totalSeriesAdded.WithLabelValues(tenant),
		metricTotalSeriesRemoved: metrics.totalSeriesRemoved.WithLabelValues(tenant),
	}
}

func (l *LocalSeriesLimiter) OnAdd(_ uint64, seriesCount uint32) bool {
	maxSeries := l.maxSeriesFunc(l.tenant)
	if maxSeries != 0 && l.activeSeries.Load()+seriesCount > maxSeries {
		l.metricTotalSeriesLimited.Add(float64(seriesCount))
		l.limitLogger.Log("msg", "reached max active series", "active_series", l.activeSeries.Load(), "max_active_series", maxSeries)
		return false
	}

	l.activeSeries.Add(seriesCount)
	l.metricActiveSeries.Set(float64(l.activeSeries.Load()))
	l.metricMaxActiveSeries.Set(float64(maxSeries))
	l.metricTotalSeriesAdded.Add(float64(seriesCount))
	return true
}

func (l *LocalSeriesLimiter) OnUpdate(uint64, uint32) {
	// No-op, we rely on OnDelete to clean up the series
}

func (l *LocalSeriesLimiter) OnDelete(_ uint64, seriesCount uint32) {
	l.activeSeries.Sub(seriesCount)
	l.metricActiveSeries.Set(float64(l.activeSeries.Load()))
	l.metricTotalSeriesRemoved.Add(float64(seriesCount))
}
