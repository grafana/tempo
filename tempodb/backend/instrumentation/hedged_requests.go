package instrumentation

import (
	"github.com/cristalhq/hedgedhttp"
	"github.com/grafana/tempo/v2/pkg/hedgedmetrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var hedgedRequestsMetrics = promauto.NewGauge(
	prometheus.GaugeOpts{
		Namespace: "tempodb",
		Name:      "backend_hedged_roundtrips_total",
		Help:      "Total number of hedged backend requests. Registered as a gauge for code sanity. This is a counter.",
	},
)

// PublishHedgedMetrics flushes metrics from hedged requests every 10 seconds
func PublishHedgedMetrics(s *hedgedhttp.Stats) {
	hedgedmetrics.Publish(s, hedgedRequestsMetrics)
}
