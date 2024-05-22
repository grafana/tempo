package localblocks

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace = "tempo"
	subsystem = "metrics_generator_processor_local_blocks"

	reasonLiveTracesExceeded = "live_traces_exceeded"
	reasonTraceSizeExceeded  = "trace_too_large"
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
	metricDroppedSpans = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "spans_dropped_total",
		Help:      "Number of spans dropped",
	}, []string{"tenant", "reason"})
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
	metricCompletedBlocks = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "completed_blocks",
		Help:      "Number of blocks completed by the local blocks processor",
	}, []string{"tenant"})
	metricCutBlocks = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "cut_blocks",
		Help:      "Number of blocks cut by the local blocks processor",
	}, []string{"tenant"})
	metricFlushedBlocks = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "flushed_blocks",
		Help:      "Number of blocks flushed by the local blocks processor",
	}, []string{"tenant"})
	metricFlushQueueSize = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "flush_queue_size",
		Help:      "Size of the flush queue",
	}, []string{"tenant"})
)
