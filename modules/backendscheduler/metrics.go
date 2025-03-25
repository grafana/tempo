package backendscheduler

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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
		Name:      "backend_scheduler_jobs_retry",
		Help:      "The number of jobs which have been retried",
	}, []string{"tenant", "job_type", "worker_id"})
)
