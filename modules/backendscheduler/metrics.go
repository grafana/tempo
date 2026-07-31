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
		Namespace:                       "tempo",
		Name:                            "backend_scheduler_job_duration_seconds",
		Help:                            "Duration of of a job in seconds",
		Buckets:                         prometheus.DefBuckets,
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
	if mode == tempopb.RedactionMode_REDACTION_MODE_DRY_RUN {
		return "dry_run"
	}
	return "apply"
}
