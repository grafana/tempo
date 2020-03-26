package main

import "github.com/prometheus/client_golang/prometheus"

const (
	namespace = "tempo_vulture"
)

var (
	// metricsErrorTotal is a prometheus counter that indicates the total number of unexpected errors encountered by the application
	metricErrorTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "error_total",
			Help:      "tempo vulture errors",
		},
	)

	// metricTracesInspected is a prometheus gauge that indicates the number of seconds until certificates on disk expires.
	metricTracesInspected = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "trace_total",
			Help:      "total number of traces inspected by tempo vulture",
		},
		[]string{"secondsago"},
	)

	// metricTracesInspected is a prometheus gauge that indicates the number of seconds until certificates on disk expires.
	metricTracesErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "trace_error_total",
			Help:      "total number of issues with traces",
		},
		[]string{"error", "secondsago"},
	)
)

func init() {
	prometheus.MustRegister(metricErrorTotal)
	prometheus.MustRegister(metricTracesInspected)
	prometheus.MustRegister(metricTracesErrors)
}
