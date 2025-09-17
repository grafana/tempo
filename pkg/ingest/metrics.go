package ingest

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/backoff"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	labelGroup     = "group"
	labelPartition = "partition"
)

var (
	metricPartitionLag = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Subsystem: "ingest",
		Name:      "group_partition_lag",
		Help:      "Lag of a partition.",
	}, []string{labelGroup, labelPartition})

	metricPartitionLagSeconds = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Subsystem: "ingest",
		Name:      "group_partition_lag_seconds",
		Help:      "Lag of a partition in seconds.",
	}, []string{labelGroup, labelPartition})
)

type Metrics struct {
	TracesCreatedTotal       *prometheus.CounterVec
	LiveTraces               *prometheus.GaugeVec
	LiveTraceBytes           *prometheus.GaugeVec
	BytesReceivedTotal       *prometheus.CounterVec
	BlocksClearedTotal       *prometheus.CounterVec
	CompletionSize           prometheus.Histogram
	BackPressure             *prometheus.CounterVec
	FetchDuration            *prometheus.HistogramVec
	FetchBytesTotal          *prometheus.GaugeVec
	FetchRecordsTotal        *prometheus.GaugeVec
	ConsumeCycleDuration     prometheus.Histogram
	ProcessPartitionDuration *prometheus.HistogramVec
	FetchErrors              *prometheus.CounterVec
	RecordsProcessed         *prometheus.CounterVec
	RecordsDropped           *prometheus.CounterVec
}

func NewMetrics(subsystem string, reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		TracesCreatedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Subsystem: subsystem,
			Name:      "traces_created_total",
			Help:      "The total number of traces created per tenant.",
		}, []string{"tenant"}),
		LiveTraces: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "tempo",
			Subsystem: subsystem,
			Name:      "live_traces",
			Help:      "The current number of live traces per tenant.",
		}, []string{"tenant"}),
		LiveTraceBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "tempo",
			Subsystem: subsystem,
			Name:      "live_trace_bytes",
			Help:      "The current number of bytes consumed by live traces per tenant.",
		}, []string{"tenant"}),
		BytesReceivedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Subsystem: subsystem,
			Name:      "bytes_received_total",
			Help:      "The total bytes received per tenant.",
		}, []string{"tenant", "data_type"}),
		BlocksClearedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Subsystem: subsystem,
			Name:      "blocks_cleared_total",
			Help:      "The total number of blocks cleared.",
		}, []string{"block_type"}),
		CompletionSize: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "tempo",
			Subsystem: subsystem,
			Name:      "completion_size_bytes",
			Help:      "Size in bytes of blocks completed.",
			Buckets:   prometheus.ExponentialBuckets(1024*1024, 2, 10), // from 1MB up to 1GB
		}),
		BackPressure: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Subsystem: subsystem,
			Name:      "back_pressure_seconds_total",
			Help:      "The total amount of time spent waiting to process data from queue",
		}, []string{"reason"}),
		FetchDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace:                   "tempo",
			Subsystem:                   subsystem,
			Name:                        "fetch_duration_seconds",
			Help:                        "Time spent fetching from Kafka.",
			NativeHistogramBucketFactor: 1.1,
		}, []string{"partition"}),
		FetchBytesTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "tempo",
			Subsystem: subsystem,
			Name:      "fetch_bytes_total",
			Help:      "Total number of bytes fetched from Kafka",
		}, []string{"partition"}),
		FetchRecordsTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "tempo",
			Subsystem: subsystem,
			Name:      "fetch_records_total",
			Help:      "Total number of records fetched from Kafka",
		}, []string{"partition"}),
		ConsumeCycleDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace:                   "tempo",
			Subsystem:                   subsystem,
			Name:                        "consume_cycle_duration_seconds",
			Help:                        "Time spent consuming a full cycle.",
			NativeHistogramBucketFactor: 1.1,
		}),
		ProcessPartitionDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace:                   "tempo",
			Subsystem:                   subsystem,
			Name:                        "process_partition_duration_seconds",
			Help:                        "Time spent processing partition data.",
			NativeHistogramBucketFactor: 1.1,
		}, []string{"partition"}),
		FetchErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Subsystem: subsystem,
			Name:      "fetch_errors_total",
			Help:      "Total number of errors while fetching by the consumer.",
		}, []string{"partition"}),
		RecordsProcessed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Subsystem: subsystem,
			Name:      "kafka_records_processed_total",
			Help:      "The total number of kafka records processed per tenant.",
		}, []string{"tenant"}),
		RecordsDropped: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Subsystem: subsystem,
			Name:      "kafka_records_dropped_total",
			Help:      "The total number of kafka records dropped per tenant.",
		}, []string{"tenant", "reason"}),
	}
	reg.MustRegister(m.BackPressure, m.BlocksClearedTotal, m.BytesReceivedTotal, m.CompletionSize, m.ConsumeCycleDuration, m.FetchBytesTotal, m.FetchDuration, m.FetchErrors, m.FetchRecordsTotal, m.LiveTraceBytes, m.LiveTraces, m.ProcessPartitionDuration, m.RecordsDropped, m.RecordsProcessed, m.TracesCreatedTotal)
	return m
}

// ExportPartitionLagMetrics in a background goroutine by periodically querying Kafka state
// for the assigned and active partitions.  This exports the lag metric in number of records
// which is different than the lag metric for age.
// Call ResetLagMetricsForRevokedPartitions when partitions are revoked to prevent exporting
// stale data. For efficiency this is not detected automatically from changes inthe assigned
// partition callback.
func ExportPartitionLagMetrics(ctx context.Context, kclient *kgo.Client, log log.Logger, cfg Config, getAssignedActivePartitions func() []int32, forceMetadataRefresh func()) {
	go func() {
		var (
			waitTime = cfg.Kafka.ConsumerGroupLagMetricUpdateInterval
			topic    = cfg.Kafka.Topic
			group    = cfg.Kafka.ConsumerGroup
			boff     = backoff.New(ctx, backoff.Config{
				MinBackoff: 100 * time.Millisecond,
				MaxBackoff: waitTime,
				MaxRetries: 5,
			})
			admClient       = kadm.NewClient(kclient)
			partitionClient = NewPartitionOffsetClient(kclient, topic)
		)

		for {
			select {
			case <-time.After(waitTime):
				var (
					lag kadm.GroupLag
					err error
				)
				assignedPartitions := getAssignedActivePartitions()
				boff.Reset()
				for boff.Ongoing() {
					lag, err = getGroupLag(ctx, admClient, partitionClient, group, assignedPartitions)
					if err == nil {
						break
					}
					HandleKafkaError(err, forceMetadataRefresh)
					boff.Wait()
				}

				if err != nil {
					level.Error(log).Log("msg", "metric lag failed:", "err", err, "retries", boff.NumRetries())
					continue
				}
				for _, p := range assignedPartitions {
					l, ok := lag.Lookup(topic, p)
					if ok {
						metricPartitionLag.WithLabelValues(group, strconv.Itoa(int(p))).Set(float64(l.Lag))
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

// SetPartitionLagSeconds is similar to the auto exported lag, except it is in real clock seconds
// which can only be known after the record is read from the queue, therefore it is set by the caller.
// Call ResetLagMetricsForRevokedPartitions when partitions are revoked to prevent exporting stale data.
func SetPartitionLagSeconds(group string, partition int32, lag time.Duration) {
	metricPartitionLagSeconds.WithLabelValues(group, strconv.Itoa(int(partition))).Set(lag.Seconds())
}

// ResetLagMetricsForRevokedPartitions should be called when a partition is revoked to prevent
// exporting stale metrics for partitions that the application no longer owns.
func ResetLagMetricsForRevokedPartitions(group string, partitions []int32) {
	for _, p := range partitions {
		l := strconv.Itoa(int(p))
		metricPartitionLag.DeletePartialMatch(prometheus.Labels{labelGroup: group, labelPartition: l})
		metricPartitionLagSeconds.DeletePartialMatch(prometheus.Labels{labelGroup: group, labelPartition: l})
	}
}

// getGroupLag is similar to `kadm.Client.Lag` but works when the group doesn't have live participants.
// Similar to `kadm.CalculateGroupLagWithStartOffsets`, it takes into account that the group may not have any commits.
//
// The lag is the difference between the last produced offset (high watermark) and an offset in the "past".
// If the block builder committed an offset for a given partition to the consumer group at least once, then
// the lag is the difference between the last produced offset and the offset committed in the consumer group.
// Otherwise, if the block builder didn't commit an offset for a given partition yet (e.g. block builder is
// running for the first time), then the lag is the difference between the last produced offset and fallbackOffsetMillis.
func getGroupLag(ctx context.Context, admClient *kadm.Client, partitionClient *PartitionOffsetClient, group string, assignedPartitions []int32) (kadm.GroupLag, error) {
	offsets, err := admClient.FetchOffsets(ctx, group)
	if err != nil {
		if !errors.Is(err, kerr.GroupIDNotFound) {
			return nil, fmt.Errorf("fetch offsets: %w", err)
		}
	}
	if err := offsets.Error(); err != nil {
		return nil, fmt.Errorf("fetch offsets got error in response: %w", err)
	}

	startOffsets, err := partitionClient.FetchPartitionsStartProducedOffsets(ctx, assignedPartitions)
	if err != nil {
		return nil, err
	}
	endOffsets, err := partitionClient.FetchPartitionsLastProducedOffsets(ctx, assignedPartitions)
	if err != nil {
		return nil, err
	}

	descrGroup := kadm.DescribedGroup{
		// "Empty" is the state that indicates that the group doesn't have active consumer members; this is always the case for block-builder,
		// because we don't use group consumption.
		State: "Empty",
	}
	return kadm.CalculateGroupLagWithStartOffsets(descrGroup, offsets, startOffsets, endOffsets), nil
}
