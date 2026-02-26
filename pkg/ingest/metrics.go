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
					lag, err = getGroupLag(ctx, admClient, partitionClient, group, topic, assignedPartitions)
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
//
// When the group has current members (DescribeGroups), we use CalculateGroupLagWithStartOffsets so lag is
// computed for each member's assignment. When the group has no members (e.g. rebalancing, or block-builder
// which does not use consumer groups), we compute lag manually for (topic, assignedPartitions) so that
// newly assigned partitions (e.g. after a rebalance) report correct lag.
func getGroupLag(ctx context.Context, admClient *kadm.Client, partitionClient *PartitionOffsetClient, group, topic string, assignedPartitions []int32) (kadm.GroupLag, error) {
	if len(assignedPartitions) == 0 {
		return kadm.GroupLag{}, nil
	}
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

	// Prefer the real group description so lag is computed for all members' assignments.
	if described, err := admClient.DescribeGroups(ctx, group); err == nil {
		if g, ok := described[group]; ok && len(g.Members) > 0 {
			return kadm.CalculateGroupLagWithStartOffsets(g, offsets, startOffsets, endOffsets), nil
		}
	}

	// No members (rebalancing, or group not using consumer protocol). Compute lag for our assigned partitions
	// so that when assignment lands on this instance (e.g. after abandoned partitions time out), lag is correct.
	tcommit := offsets[topic]
	tstart := startOffsets[topic]
	tend := endOffsets[topic]
	if tend == nil {
		return kadm.GroupLag{}, nil
	}
	result := make(kadm.GroupLag)
	result[topic] = make(map[int32]kadm.GroupMemberLag)
	for _, p := range assignedPartitions {
		pend, ok := tend[p]
		if !ok || pend.Err != nil {
			continue
		}
		lag := int64(-1)
		if tcommit != nil {
			if pcommit, ok := tcommit[p]; ok && pcommit.Err == nil && pcommit.At >= 0 {
				lag = pend.Offset - pcommit.At
			}
		}
		if lag < 0 && tstart != nil {
			if pstart, ok := tstart[p]; ok && pstart.Err == nil {
				lag = pend.Offset - pstart.Offset
			}
		}
		if lag < 0 {
			lag = 0
		}
		result[topic][p] = kadm.GroupMemberLag{Topic: topic, Partition: p, Lag: lag}
	}
	return result, nil
}
