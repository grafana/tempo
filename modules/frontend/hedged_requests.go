package frontend

import (
	"net/http"
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
			Namespace: "tempo",
			Name:      "query_frontend_hedged_roundtrips_total",
			Help:      "Total number of hedged search requests. Registered as a gauge for code sanity. This is a counter.",
		},
	)
)

func newHedgedRequestWare(hedgeRequestsAt time.Duration, hedgeRequestsUpTo int) Middleware {
	return MiddlewareFunc(func(r http.RoundTripper) http.RoundTripper {
		if hedgeRequestsAt == 0 || hedgeRequestsUpTo == 0 {
			return r
		}
		ret, stats, err := hedgedhttp.NewRoundTripperAndStats(hedgeRequestsAt, hedgeRequestsUpTo, r)
		if err != nil {
			panic(err)
		}
		publishHedgedMetrics(stats)
		return ret
	})
}

// PublishHedgedMetrics flushes metrics from hedged requests every 10 seconds
func publishHedgedMetrics(s *hedgedhttp.Stats) {
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
