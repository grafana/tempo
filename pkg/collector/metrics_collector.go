package collector

import (
	"go.uber.org/atomic"
)

// MetricsCollector is a thread-safe collector that uses atomic operations
// to accumulate metrics. We primarily use it to collect the total bytes read from
// a reader across a request
type MetricsCollector struct {
	totalValue *atomic.Uint64
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		totalValue: atomic.NewUint64(0),
	}
}

// Add adds new bytes read to TotalValue. this method is thread safe and
// satisfies the common.MetricsCallback type so it's used as callback at a lot of places
func (mc *MetricsCollector) Add(value uint64) {
	mc.totalValue.Add(value)
}

// TotalValue returns the sum of total values collected by the collector
func (mc *MetricsCollector) TotalValue() uint64 {
	return mc.totalValue.Load()
}
