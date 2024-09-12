package instrumentation

import (
	"github.com/cristalhq/hedgedhttp"
	"github.com/grafana/tempo/pkg/hedgedmetrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	hedgedRequestsMetrics = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "tempodb",
			Name:      "backend_hedged_roundtrips_total",
			Help:      "Total number of hedged backend requests.",
		},
	)
	hedgedRequestsMetricsWithValue = hedgedmetrics.NewCounterWithValue(hedgedRequestsMetrics)
)

// PublishHedgedMetrics flushes metrics from hedged requests every 10 seconds
func PublishHedgedMetrics(s *hedgedhttp.Stats) {
	hedgedmetrics.Publish(s, hedgedRequestsMetricsWithValue, hedgedmetrics.PublishDuration)
}
