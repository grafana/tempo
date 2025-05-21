package provider

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricJobsCreated = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo_backend_scheduler",
		Name:      "compaction_jobs_created_total",
		Help:      "Total number of compaction jobs created",
	}, []string{"tenant"})
	metricTenantReset = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo_backend_scheduler",
		Name:      "compaction_tenant_reset_total",
		Help:      "The number of times the tenant is changed",
	}, []string{"tenant"})
	metricTenantBackoff = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo_backend_scheduler",
		Name:      "compaction_tenant_backoff_total",
		Help:      "The number of times the backoff is triggered",
	})
	metricTenantEmptyJob = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo_backend_scheduler",
		Name:      "compaction_tenant_empty_job_total",
		Help:      "The number of times an empty job was received from the priority queue",
	})
)
