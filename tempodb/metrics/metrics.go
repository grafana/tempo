package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	MetricCompactionBlocks = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempodb",
		Name:      "compaction_blocks_total",
		Help:      "Total number of blocks compacted.",
	}, []string{"level"})
	MetricCompactionObjectsWritten = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempodb",
		Name:      "compaction_objects_written_total",
		Help:      "Total number of objects written to backend during compaction.",
	}, []string{"level"})
	MetricCompactionBytesWritten = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempodb",
		Name:      "compaction_bytes_written_total",
		Help:      "Total number of bytes written to backend during compaction.",
	}, []string{"level"})
	MetricCompactionErrors = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempodb",
		Name:      "compaction_errors_total",
		Help:      "Total number of errors occurring during compaction.",
	})
	MetricCompactionObjectsCombined = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempodb",
		Name:      "compaction_objects_combined_total",
		Help:      "Total number of objects combined during compaction.",
	}, []string{"level"})
	MetricCompactionOutstandingBlocks = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempodb",
		Name:      "compaction_outstanding_blocks",
		Help:      "Number of blocks remaining to be compacted before next maintenance cycle",
	}, []string{"tenant"})
)
