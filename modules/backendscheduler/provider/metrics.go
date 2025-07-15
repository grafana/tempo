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
	metricEmptyTenantCycle = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo_backend_scheduler",
		Name:      "compaction_empty_tenant_cycle_total",
		Help:      "The number of compaction cycles where no tenant had work available",
	})
	metricTenantEmptyJob = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo_backend_scheduler",
		Name:      "compaction_tenant_empty_job_total",
		Help:      "The number of times an empty job was received from the priority queue",
	})
)
