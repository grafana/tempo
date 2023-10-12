package localblocks

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace = "tempo"
	subsystem = "metrics_generator_processor_local_blocks"

	reasonLiveTracesExceeded = "live_traces_exceeded"
)

var (
	metricTotalTraces = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "traces_total",
		Help:      "Total number of traces created",
	}, []string{"tenant"})
	metricTotalSpans = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "spans_total",
		Help:      "Total number of spans after filtering",
	}, []string{"tenant"})
	metricLiveTraces = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "live_traces",
		Help:      "Number of live traces",
	}, []string{"tenant"})
	metricDroppedTraces = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "traces_dropped_total",
		Help:      "Number of traces dropped",
	}, []string{"tenant", "reason"})
	metricBlockSize = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "bytes",
		Help:      "Total size of local blocks",
	}, []string{"tenant"})
)
