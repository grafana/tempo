package registry

import (
	"sync"

	"github.com/go-kit/log"
	tempo_log "github.com/grafana/tempo/pkg/util/log"
	"go.uber.org/atomic"
)

type SeriesLimiterFactory func(tenant string) SeriesLimiter

// SeriesLimiter is an interface that limits the number of metrics that can be added to the registry.
type SeriesLimiter interface {
	Allow(hashes []uint64) bool
	Remove(count uint32)
}

type multiTenantSeriesLimiter struct {
	overrides   Overrides
	limitLogger *tempo_log.RateLimitedLogger

	tenantsMtx sync.Mutex
	tenants    map[string]*singleTenantSeriesLimiter
}

func NewLocalSeriesLimiterFactory(overrides Overrides, logger log.Logger) SeriesLimiterFactory {
	return func(tenant string) SeriesLimiter {
		return newSingleTenantSeriesLimiter(tenant, overrides, tempo_log.NewRateLimitedLogger(1, logger))
	}
}

func newSingleTenantSeriesLimiter(tenant string, overrides Overrides, limitLogger *tempo_log.RateLimitedLogger) *singleTenantSeriesLimiter {
	return &singleTenantSeriesLimiter{
		tenant:      tenant,
		overrides:   overrides,
		limitLogger: limitLogger,
	}
}

type singleTenantSeriesLimiter struct {
	tenant       string
	overrides    Overrides
	activeSeries atomic.Uint32
	limitLogger  *tempo_log.RateLimitedLogger
}

func (l *singleTenantSeriesLimiter) Allow(hashes []uint64) bool {
	count := uint32(len(hashes))
	maxActiveSeries := l.overrides.MetricsGeneratorMaxActiveSeries(l.tenant)
	if maxActiveSeries != 0 && l.activeSeries.Load()+count > maxActiveSeries {
		l.limitLogger.Log("msg", "reached max active series", "active_series", l.activeSeries.Load(), "max_active_series", maxActiveSeries)
		return false
	}

	l.activeSeries.Add(count)

	return true
}

func (l *singleTenantSeriesLimiter) Remove(count uint32) {
	l.activeSeries.Sub(count)
}
