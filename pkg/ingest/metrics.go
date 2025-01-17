package ingest

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
)

var metricPartitionLag = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: "tempo",
	Subsystem: "ingest",
	Name:      "group_partition_lag",
	Help:      "Lag of a partition.",
}, []string{"group", "partition"})

// TODO - Simplify signature to create client instead?
func ExportPartitionLagMetrics(ctx context.Context, admClient *kadm.Client, log log.Logger, cfg Config, getAssignedActivePartitions func() []int32) {
	go func() {
		var (
			waitTime = time.Second * 15
			topic    = cfg.Kafka.Topic
			group    = cfg.Kafka.ConsumerGroup
		)

		for {
			select {
			case <-time.After(waitTime):
				lag, err := getGroupLag(ctx, admClient, topic, group)
				if err != nil {
					level.Error(log).Log("msg", "metric lag failed:", "err", err)
					continue
				}
				for _, p := range getAssignedActivePartitions() {
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

// getGroupLag is similar to `kadm.Client.Lag` but works when the group doesn't have live participants.
// Similar to `kadm.CalculateGroupLagWithStartOffsets`, it takes into account that the group may not have any commits.
//
// The lag is the difference between the last produced offset (high watermark) and an offset in the "past".
// If the block builder committed an offset for a given partition to the consumer group at least once, then
// the lag is the difference between the last produced offset and the offset committed in the consumer group.
// Otherwise, if the block builder didn't commit an offset for a given partition yet (e.g. block builder is
// running for the first time), then the lag is the difference between the last produced offset and fallbackOffsetMillis.
func getGroupLag(ctx context.Context, admClient *kadm.Client, topic, group string) (kadm.GroupLag, error) {
	offsets, err := admClient.FetchOffsets(ctx, group)
	if err != nil {
		if !errors.Is(err, kerr.GroupIDNotFound) {
			return nil, fmt.Errorf("fetch offsets: %w", err)
		}
	}
	if err := offsets.Error(); err != nil {
		return nil, fmt.Errorf("fetch offsets got error in response: %w", err)
	}

	startOffsets, err := admClient.ListStartOffsets(ctx, topic)
	if err != nil {
		return nil, err
	}
	endOffsets, err := admClient.ListEndOffsets(ctx, topic)
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
