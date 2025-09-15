package livestore

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Instance-level metrics (similar to ingester instance.go)
	metricTracesCreatedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo_live_store",
		Name:      "traces_created_total",
		Help:      "The total number of traces created per tenant.",
	}, []string{"tenant"})
	metricLiveTraces = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo_live_store",
		Name:      "live_traces",
		Help:      "The current number of live traces per tenant.",
	}, []string{"tenant"})
	metricLiveTraceBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo_live_store",
		Name:      "live_trace_bytes",
		Help:      "The current number of bytes consumed by live traces per tenant.",
	}, []string{"tenant"})
	metricBytesReceivedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo_live_store",
		Name:      "bytes_received_total",
		Help:      "The total bytes received per tenant.",
	}, []string{"tenant", "data_type"})
	metricBlocksClearedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo_live_store",
		Name:      "blocks_cleared_total",
		Help:      "The total number of blocks cleared.",
	}, []string{"block_type"})
	metricCompletionSize = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "tempo_live_store",
		Name:      "completion_size_bytes",
		Help:      "Size in bytes of blocks completed.",
		Buckets:   prometheus.ExponentialBuckets(1024*1024, 2, 10), // from 1MB up to 1GB
	})
	metricBackPressure = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Subsystem: "live_store",
		Name:      "back_pressure_seconds_total",
		Help:      "The total amount of time spent waiting to process data from queue",
	}, []string{"reason"})

	metricFetchDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace:                   "tempo",
		Subsystem:                   "live_store",
		Name:                        "fetch_duration_seconds",
		Help:                        "Time spent fetching from Kafka.",
		NativeHistogramBucketFactor: 1.1,
	}, []string{"partition"})
	metricFetchBytesTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Subsystem: "live_store",
		Name:      "fetch_bytes_total",
		Help:      "Total number of bytes fetched from Kafka",
	}, []string{"partition"})
	metricFetchRecordsTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Subsystem: "live_store",
		Name:      "fetch_records_total",
		Help:      "Total number of records fetched from Kafka",
	}, []string{"partition"})
	metricConsumeCycleDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace:                   "tempo",
		Subsystem:                   "live_store",
		Name:                        "consume_cycle_duration_seconds",
		Help:                        "Time spent consuming a full cycle.",
		NativeHistogramBucketFactor: 1.1,
	})
	metricProcessPartitionDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace:                   "tempo",
		Subsystem:                   "live_store",
		Name:                        "process_partition_duration_seconds",
		Help:                        "Time spent processing partition data.",
		NativeHistogramBucketFactor: 1.1,
	}, []string{"partition"})
	metricFetchErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Subsystem: "live_store",
		Name:      "fetch_errors_total",
		Help:      "Total number of errors while fetching by the consumer.",
	}, []string{"partition"})
	metricOwnedPartitions = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Subsystem: "live_store",
		Name:      "owned_partitions",
		Help:      "Indicates partition ownership by this live store instance (1 = owned).",
	}, []string{"partition", "state"})
	metricRecordsProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Subsystem: "live_store",
		Name:      "kafka_records_processed_total",
		Help:      "The total number of kafka records processed per tenant.",
	}, []string{"tenant"})
	metricRecordsDropped = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Subsystem: "live_store",
		Name:      "kafka_records_dropped_total",
		Help:      "The total number of kafka records dropped per tenant.",
	}, []string{"tenant", "reason"})
)
