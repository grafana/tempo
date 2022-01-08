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
	hedgedRequestsMetrics = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "tempodb",
			Name:      "backend_hedged_roundtrips_total",
			Help:      "Total number of hedged backend requests. Registered as a gauge for code sanity. This is a counter.",
		},
	)
)

// PublishHedgedMetrics flushes metrics from hedged requests every 10 seconds
func PublishHedgedMetrics(s *hedgedhttp.Stats) {
	ticker := time.NewTicker(hedgedMetricsPublishDuration)
	go func() {
		for range ticker.C {
			snap := s.Snapshot()
			hedgedRequests := int64(snap.ActualRoundTrips) - int64(snap.RequestedRoundTrips)
			if hedgedRequests < 0 {
				hedgedRequests = 0
			}
			hedgedRequestsMetrics.Set(float64(hedgedRequests))
		}
	}()
}
