package instrumentation

import (
	"time"

	"github.com/cristalhq/hedgedhttp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	hedgedMetricsPublishDuration = 10 * time.Second
)

var (
	hedgedRequestsMetrics = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "tempodb",
			Name:      "backend_hedged_roundtrips_total",
			Help:      "Total number of hedged backend requests",
		},
	)
)

// PublishHedgedMetrics flushes metrics from hedged requests every 10 seconds
func PublishHedgedMetrics(s *hedgedhttp.Stats) {
	ticker := time.NewTicker(hedgedMetricsPublishDuration)
	go func() {
		for range ticker.C {
			hedgedRequestsMetrics.Add(float64(s.Snapshot().RequestedRoundTrips))
		}
	}()
}
