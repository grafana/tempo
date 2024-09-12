package hedgedmetrics

import (
	"time"

	"github.com/cristalhq/hedgedhttp"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	hedgedMetricsPublishDuration = 10 * time.Second
)

// PublishHedgedMetrics flushes metrics from hedged requests every 10 seconds
func Publish(s *hedgedhttp.Stats, counter prometheus.Counter) {
	ticker := time.NewTicker(hedgedMetricsPublishDuration)
	go func() {
		for range ticker.C {
			snap := s.Snapshot()
			hedgedRequests := int64(snap.ActualRoundTrips) - int64(snap.RequestedRoundTrips)
			if hedgedRequests < 0 {
				hedgedRequests = 0
			}
			counter.Add(float64(hedgedRequests))
		}
	}()
}
