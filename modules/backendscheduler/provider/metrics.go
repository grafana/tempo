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
)
