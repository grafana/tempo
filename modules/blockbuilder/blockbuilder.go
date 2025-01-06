package blockbuilder

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/backoff"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	blockBuilderServiceName = "block-builder"
	ConsumerGroup           = "block-builder"
	pollTimeout             = 2 * time.Second
)

var (
	metricPartitionLag = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Subsystem: "block_builder",
		Name:      "partition_lag",
		Help:      "Lag of a partition.",
	}, []string{"partition"})
	metricPartitionLagSeconds = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Subsystem: "block_builder",
		Name:      "partition_lag_s",
		Help:      "Lag of a partition in seconds.",
	}, []string{"partition"})
	metricConsumeCycleDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace:                   "tempo",
		Subsystem:                   "block_builder",
		Name:                        "consume_cycle_duration_seconds",
		Help:                        "Time spent consuming a full cycle.",
		NativeHistogramBucketFactor: 1.1,
	})
	metricProcessPartitionSectionDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace:                   "tempo",
		Subsystem:                   "block_builder",
		Name:                        "process_partition_section_duration_seconds",
		Help:                        "Time spent processing one partition section.",
		NativeHistogramBucketFactor: 1.1,
	}, []string{"partition"})
	metricFetchErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Subsystem: "block_builder",
		Name:      "fetch_errors_total",
		Help:      "Total number of errors while fetching by the consumer.",
	}, []string{"partition"})
)

type BlockBuilder struct {
	services.Service

	logger               log.Logger
	cfg                  Config
	assignedPartitions   []int32 // TODO - Necessary?
	fallbackOffsetMillis int64

	kafkaClient   *kgo.Client
	kadm          *kadm.Client
	decoder       *ingest.Decoder
	partitionRing ring.PartitionRingReader

	overrides Overrides
	enc       encoding.VersionedEncoding
	wal       *wal.WAL // TODO - Shared between tenants, should be per tenant?
	writer    tempodb.Writer
}

func New(
	cfg Config,
	logger log.Logger,
	partitionRing ring.PartitionRingReader,
	overrides Overrides,
	store storage.Store,
) *BlockBuilder {
	b := &BlockBuilder{
		logger:        logger,
		cfg:           cfg,
		partitionRing: partitionRing,
		decoder:       ingest.NewDecoder(),
		overrides:     overrides,
		writer:        store,
	}

	b.Service = services.NewBasicService(b.starting, b.running, b.stopping)
	return b
}

func (b *BlockBuilder) starting(ctx context.Context) (err error) {
	level.Info(b.logger).Log("msg", "block builder starting")

	b.enc = encoding.DefaultEncoding()
	if version := b.cfg.BlockConfig.BlockCfg.Version; version != "" {
		b.enc, err = encoding.FromVersion(version)
		if err != nil {
			return fmt.Errorf("failed to create encoding: %w", err)
		}
	}

	b.wal, err = wal.New(&b.cfg.WAL)
	if err != nil {
		return fmt.Errorf("failed to create WAL: %w", err)
	}

	// Fallback offset is a millisecond timestamp used to look up a real offset if partition doesn't have a commit.
	b.fallbackOffsetMillis = time.Now().Add(-b.cfg.LookbackOnNoCommit).UnixMilli()

	b.kafkaClient, err = ingest.NewReaderClient(
		b.cfg.IngestStorageConfig.Kafka,
		ingest.NewReaderClientMetrics(blockBuilderServiceName, prometheus.DefaultRegisterer),
		b.logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create kafka reader client: %w", err)
	}

	boff := backoff.New(ctx, backoff.Config{
		MinBackoff: 100 * time.Millisecond,
		MaxBackoff: time.Minute, // If there is a network hiccup, we prefer to wait longer retrying, than fail the service.
		MaxRetries: 10,
	})

	for boff.Ongoing() {
		err := b.kafkaClient.Ping(ctx)
		if err == nil {
			break
		}
		level.Warn(b.logger).Log("msg", "ping kafka; will retry", "err", err)
		boff.Wait()
	}
	if err := boff.ErrCause(); err != nil {
		return fmt.Errorf("failed to ping kafka: %w", err)
	}

	b.kadm = kadm.NewClient(b.kafkaClient)

	go b.metricLag(ctx)

	return nil
}

func (b *BlockBuilder) runningOld(ctx context.Context) error {
	// Initial polling and delay
	cycleEndTime := cycleEndAtStartup(time.Now(), b.cfg.ConsumeCycleDuration)
	waitTime := 2 * time.Second
	for {
		select {
		case <-time.After(waitTime):
			err := b.consumeCycle(ctx, cycleEndTime)
			if err != nil {
				b.logger.Log("msg", "consumeCycle failed", "err", err)

				// Don't progress cycle forward, keep trying at this timestamp
				continue
			}

			cycleEndTime, waitTime = nextCycleEnd(cycleEndTime, b.cfg.ConsumeCycleDuration)
		case <-ctx.Done():
			return nil
		}
	}
}

func (b *BlockBuilder) running(ctx context.Context) error {
	// Initial delay
	waitTime := 0 * time.Second
	for {
		select {
		case <-time.After(waitTime):
			err := b.consume(ctx)
			if err != nil {
				level.Error(b.logger).Log("msg", "consumeCycle failed", "err", err)
			}

			// Real delay on subsequent
			waitTime = b.cfg.ConsumeCycleDuration
		case <-ctx.Done():
			return nil
		}
	}
}

func (b *BlockBuilder) consume(ctx context.Context) error {
	var (
		end        = time.Now()
		partitions = b.getAssignedActivePartitions()
	)

	level.Info(b.logger).Log("msg", "starting consume cycle", "cycle_end", end, "active_partitions", partitions)
	defer func(t time.Time) { metricConsumeCycleDuration.Observe(time.Since(t).Seconds()) }(time.Now())

	for _, partition := range partitions {
		// Consume partition while data remains.
		// TODO - round-robin one consumption per partition instead to equalize catch-up time.
		for {
			more, err := b.consumePartition2(ctx, partition, end)
			if err != nil {
				return err
			}

			if !more {
				break
			}
		}
	}

	return nil
}

func (b *BlockBuilder) consumePartition2(ctx context.Context, partition int32, overallEnd time.Time) (more bool, err error) {
	defer func(t time.Time) {
		metricProcessPartitionSectionDuration.WithLabelValues(strconv.Itoa(int(partition))).Observe(time.Since(t).Seconds())
	}(time.Now())

	var (
		dur         = b.cfg.ConsumeCycleDuration
		topic       = b.cfg.IngestStorageConfig.Kafka.Topic
		group       = b.cfg.IngestStorageConfig.Kafka.ConsumerGroup
		startOffset kgo.Offset
		writer      *writer
		lastRec     *kgo.Record
		begin       time.Time
		end         time.Time
	)

	commits, err := b.kadm.FetchOffsetsForTopics(ctx, group, topic)
	if err != nil {
		return false, err
	}

	lastCommit, ok := commits.Lookup(topic, partition)
	if ok && lastCommit.At >= 0 {
		startOffset = startOffset.At(lastCommit.At)
	} else {
		startOffset = kgo.NewOffset().AtStart()
	}

	// We always rewind the partition's offset to the commit offset by reassigning the partition to the client (this triggers partition assignment).
	// This is so the cycle started exactly at the commit offset, and not at what was (potentially over-) consumed previously.
	// In the end, we remove the partition from the client (refer to the defer below) to guarantee the client always consumes
	// from one partition at a time. I.e. when this partition is consumed, we start consuming the next one.
	b.kafkaClient.AddConsumePartitions(map[string]map[int32]kgo.Offset{
		topic: {
			partition: startOffset,
		},
	})
	defer b.kafkaClient.RemoveConsumePartitions(map[string][]int32{topic: {partition}})

outer:
	for {
		fetches := func() kgo.Fetches {
			ctx2, cancel := context.WithTimeout(ctx, pollTimeout)
			defer cancel()
			return b.kafkaClient.PollFetches(ctx2)
		}()
		err = fetches.Err()
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				// No more data
				break
			}
			metricFetchErrors.WithLabelValues(strconv.Itoa(int(partition))).Inc()
			return false, err
		}

		if fetches.Empty() {
			break
		}

		for iter := fetches.RecordIter(); !iter.Done(); {
			rec := iter.Next()

			// Initialize if needed
			if writer == nil {

				// Determine block begin and end time range, which is -/+ cycle duration.
				begin = rec.Timestamp.Add(-dur)
				end = rec.Timestamp.Add(dur)

				metricPartitionLagSeconds.WithLabelValues(strconv.Itoa(int(partition))).Set(time.Since(rec.Timestamp).Seconds())

				writer = newPartitionSectionWriter(b.logger, uint64(partition), uint64(rec.Offset), b.cfg.BlockConfig, b.overrides, b.wal, b.enc)
			}

			if rec.Timestamp.Before(begin) || rec.Timestamp.After(end) {
				// Cut this block but continue only if we have at least another full cycle
				if overallEnd.Sub(rec.Timestamp) >= dur {
					more = true
				}
				break outer
			}

			if rec.Timestamp.After(overallEnd) {
				break outer
			}

			err := b.pushTraces(rec.Key, rec.Value, writer)
			if err != nil {
				return false, err
			}

			lastRec = rec
		}
	}

	if lastRec == nil {
		// Received no data
		return false, nil
	}

	err = writer.flush(ctx, b.writer)
	if err != nil {
		return false, err
	}

	// TOOD - Retry commit
	resp, err := b.kadm.CommitOffsets(ctx, group, kadm.OffsetsFromRecords(*lastRec))
	if err != nil {
		return false, err
	}
	if err := resp.Error(); err != nil {
		return false, err
	}

	return more, nil
}

func (b *BlockBuilder) metricLag(ctx context.Context) {
	var (
		waitTime = time.Second * 15
		topic    = b.cfg.IngestStorageConfig.Kafka.Topic
		group    = b.cfg.IngestStorageConfig.Kafka.ConsumerGroup
	)

	for {
		select {
		case <-time.After(waitTime):
			metricPartitionLag.Reset()

			lag, err := getGroupLag(ctx, b.kadm, topic, group)
			if err != nil {
				level.Error(b.logger).Log("msg", "metric lag failed:", "err", err)
				continue
			}
			for _, p := range b.getAssignedActivePartitions() {
				l, ok := lag.Lookup(topic, p)
				if ok {
					metricPartitionLag.WithLabelValues(strconv.Itoa(int(p))).Set(float64(l.Lag))
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (b *BlockBuilder) stopping(err error) error {
	if b.kafkaClient != nil {
		b.kafkaClient.Close()
	}
	return err
}

func (b *BlockBuilder) consumeCycle(ctx context.Context, cycleEndTime time.Time) error {
	level.Info(b.logger).Log("msg", "starting consume cycle", "cycle_end", cycleEndTime)
	defer func(t time.Time) { metricConsumeCycleDuration.Observe(time.Since(t).Seconds()) }(time.Now())

	groupLag, err := getGroupLag(
		ctx,
		kadm.NewClient(b.kafkaClient),
		b.cfg.IngestStorageConfig.Kafka.Topic,
		b.cfg.IngestStorageConfig.Kafka.ConsumerGroup,
	)
	if err != nil {
		return fmt.Errorf("failed to get group lag: %w", err)
	}

	assignedPartitions := b.getAssignedActivePartitions()

	for _, partition := range assignedPartitions {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		partitionLag, ok := groupLag.Lookup(b.cfg.IngestStorageConfig.Kafka.Topic, partition)
		if !ok {
			return fmt.Errorf("lag for partition %d not found", partition)
		}

		level.Debug(b.logger).Log(
			"msg", "partition lag",
			"partition", partition,
			"lag", fmt.Sprintf("%+v", partitionLag),
		)

		metricPartitionLag.WithLabelValues(fmt.Sprintf("%d", partition)).Set(float64(partitionLag.Lag))

		if partitionLag.Lag <= 0 {
			level.Info(b.logger).Log(
				"msg", "nothing to consume in partition",
				"partition", partition,
				"commit_offset", partitionLag.Commit.At,
				"start_offset", partitionLag.Start.Offset,
				"end_offset", partitionLag.End.Offset,
				"lag", partitionLag.Lag,
			)
			continue
		}

		if err = b.consumePartition(ctx, partition, partitionLag, cycleEndTime); err != nil {
			_ = level.Error(b.logger).Log("msg", "failed to consume partition", "partition", partition, "err", err)
		}
	}
	return nil
}

func (b *BlockBuilder) consumePartition(ctx context.Context, partition int32, partitionLag kadm.GroupMemberLag, cycleEndTime time.Time) error {
	level.Info(b.logger).Log(
		"msg", "consuming partition",
		"partition", partition,
		"cycle_end", cycleEndTime,
	)

	sectionEndTime := cycleEndTime

	lastCommitTs, err := unmarshallCommitMeta(partitionLag.Commit.Metadata)
	if err != nil {
		return fmt.Errorf("failed to unmarshal commit metadata: %w", err)
	}
	if lastCommitTs == 0 {
		lastCommitTs = b.fallbackOffsetMillis // No commit yet, use fallback offset.
	}
	commitRecTs := time.UnixMilli(lastCommitTs)

	// We need to align the commit record timestamp to the section end time so we don't consume the same section again.
	commitSectionEndTime := alignToSectionEndTime(commitRecTs, b.cfg.ConsumeCycleDuration)
	if sectionEndTime.Sub(commitSectionEndTime) > time.Duration(1.5*float64(b.cfg.ConsumeCycleDuration)) {
		// We're lagging behind or there is no commit, we need to consume in smaller sections.
		// We iterate through all the ConsumeInterval intervals, starting from the first one after the last commit until the cycleEndTime,
		// i.e. [T, T+interval), [T+interval, T+2*interval), ... [T+S*interval, cycleEndTime)
		// where T is the CommitRecordTimestamp, the timestamp of the record, whose offset we committed previously.
		sectionEndTime, _ = nextCycleEnd(commitSectionEndTime, b.cfg.ConsumeCycleDuration)

		level.Debug(b.logger).Log(
			"msg", "lagging behind, consuming in sections",
			"partition", partition,
			"section_end", sectionEndTime,
			"commit_rec_ts", commitRecTs,
			"commit_section_end", commitSectionEndTime,
			"cycle_end", cycleEndTime,
		)
	}

	// Continue consuming in sections until we're caught up.
	for !sectionEndTime.After(cycleEndTime) {
		newCommitAt, err := b.consumePartitionSection(ctx, partition, sectionEndTime, partitionLag)
		if err != nil {
			return fmt.Errorf("failed to consume partition section: %w", err)
		}
		sectionEndTime = sectionEndTime.Add(b.cfg.ConsumeCycleDuration)
		if newCommitAt > partitionLag.Commit.At {
			// We've committed a new offset, so we need to update the lag.
			partitionLag.Commit.At = newCommitAt
			partitionLag.Lag = partitionLag.End.Offset - newCommitAt
		}
	}
	return nil
}

func (b *BlockBuilder) consumePartitionSection(ctx context.Context, partition int32, sectionEndTime time.Time, lag kadm.GroupMemberLag) (int64, error) {
	level.Info(b.logger).Log(
		"msg", "consuming partition section",
		"partition", partition,
		"section_end", sectionEndTime,
		"commit_offset", lag.Commit.At,
		"start_offset", lag.Start.Offset,
		"end_offset", lag.End.Offset,
		"lag", lag.Lag,
	)

	defer func(t time.Time) {
		metricProcessPartitionSectionDuration.WithLabelValues(fmt.Sprintf("%d", partition)).Observe(time.Since(t).Seconds())
	}(time.Now())

	// TODO - Review what endTimestamp is used here
	writer := newPartitionSectionWriter(b.logger, uint64(partition), uint64(sectionEndTime.UnixMilli()), b.cfg.BlockConfig, b.overrides, b.wal, b.enc)

	// We always rewind the partition's offset to the commit offset by reassigning the partition to the client (this triggers partition assignment).
	// This is so the cycle started exactly at the commit offset, and not at what was (potentially over-) consumed previously.
	// In the end, we remove the partition from the client (refer to the defer below) to guarantee the client always consumes
	// from one partition at a time. I.e. when this partition is consumed, we start consuming the next one.
	b.kafkaClient.AddConsumePartitions(map[string]map[int32]kgo.Offset{
		b.cfg.IngestStorageConfig.Kafka.Topic: {
			partition: kgo.NewOffset().At(lag.Commit.At),
		},
	})
	defer b.kafkaClient.RemoveConsumePartitions(map[string][]int32{b.cfg.IngestStorageConfig.Kafka.Topic: {partition}})

	var (
		firstRec *kgo.Record
		lastRec  *kgo.Record
	)

consumerLoop:
	for recOffset := int64(-1); recOffset < lag.End.Offset-1; {
		if err := context.Cause(ctx); err != nil {
			return lag.Commit.At, err
		}

		// PollFetches can return a non-failed fetch with zero records. In such a case, with only the fetches at hands,
		// we cannot tell if the consumer has already reached the latest end of the partition, i.e. no more records to consume,
		// or there is more data in the backlog, and we must retry the poll. That's why the consumer loop above has to guard
		// the iterations against the cycleEndOffset, so it retried the polling up until the expected end of the partition is reached.
		fetches := b.kafkaClient.PollFetches(ctx)
		fetches.EachError(func(_ string, _ int32, err error) {
			if !errors.Is(err, context.Canceled) {
				level.Error(b.logger).Log("msg", "failed to fetch records", "err", err)
				metricFetchErrors.WithLabelValues(fmt.Sprintf("%d", partition)).Inc()
			}
		})

		for recIter := fetches.RecordIter(); !recIter.Done(); {
			rec := recIter.Next()
			recOffset = rec.Offset
			level.Debug(b.logger).Log(
				"msg", "processing record",
				"partition", rec.Partition,
				"offset", rec.Offset,
				"timestamp", rec.Timestamp,
			)

			if firstRec == nil {
				firstRec = rec
			}

			// Stop consuming after we reached the sectionEndTime marker.
			// NOTE: the timestamp of the record is when the record was produced relative to distributor's time.
			if rec.Timestamp.After(sectionEndTime) {
				break consumerLoop
			}

			err := b.pushTraces(rec.Key, rec.Value, writer) // TODO - Batch pushes by tenant
			if err != nil {
				// All "non-terminal" errors are handled by the TSDBBuilder.
				return lag.Commit.At, fmt.Errorf("process record in partition %d at offset %d: %w", rec.Partition, rec.Offset, err)
			}
			lastRec = rec
		}
	}

	// Nothing was consumed from Kafka at all.
	if firstRec == nil {
		level.Info(b.logger).Log("msg", "no records were consumed")
		return lag.Commit.At, nil
	}

	// No records were processed for this cycle.
	if lastRec == nil {
		level.Info(b.logger).Log("msg", "nothing to commit due to first record has a timestamp greater than this section end", "first_rec_offset", firstRec.Offset, "first_rec_ts", firstRec.Timestamp)
		return lag.Commit.At, nil
	}

	if err := writer.flush(ctx, b.writer); err != nil {
		return lag.Commit.At, fmt.Errorf("failed to flush partition to object storage: %w", err)
	}

	commit := kadm.Offset{
		Topic:       lastRec.Topic,
		Partition:   lastRec.Partition,
		At:          lastRec.Offset + 1, // offset+1 means everything up to (including) the offset was processed
		LeaderEpoch: lastRec.LeaderEpoch,
		Metadata:    marshallCommitMeta(lastRec.Timestamp.UnixMilli()),
	}
	return commit.At, b.commitState(ctx, commit)
}

func (b *BlockBuilder) commitState(ctx context.Context, commit kadm.Offset) error {
	offsets := make(kadm.Offsets)
	offsets.Add(commit)

	// TODO - Commit with backoff
	adm := kadm.NewClient(b.kafkaClient)
	err := adm.CommitAllOffsets(ctx, b.cfg.IngestStorageConfig.Kafka.ConsumerGroup, offsets)
	if err != nil {
		return fmt.Errorf("failed to commit offsets: %w", err)
	}
	level.Info(b.logger).Log("msg", "successfully committed offset to kafka", "offset", commit.At)

	return nil
}

func (b *BlockBuilder) pushTraces(tenantBytes, reqBytes []byte, p partitionSectionWriter) error {
	req, err := b.decoder.Decode(reqBytes)
	if err != nil {
		return fmt.Errorf("failed to decode trace: %w", err)
	}
	defer b.decoder.Reset()

	return p.pushBytes(string(tenantBytes), req)
}

func (b *BlockBuilder) getAssignedActivePartitions() []int32 {
	activePartitionsCount := b.partitionRing.PartitionRing().ActivePartitionsCount()
	assignedActivePartitions := make([]int32, 0, activePartitionsCount)
	for _, partition := range b.cfg.AssignedPartitions[b.cfg.InstanceID] {
		if partition > int32(activePartitionsCount) {
			break
		}
		assignedActivePartitions = append(assignedActivePartitions, partition)
	}
	return assignedActivePartitions
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

func (b *BlockBuilder) onRevoked(_ context.Context, _ *kgo.Client, revoked map[string][]int32) {
	for topic, partitions := range revoked {
		partitionsStr := fmt.Sprintf("%v", partitions)
		level.Info(b.logger).Log("msg", "partitions revoked", "topic", topic, "partitions", partitionsStr)
	}
	b.assignedPartitions = revoked[b.cfg.IngestStorageConfig.Kafka.Topic]
}

func (b *BlockBuilder) onAssigned(_ context.Context, _ *kgo.Client, assigned map[string][]int32) {
	// TODO - All partitions are assigned, not just the ones in use by the partition ring (ingesters).
	for topic, partitions := range assigned {
		var partitionsStr string
		for _, partition := range partitions {
			partitionsStr += fmt.Sprintf("%d, ", partition)
		}
		level.Info(b.logger).Log("msg", "partitions assigned", "topic", topic, "partitions", partitionsStr)
	}
	b.assignedPartitions = assigned[b.cfg.IngestStorageConfig.Kafka.Topic]
}

// cycleEndAtStartup is the timestamp of the cycle end at startup.
// It's the nearest interval boundary in the past.
func cycleEndAtStartup(t time.Time, interval time.Duration) time.Time {
	cycleEnd := t.Truncate(interval)
	if cycleEnd.After(t) {
		cycleEnd = cycleEnd.Add(-interval)
	}
	return cycleEnd
}

// nextCycleEnd returns the timestamp of the next cycleEnd relative to the time t.
// One cycle is a duration of one interval.
func nextCycleEnd(t time.Time, interval time.Duration) (time.Time, time.Duration) {
	cycleEnd := t.Truncate(interval).Add(interval)
	waitTime := cycleEnd.Sub(t)
	for waitTime > interval {
		// Example - with interval=1h and buffer=15m:
		// - at t=14:12, next cycle starts at 14:15 (startup cycle ended at 13:15)
		// - at t=14:17, next cycle starts at 15:15 (startup cycle ended at 14:15)
		cycleEnd = cycleEnd.Add(-interval)
		waitTime -= interval
	}
	return cycleEnd, waitTime
}

func alignToSectionEndTime(t time.Time, interval time.Duration) time.Time {
	return t.Truncate(interval).Add(interval)
}
