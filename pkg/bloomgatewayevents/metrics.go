package bloomgatewayevents

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	metricsNamespace = "tempo"
	metricsSubsystem = "bloom_gateway"
)

// metrics is the producer-side half of DESIGN.md's § Metrics table (the
// consumer/gateway-side series live in modules/bloomgateway/metrics.go).
// Built by newMetrics(reg), not package-level promauto vars, for the same
// reason as the consumer side: every producer (block-builder,
// backend-worker) constructs its own instance against its own Registerer,
// and multiple can exist in one process.
type metrics struct {
	// publishesTotal counts every publish attempt by result: "ok" (acked),
	// "dropped" (retry budget exhausted -- pruning for that block is
	// deferred to § Reconciliation, never a correctness issue), and
	// "rate_limited" (rejected by the per-tenant publish rate limit before
	// ever reaching Kafka, DESIGN.md § Multi-tenant cells's producer-side
	// guardrail). Retries themselves occur inside the client's bounded
	// delivery budget and are not separately observable.
	publishesTotal *prometheus.CounterVec

	// publishDurationSeconds is the per-publish latency: a per-event
	// hot-path timing, like the consumer side's addApplyDurationSeconds,
	// hence the same native-histogram settings.
	publishDurationSeconds prometheus.Histogram

	// invalidTraceIDsTotal counts trace IDs dropped by PublishAdd's
	// defensive filter for being outside the valid 1..16 byte range
	// (modules/bloomgateway/events.go's validateTraceIDs drops a WHOLE
	// chunk over a single such ID, so producer-side filtering keeps one bad
	// ID from poisoning every valid ID that would otherwise share its
	// chunk). Never logged alongside the offending bytes -- only ever
	// counted.
	invalidTraceIDsTotal prometheus.Counter
}

// newMetrics registers this producer's series against reg and returns them.
// Namespace/Subsystem "tempo"/"bloom_gateway" so the registered names are
// tempo_bloom_gateway_publishes_total and
// tempo_bloom_gateway_publish_duration_seconds, matching DESIGN.md's §
// Metrics producer-side names exactly.
func newMetrics(reg prometheus.Registerer) *metrics {
	return &metrics{
		publishesTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "publishes_total",
			Help:      "Total bloom-gateway event publish attempts, by result (ok|dropped|rate_limited).",
		}, []string{"result"}),
		publishDurationSeconds: promauto.With(reg).NewHistogram(prometheus.HistogramOpts{
			Namespace:                       metricsNamespace,
			Subsystem:                       metricsSubsystem,
			Name:                            "publish_duration_seconds",
			Help:                            "Duration of publishing one bloom-gateway event to Kafka.",
			Buckets:                         prometheus.DefBuckets,
			NativeHistogramBucketFactor:     1.1,
			NativeHistogramMaxBucketNumber:  100,
			NativeHistogramMinResetDuration: time.Hour,
		}),
		invalidTraceIDsTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "invalid_trace_ids_total",
			Help:      "Total trace IDs dropped by the producer for being outside the valid 1..16 byte range.",
		}),
	}
}
