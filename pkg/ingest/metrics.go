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

// ExportPartitionLagMetrics in a background goroutine by periodically querying Kafka state
// for the assigned and active partitions.  This exports the lag metric in number of records
// which is different than the lag metric for age.
// Each tick also reconciles the exported series against the current assigned-partition set and
// deletes series for partitions no longer assigned to this consumer. This is a safety net for
// handoffs that do not deliver a clean OnPartitionsRevoked (e.g. the round-robin distribution of
// inactive partitions in cooperativeActiveStickyBalancer), which otherwise leaves stale,
// ever-growing lag on the previous owner. Callers may still call ResetLagMetricsForRevokedPartitions
// on revoke for an immediate cleanup rather than waiting for the next tick.
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

			// trackedPartitions is the set of partitions for which we have exported lag
			// metrics, used to prune series for partitions that are no longer assigned to
			// this consumer so stale lag can't accumulate and trip alerts.
			trackedPartitions = make(map[int32]struct{})
		)

		for {
			select {
			case <-time.After(waitTime):
				var (
					lag kadm.GroupLag
					err error
				)
				assignedPartitions := getAssignedActivePartitions()

				// Prune series for partitions moved off this consumer since the last tick, and
				// record the current assigned set as tracked. This runs before the lag fetch so
				// ownership changes are still reconciled even if getGroupLag fails for several ticks.
				pruneUnassignedLagMetrics(group, assignedPartitions, trackedPartitions)

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

// pruneUnassignedLagMetrics deletes exported lag series for partitions in tracked that are no
// longer present in assignedPartitions, then records assignedPartitions as the new tracked set.
// tracked is mutated in place. This reconciles the exported metrics with actual ownership each
// tick, so a partition moved to another consumer stops exporting stale lag even if no clean
// OnPartitionsRevoked was delivered for it.
func pruneUnassignedLagMetrics(group string, assignedPartitions []int32, tracked map[int32]struct{}) {
	assigned := make(map[int32]struct{}, len(assignedPartitions))
	for _, p := range assignedPartitions {
		assigned[p] = struct{}{}
	}
	for p := range tracked {
		if _, ok := assigned[p]; !ok {
			ResetLagMetricsForRevokedPartitions(group, []int32{p})
			delete(tracked, p)
		}
	}
	for _, p := range assignedPartitions {
		tracked[p] = struct{}{}
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
