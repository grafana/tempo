package blocklist

import (
	"context"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel/codes"
)

const pollerMetricNamespace = "tempodb"

// Tenant IDs belong in logs and traces, not in these histogram labels. Stage
// names and outcomes are fixed at the call sites.
var (
	metricPollStageDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace:                       pollerMetricNamespace,
		Name:                            "blocklist_poll_stage_duration_seconds",
		Help:                            "Time spent in each tenant polling stage, including failed attempts.",
		Buckets:                         prometheus.ExponentialBuckets(0.01, 4, 10),
		NativeHistogramBucketFactor:     1.1,
		NativeHistogramMaxBucketNumber:  100,
		NativeHistogramMinResetDuration: time.Hour,
	}, []string{"stage", "outcome"})
	metricTenantQueueDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace:                       pollerMetricNamespace,
		Name:                            "blocklist_poll_tenant_queue_duration_seconds",
		Help:                            "Time from completing tenant discovery to starting a tenant poll, including waiting behind other tenants. Only started tenants are observed.",
		Buckets:                         prometheus.ExponentialBuckets(0.01, 4, 10),
		NativeHistogramBucketFactor:     1.1,
		NativeHistogramMaxBucketNumber:  100,
		NativeHistogramMinResetDuration: time.Hour,
	})
	metricActiveTenantPolls = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: pollerMetricNamespace,
		Name:      "blocklist_poll_tenants_active",
		Help:      "Number of tenant polling workers currently running, including retries.",
	})
)

type (
	pollCycleKey struct{}
	pollQueueKey struct{}
)

func (p *Poller) tenantLogger(ctx context.Context, tenant string) log.Logger {
	logger := log.With(p.logger, "tenant", tenant)
	if cycle, ok := ctx.Value(pollCycleKey{}).(string); ok {
		logger = log.With(logger, "poll_cycle", cycle)
	}
	if queued, ok := ctx.Value(pollQueueKey{}).(time.Duration); ok {
		logger = log.With(logger, "queue_seconds", queued.Seconds())
	}
	return logger
}

// startPollStage measures wall time, including backend retries and any bounded
// concurrency waits within the stage. It does not change error handling.
func startPollStage(ctx context.Context, stage string) (context.Context, func(error) time.Duration) {
	ctx, span := tracer.Start(ctx, "Poller."+stage)
	start := time.Now()
	return ctx, func(err error) time.Duration {
		duration := time.Since(start)
		outcome := "success"
		if err != nil {
			outcome = "error"
			span.RecordError(err)
			span.SetStatus(codes.Error, "polling stage failed")
		}
		metricPollStageDuration.WithLabelValues(stage, outcome).Observe(duration.Seconds())
		span.End()
		return duration
	}
}
