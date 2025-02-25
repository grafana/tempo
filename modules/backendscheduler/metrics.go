package backendscheduler

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type metrics struct {
	// schedulerIsLeader prometheus.Gauge
	schedulingCycles *prometheus.CounterVec
	jobsCreated      *prometheus.CounterVec
	jobsCompleted    *prometheus.CounterVec
	jobsFailed       *prometheus.CounterVec
	jobsActive       *prometheus.GaugeVec
}

func newMetrics(reg prometheus.Registerer) *metrics {
	p := promauto.With(reg)

	return &metrics{
		// schedulerIsLeader: p.NewGauge(prometheus.GaugeOpts{
		// 	Namespace: "tempo",
		// 	Name:      "backend_scheduler_is_leader",
		// 	Help:      "1 if this instance is the leader, 0 otherwise",
		// }),
		schedulingCycles: p.NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "backend_scheduler_scheduling_cycles_total",
			Help:      "Total number of scheduling cycles run",
		}, []string{"status"}),
		jobsCreated: p.NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "backend_scheduler_jobs_created_total",
			Help:      "Total number of jobs created",
		}, []string{"tenant", "job_type"}),
		jobsCompleted: p.NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "backend_scheduler_jobs_completed_total",
			Help:      "Total number of jobs completed",
		}, []string{"tenant", "job_type"}),
		jobsFailed: p.NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "backend_scheduler_jobs_failed_total",
			Help:      "Total number of jobs that failed",
		}, []string{"tenant", "job_type"}),
		jobsActive: p.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "tempo",
			Name:      "backend_scheduler_jobs_active",
			Help:      "Number of currently active jobs",
		}, []string{"tenant", "job_type"}),
	}
}
