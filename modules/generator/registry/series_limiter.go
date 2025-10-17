package registry

import (
	tempo_log "github.com/grafana/tempo/pkg/util/log"
	"go.uber.org/atomic"
)

// SeriesLimiter is an interface that limits the number of metrics that can be added to the registry.
type SeriesLimiter interface {
	Allow(tenant string, hashes []uint64) bool
	Remove(tenant string, count uint32)
}

type localSeriesLimiter struct {
	overrides    Overrides
	activeSeries atomic.Uint32
	limitLogger  *tempo_log.RateLimitedLogger
}

func newLocalMetricLimiter(overrides Overrides, limitLogger *tempo_log.RateLimitedLogger) *localSeriesLimiter {
	return &localSeriesLimiter{
		overrides:   overrides,
		limitLogger: limitLogger,
	}
}

func (l *localSeriesLimiter) Allow(tenant string, hashes []uint64) bool {
	count := uint32(len(hashes))
	maxActiveSeries := l.overrides.MetricsGeneratorMaxActiveSeries(tenant)
	if maxActiveSeries != 0 && l.activeSeries.Load()+count > maxActiveSeries {
		l.limitLogger.Log("msg", "reached max active series", "active_series", l.activeSeries.Load(), "max_active_series", maxActiveSeries)
		return false
	}

	l.activeSeries.Add(count)

	return true
}

func (l *localSeriesLimiter) Remove(tenant string, count uint32) {
	l.activeSeries.Sub(count)
}
