package generator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/tempopb"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
)

func (g *Generator) startKafka() {
	// Create context that will be used to stop the goroutines.
	var ctx context.Context
	ctx, g.kafkaStop = context.WithCancel(context.Background())

	g.kafkaWG.Add(1)
	go g.listenKafka(ctx)
	ingest.ExportPartitionLagMetrics(ctx, g.kafkaAdm, g.logger, g.cfg.Ingest, g.getAssignedActivePartitions)
}

func (g *Generator) stopKafka() {
	g.kafkaStop()
	g.kafkaWG.Wait()
}

func (g *Generator) listenKafka(ctx context.Context) {
	defer g.kafkaWG.Done()

	level.Info(g.logger).Log("msg", "generator now listening to kafka")
	for {
		select {
		case <-time.After(2 * time.Second):
			if g.readOnly.Load() {
				// Starting up or shutting down
				continue
			}
			err := g.readKafka(ctx)
			if err != nil {
				level.Error(g.logger).Log("msg", "readKafka failed", "err", err)
				continue
			}
		case <-ctx.Done():
			return
		}
	}
}

func (g *Generator) readKafka(ctx context.Context) error {
	fallback := time.Now().Add(-time.Minute)

	groupLag, err := getGroupLag(
		ctx,
		kadm.NewClient(g.kafkaClient),
		g.cfg.Ingest.Kafka.Topic,
		g.cfg.Ingest.Kafka.ConsumerGroup,
		fallback,
	)
	if err != nil {
		return fmt.Errorf("failed to get group lag: %w", err)
	}

	assignedPartitions := g.getAssignedActivePartitions()

	for _, partition := range assignedPartitions {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		partitionLag, ok := groupLag.Lookup(g.cfg.Ingest.Kafka.Topic, partition)
		if !ok {
			return fmt.Errorf("lag for partition %d not found", partition)
		}

		if partitionLag.Lag <= 0 {
			// Nothing to consume
			continue
		}

		err := g.consumePartition(ctx, partition, partitionLag)
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) consumePartition(ctx context.Context, partition int32, lag kadm.GroupMemberLag) error {
	d := ingest.NewDecoder()

	// We always rewind the partition's offset to the commit offset by reassigning the partition to the client (this triggers partition assignment).
	// This is so the cycle started exactly at the commit offset, and not at what was (potentially over-) consumed previously.
	// In the end, we remove the partition from the client (refer to the defer below) to guarantee the client always consumes
	// from one partition at a time. I.e. when this partition is consumed, we start consuming the next one.
	g.kafkaClient.AddConsumePartitions(map[string]map[int32]kgo.Offset{
		g.cfg.Ingest.Kafka.Topic: {
			partition: kgo.NewOffset().At(lag.Commit.At),
		},
	})
	defer g.kafkaClient.RemoveConsumePartitions(map[string][]int32{g.cfg.Ingest.Kafka.Topic: {partition}})

	fetches := g.kafkaClient.PollFetches(ctx)
	fetches.EachError(func(_ string, _ int32, err error) {
		if !errors.Is(err, context.Canceled) {
			level.Error(g.logger).Log("msg", "failed to fetch records", "err", err)
		}
	})

	for iter := fetches.RecordIter(); !iter.Done(); {
		r := iter.Next()

		tenant := string(r.Key)

		i, err := g.getOrCreateInstance(tenant)
		if err != nil {
			return err
		}

		d.Reset()
		req, err := d.Decode(r.Value)
		if err != nil {
			return err
		}

		for _, tr := range req.Traces {
			trace := &tempopb.Trace{}
			err = trace.Unmarshal(tr.Slice)
			if err != nil {
				return err
			}

			i.pushSpansFromQueue(ctx, &tempopb.PushSpansRequest{
				Batches: trace.ResourceSpans,
			})
		}
	}

	offsets := kadm.OffsetsFromFetches(fetches)
	err := g.kafkaAdm.CommitAllOffsets(ctx, g.cfg.Ingest.Kafka.ConsumerGroup, offsets)
	if err != nil {
		return fmt.Errorf("generator failed to commit offsets: %w", err)
	}

	return nil
}

func getGroupLag(ctx context.Context, admClient *kadm.Client, topic, group string, fallback time.Time) (kadm.GroupLag, error) {
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

	resolveFallbackOffsets := sync.OnceValues(func() (kadm.ListedOffsets, error) {
		return admClient.ListOffsetsAfterMilli(ctx, fallback.UnixMilli(), topic)
	})
	// If the group-partition in offsets doesn't have a commit, fall back depending on where fallbackOffsetMillis points at.
	for topic, pt := range startOffsets.Offsets() {
		for partition, startOffset := range pt {
			if _, ok := offsets.Lookup(topic, partition); ok {
				continue
			}
			fallbackOffsets, err := resolveFallbackOffsets()
			if err != nil {
				return nil, fmt.Errorf("resolve fallback offsets: %w", err)
			}
			o, ok := fallbackOffsets.Lookup(topic, partition)
			if !ok {
				return nil, fmt.Errorf("partition %d not found in fallback offsets for topic %s", partition, topic)
			}
			if o.Offset < startOffset.At {
				// Skip the resolved fallback offset if it's before the partition's start offset (i.e. before the earliest offset of the partition).
				// This should not happen in Kafka, but can happen in Kafka-compatible systems, e.g. Warpstream.
				continue
			}
			offsets.Add(kadm.OffsetResponse{Offset: kadm.Offset{
				Topic:       o.Topic,
				Partition:   o.Partition,
				At:          o.Offset,
				LeaderEpoch: o.LeaderEpoch,
			}})
		}
	}

	descrGroup := kadm.DescribedGroup{
		// "Empty" is the state that indicates that the group doesn't have active consumer members; this is always the case for block-builder,
		// because we don't use group consumption.
		State: "Empty",
	}
	return kadm.CalculateGroupLagWithStartOffsets(descrGroup, offsets, startOffsets, endOffsets), nil
}

func (g *Generator) getAssignedActivePartitions() []int32 {
	activePartitionsCount := g.partitionRing.PartitionRing().ActivePartitionsCount()
	assignedActivePartitions := make([]int32, 0, activePartitionsCount)
	for _, partition := range g.cfg.AssignedPartitions[g.cfg.InstanceID] {
		if partition > int32(activePartitionsCount) {
			break
		}
		assignedActivePartitions = append(assignedActivePartitions, partition)
	}
	return assignedActivePartitions
}
