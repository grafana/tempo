package backendworker

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricWorkerJobsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "backend_worker_jobs_total",
		Help:      "Total number of jobs processed",
	}, []string{})
	metricWorkerBadJobsRecieved = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "backend_worker_bad_jobs_received_total",
		Help:      "Total number of bad jobs received",
	}, []string{"status"})
	metricWorkerCallRetries = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "backend_worker_call_retries_total",
		Help:      "Total number of retries for calls",
	}, []string{})
)
