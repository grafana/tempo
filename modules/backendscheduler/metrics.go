package backendscheduler

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/grafana/tempo/pkg/tempopb"
)

var (
	metricJobsCreated = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "backend_scheduler_jobs_created_total",
		Help:      "Total number of jobs created",
	}, []string{"tenant", "job_type"})
	metricJobsCompleted = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "backend_scheduler_jobs_completed_total",
		Help:      "Total number of jobs completed",
	}, []string{"tenant", "job_type"})
	metricJobsFailed = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "backend_scheduler_jobs_failed_total",
		Help:      "Total number of jobs that failed",
	}, []string{"tenant", "job_type"})
	metricJobsActive = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Name:      "backend_scheduler_jobs_active",
		Help:      "Number of currently active jobs",
	}, []string{"tenant", "job_type"})
	metricJobsPending = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Name:      "backend_scheduler_jobs_pending",
		Help:      "Number of jobs enqueued and not yet dispatched to a worker, by tenant and type. This is queue depth: jobs_active counts work already handed out, so it is bounded by the worker count and cannot indicate that more capacity is needed.",
	}, []string{"tenant", "job_type"})
	metricJobsRetry = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "backend_scheduler_jobs_retry_total",
		Help:      "The number of jobs which have been retried",
	}, []string{"tenant", "job_type", "worker_id"})
	metricJobsNotFound = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "backend_scheduler_jobs_not_found_total",
		Help:      "The number of calls to get a job that were not found",
	}, []string{"worker_id"})
	metricJobsDropped = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "backend_scheduler_jobs_dropped_total",
		Help:      "Total number of jobs dropped at assignment time because preconditions were no longer met",
	}, []string{"tenant", "job_type"})
	metricProviderJobsMerged = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "backend_scheduler_provider_jobs_merged_total",
		Help:      "The number of jobs merged from providers",
	}, []string{"id"})
	metricWorkFlushesFailed = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "backend_scheduler_work_flushes_failed_total",
		Help:      "The number of times the work cache flush to backend storage failed",
	})
	metricWorkFlushes = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "backend_scheduler_work_flushes_total",
		Help:      "The number of times the work cache was flushed to backend storage",
	})
	metricWorkCacheFileSize = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace:                       "tempo",
		Name:                            "backend_scheduler_work_cache_file_size_bytes",
		Help:                            "Size of the work cache file in bytes",
		Buckets:                         prometheus.ExponentialBuckets(1024, 2, 16), // 1KB to 32MB
		NativeHistogramBucketFactor:     1.1,
		NativeHistogramMaxBucketNumber:  100,
		NativeHistogramMinResetDuration: 1 * time.Hour,
	})
	metricJobDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "tempo",
		Name:      "backend_scheduler_job_duration_seconds",
		Help:      "Duration of a job in seconds",
		// DefBuckets stops at 10s and puts nothing between 2.5s and 5s, where roughly 60% of
		// redaction jobs land — every percentile drawn from it interpolates inside one bucket, so
		// p50, p90 and p99 move together and say nothing. It also cannot represent a job slower
		// than 10s at all, and a redaction over a large block runs for minutes.
		//
		// Powers of two from 10ms to ~11m: fast retention jobs stay resolved, the redaction mass
		// splits across the 2.56s and 5.12s boundaries, and long jobs land in a real bucket
		// instead of +Inf.
		Buckets:                         prometheus.ExponentialBuckets(0.01, 2, 17),
		NativeHistogramBucketFactor:     1.1,
		NativeHistogramMaxBucketNumber:  100,
		NativeHistogramMinResetDuration: 1 * time.Hour,
	}, []string{"job_type"})
	metricRedactionTracesFound = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "backend_scheduler_redaction_traces_found_total",
		Help:      "Total traces matched by redaction jobs, by tenant and mode. mode=apply counts traces actually removed; mode=dry_run counts previewed blast radius (nothing removed).",
	}, []string{"tenant", "mode"})
)

// recordRedactionResult records a completed redaction job's match count on the per-tenant,
// per-mode counter. A zero/empty result (block scanned clean) is a no-op.
func recordRedactionResult(tenant string, mode tempopb.RedactionMode, found int32) {
	if found <= 0 {
		return
	}
	metricRedactionTracesFound.WithLabelValues(tenant, redactionModeLabel(mode)).Add(float64(found))
}

// redactionModeLabel is a short, stable metric label for a redaction mode.
func redactionModeLabel(mode tempopb.RedactionMode) string {
	if mode.IsDryRun() {
		return "dry_run"
	}
	return "apply"
}

// recordPendingJobs publishes the queue depth per tenant and job type.
//
// Reset first: this is a gauge keyed by tenant, and a tenant whose queue drains stops appearing in
// the snapshot entirely. Without the reset its last non-zero value would persist forever, so a
// finished redaction would look permanently backlogged — and an autoscaler reading it would never
// scale back down.
func (s *BackendScheduler) recordPendingJobs() {
	metricJobsPending.Reset()

	for tenant, byType := range s.work.PendingJobCounts() {
		for jobType, n := range byType {
			metricJobsPending.WithLabelValues(tenant, jobType.String()).Set(float64(n))
		}
	}
}
