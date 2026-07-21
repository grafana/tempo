package generator

import (
	"context"
	"errors"
	"slices"
	"sort"
	"strconv"
	"time"

	"github.com/go-kit/log/level"

	"github.com/grafana/tempo/pkg/ingest"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

var metricEnqueueTime = promauto.NewCounter(prometheus.CounterOpts{
	Namespace: "tempo",
	Subsystem: "metrics_generator",
	Name:      "enqueue_time_seconds_total",
	Help:      "The total amount of time spent waiting to enqueue for processing",
})

// metricAssignedPartitions tracks the number of Kafka partitions currently assigned to this
// generator instance. A value of 0 for an extended period indicates a misconfiguration
// (e.g. more generator replicas than topic partitions, or a stuck rebalance).
var metricAssignedPartitions = promauto.NewGauge(prometheus.GaugeOpts{
	Namespace: "tempo",
	Subsystem: "metrics_generator",
	Name:      "assigned_partitions",
	Help:      "Number of Kafka partitions currently assigned to this generator instance.",
})

func (g *Generator) startKafka() {
	g.kafkaCh = make(chan *kgo.Record, g.cfg.IngestConcurrency)
	// Create context that will be used to stop the goroutines.
	var ctx context.Context
	ctx, g.kafkaStop = context.WithCancel(context.Background())

	for i := uint(0); i < g.cfg.IngestConcurrency; i++ {
		g.kafkaWG.Add(1)
		go g.readCh(ctx)
	}

	g.kafkaWG.Add(1)
	go g.listenKafka(ctx)

	ingest.ExportPartitionLagMetrics(ctx, g.kafkaClient.Client, g.logger, g.cfg.Ingest, g.getAssignedActivePartitions, g.kafkaClient.ForceMetadataRefresh)
}

func (g *Generator) stopKafka() {
	g.kafkaStop()
	g.kafkaWG.Wait()
	close(g.kafkaCh)
	// When enabled, with static membership (InstanceID) franz-go does not send
	// LeaveGroup on Close(); explicitly leave by instance ID so the coordinator
	// can rebalance immediately. When disabled, avoid two rebalances (leave then
	// join) e.g. when all replicas go down then up together.
	if g.cfg.LeaveConsumerGroupOnShutdown && g.cfg.InstanceID != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := g.leaveGroupFn(ctx); err != nil {
			level.Warn(g.logger).Log(
				"msg", "failed to leave Kafka consumer group by instance ID (partitions may reassign after session timeout)",
				"err", err,
				"instance_id", g.cfg.InstanceID,
				"group", g.cfg.Ingest.Kafka.ConsumerGroup,
			)
		}
	}
	g.kafkaClient.Close()
}

func (g *Generator) listenKafka(ctx context.Context) {
	defer g.kafkaWG.Done()

	level.Info(g.logger).Log("msg", "generator now listening to kafka")
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if g.readOnly.Load() {
				// Starting up or shutting down
				continue
			}
			err := g.readKafka(ctx)
			if err != nil {
				level.Error(g.logger).Log("msg", "readKafka failed", "err", err)
				continue
			}
		}
	}
}

func (g *Generator) readKafka(ctx context.Context) error {
	fetches := g.kafkaClient.PollFetches(ctx)
	fetches.EachError(func(_ string, _ int32, err error) {
		if !errors.Is(err, context.Canceled) {
			level.Error(g.logger).Log("msg", "failed to fetch records", "err", err)
		}
	})
	if err := fetches.Err(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	// Metric lag based on first message in each partition.
	// This balances overhead with granularity.
	fetches.EachPartition(func(p kgo.FetchTopicPartition) {
		if len(p.Records) > 0 {
			lag := time.Since(p.Records[0].Timestamp)
			ingest.SetPartitionLagSeconds(g.cfg.Ingest.Kafka.ConsumerGroup, p.Partition, lag)
		}
	})

	start := time.Now()

	for iter := fetches.RecordIter(); !iter.Done(); {
		select {
		case g.kafkaCh <- iter.Next():
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	metricEnqueueTime.Add(time.Since(start).Seconds())

	return nil
}

// readCh reads records from the internal channel.
// This allows for offloading the expensive proto unmarshal
// to multiple goroutines.
func (g *Generator) readCh(ctx context.Context) {
	defer g.kafkaWG.Done()

	var c ingest.GeneratorCodec
	switch g.cfg.Codec {
	case codecPushBytes:
		c = ingest.NewPushBytesDecoder()
	case codecOTLP:
		c = ingest.NewOTLPDecoder()
	}

	for {
		var r *kgo.Record
		select {
		case r = <-g.kafkaCh:
		case <-ctx.Done():
			return
		}

		tenant := string(r.Key)

		i, err := g.getOrCreateInstance(tenant)
		if err != nil {
			level.Error(g.logger).Log("tenant", tenant, "msg", "consumeKafkaChannel getOrCreateInstance", "err", err)
			continue
		}

		iterator, err := c.Decode(r.Value)
		if err != nil {
			level.Error(g.logger).Log("tenant", tenant, "msg", "consumeKafkaChannel decode", "err", err)
			continue
		}

		for resourceSpans, err := range iterator {
			if err != nil {
				level.Error(g.logger).Log("tenant", tenant, "msg", "consumeKafkaChannel unmarshal", "err", err)
				continue
			}

			i.pushSpansFromQueue(ctx, r.Timestamp, resourceSpans)
		}
	}
}

func (g *Generator) getAssignedActivePartitions() []int32 {
	g.partitionMtx.Lock()
	defer g.partitionMtx.Unlock()
	return slices.Clone(g.assignedPartitions)
}

func (g *Generator) handlePartitionsAssigned(m map[string][]int32) {
	assigned := m[g.cfg.Ingest.Kafka.Topic]
	level.Info(g.logger).Log("msg", "partitions assigned", "partitions", formatInt32Slice(assigned))
	g.partitionMtx.Lock()
	defer g.partitionMtx.Unlock()

	// In cooperative (incremental) rebalancing this callback fires with only
	// newly added partitions; stable partitions are not re-reported. Append
	// rather than replace so we don't lose partitions that weren't moved.
	g.assignedPartitions = append(g.assignedPartitions, assigned...)
	sort.Slice(g.assignedPartitions, func(i, j int) bool { return g.assignedPartitions[i] < g.assignedPartitions[j] })
	metricAssignedPartitions.Set(float64(len(g.assignedPartitions)))
}

func (g *Generator) handlePartitionsRevoked(partitions map[string][]int32) {
	revoked := partitions[g.cfg.Ingest.Kafka.Topic]
	level.Info(g.logger).Log("msg", "partitions revoked", "partitions", formatInt32Slice(revoked))
	g.partitionMtx.Lock()
	defer g.partitionMtx.Unlock()

	sort.Slice(revoked, func(i, j int) bool { return revoked[i] < revoked[j] })
	// Remove revoked partitions
	g.assignedPartitions = revokePartitions(g.assignedPartitions, revoked)
	metricAssignedPartitions.Set(float64(len(g.assignedPartitions)))

	ingest.ResetLagMetricsForRevokedPartitions(g.cfg.Ingest.Kafka.ConsumerGroup, revoked)
}

// Helper function to format []int32 slice
func formatInt32Slice(slice []int32) string {
	if len(slice) == 0 {
		return "[]"
	}
	result := "["
	for i, v := range slice {
		if i > 0 {
			result += ","
		}
		result += strconv.Itoa(int(v))
	}
	result += "]"
	return result
}

// Helper function to revoke partitions
// Assumes both slices are sorted
func revokePartitions(assigned, revoked []int32) []int32 {
	i, j := 0, 0
	// k is used to track the position where we will overwrite elements in assigned
	k := 0

	// Traverse both slices
	for i < len(assigned) && j < len(revoked) {
		if assigned[i] < revoked[j] {
			// If element in assigned is smaller, it's not in revoked, retain it
			assigned[k] = assigned[i]
			k++
			i++
		} else if assigned[i] > revoked[j] {
			// If element in revoked is smaller, move the pointer j
			j++
		} else {
			// If both elements are equal, skip the element from assigned
			i++
		}
	}

	// If there are leftover elements in assigned, retain them
	for i < len(assigned) {
		assigned[k] = assigned[i]
		k++
		i++
	}

	// Resize assigned to only include retained elements
	return assigned[:k]
}

// kafkaOffsetNone marks the absence of a committed group offset for a partition.
const kafkaOffsetNone = int64(-1)

// startupSeekOffset decides which offset to resume a partition from at startup
// when skip_stale_backlog_on_startup is enabled. It never rewinds behind the
// committed offset, but skips forward to horizonOffset (the offset of the first
// record at/after now-horizon) when the committed offset is behind that horizon
// or absent. The returned bool reports whether an explicit seek is needed;
// false means resume from the committed offset as usual.
func startupSeekOffset(committed, horizonOffset int64) (int64, bool) {
	if committed != kafkaOffsetNone && committed >= horizonOffset {
		return committed, false
	}
	return horizonOffset, true
}

// startupSeekOffsets builds the set of offsets to seek to for the given
// partitions, applying startupSeekOffset per partition. A partition is included
// only when it has a known horizon offset and its committed offset is behind
// that horizon (or absent); partitions already at/ahead of the horizon are
// omitted so consumption resumes from the committed offset as usual.
func startupSeekOffsets(partitions []int32, committed, horizon map[int32]int64) map[int32]int64 {
	out := make(map[int32]int64)
	for _, p := range partitions {
		h, ok := horizon[p]
		if !ok {
			continue
		}
		c, ok := committed[p]
		if !ok {
			c = kafkaOffsetNone
		}
		if target, seek := startupSeekOffset(c, h); seek {
			out[p] = target
		}
	}
	return out
}

// adjustStartupOffsets is the franz-go AdjustFetchOffsetsFn hook. When
// skip_stale_backlog_on_startup is enabled it rewrites the fetch offsets for the
// joining partitions to skip forward past backlog older than
// metrics_ingestion_time_range_slack, so the generator does not replay spans the
// slack would discard and its partition-lag metric stays honest on restart.
// Partitions already at/ahead of the slack horizon keep their committed offset.
func (g *Generator) adjustStartupOffsets(ctx context.Context, offsets map[string]map[int32]kgo.Offset) (map[string]map[int32]kgo.Offset, error) {
	topic := g.cfg.Ingest.Kafka.Topic
	parts := offsets[topic]
	if len(parts) == 0 {
		return offsets, nil
	}

	partitionIDs := make([]int32, 0, len(parts))
	committed := make(map[int32]int64, len(parts))
	for p, off := range parts {
		partitionIDs = append(partitionIDs, p)
		committed[p] = off.EpochOffset().Offset
	}

	horizonMs := time.Now().Add(-g.cfg.MetricsIngestionSlack).UnixMilli()
	horizonOffsets, err := g.partitionClient.FetchPartitionsOffsetsAfterMilli(ctx, horizonMs, partitionIDs)
	if err != nil {
		// Best-effort optimization: on lookup failure, fall back to replaying
		// from the committed offset rather than blocking startup.
		level.Warn(g.logger).Log("msg", "skip stale backlog on startup: horizon offset lookup failed, replaying from committed offset", "err", err)
		return offsets, nil
	}

	// Resolve the seek target per partition. A horizon offset of -1 means no
	// record is at/after the horizon (all backlog is stale), so seek to the end.
	var endOffsets kadm.ListedOffsets
	horizon := make(map[int32]int64, len(partitionIDs))
	for _, p := range partitionIDs {
		o, ok := horizonOffsets[topic][p]
		if !ok {
			continue
		}
		if o.Offset >= 0 {
			horizon[p] = o.Offset
			continue
		}
		if endOffsets == nil {
			if endOffsets, err = g.partitionClient.FetchPartitionsLastProducedOffsets(ctx, partitionIDs); err != nil {
				level.Warn(g.logger).Log("msg", "skip stale backlog on startup: end offset lookup failed, replaying from committed offset", "err", err)
				return offsets, nil
			}
		}
		if eo, ok := endOffsets[topic][p]; ok {
			horizon[p] = eo.Offset
		}
	}

	for p, target := range startupSeekOffsets(partitionIDs, committed, horizon) {
		parts[p] = kgo.NewOffset().At(target).WithEpoch(-1)
		level.Info(g.logger).Log("msg", "skipping stale backlog on startup",
			"partition", p, "committed", committed[p], "seek_to", target)
	}
	return offsets, nil
}
