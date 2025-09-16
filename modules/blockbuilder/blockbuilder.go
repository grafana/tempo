package blockbuilder

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
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
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	blockBuilderServiceName = "block-builder"
	ConsumerGroup           = "block-builder"
	pollTimeout             = 2 * time.Second
	cutTime                 = 10 * time.Second
	emptyPartitionEndOffset = 0  // partition has no records
	commitOffsetAtEnd       = -1 // offset is at the end of partition
	commitOffsetAtStart     = -2 // offset is at the start of partition
)

var (
	metricFetchDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace:                   "tempo",
		Subsystem:                   "block_builder",
		Name:                        "fetch_duration_seconds",
		Help:                        "Time spent fetching from Kafka.",
		NativeHistogramBucketFactor: 1.1,
	}, []string{"partition"})
	metricFetchBytesTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Subsystem: "block_builder",
		Name:      "fetch_bytes_total",
		Help:      "Total number of bytes fetched from Kafka",
	}, []string{"partition"})
	metricFetchRecordsTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Subsystem: "block_builder",
		Name:      "fetch_records_total",
		Help:      "Total number of records fetched from Kafka",
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
	metricOwnedPartitions = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Subsystem: "block_builder",
		Name:      "owned_partitions",
		Help:      "Indicates partition ownership by this block-builder (1 = owned).",
	}, []string{"partition", "state"})

	tracer = otel.Tracer("modules/blockbuilder")

	errNoPartitionsAssigned = errors.New("no partitions assigned")
)

type BlockBuilder struct {
	services.Service

	logger log.Logger
	cfg    Config

	kafkaClient           *kgo.Client
	partitionOffsetClient *ingest.PartitionOffsetClient
	kadm                  *kadm.Client
	decoder               *ingest.Decoder
	partitionRing         ring.PartitionRingReader

	overrides Overrides
	enc       encoding.VersionedEncoding
	wal       *wal.WAL // TODO - Shared between tenants, should be per tenant?

	reader    tempodb.Reader
	writer    tempodb.Writer
	compactor tempodb.Compactor

	consumeStopped chan struct{}
}

type partitionState struct {
	// Partition number
	partition int32
	// commitOffset is the last committed consumer offset for this partition
	// it is maintained per consumer group
	commitOffset int64
	// endOffset is the latest offset for this partition
	// it represents the last message written by producers
	endOffset int64
	// Last committed record timestamp
	lastRecordTs time.Time
}

func (p partitionState) getStartOffset() kgo.Offset {
	if p.commitOffset > commitOffsetAtEnd {
		return kgo.NewOffset().At(p.commitOffset)
	}
	// If commit offset is AtEnd (-1), it nevertheless will consume from the start.
	// This is a workaround for franz-go and default Kafka behaviour:
	// in case consumer is new and has no committed offsets, it will start consuming from the end,
	// while for block builder, it should consume from the earliest record.
	// The workaround is dirty and can break the consumer if it starts returning AtEnd (-1) for
	// already running consumer.
	// TODO: replace the workaround with proper new consumer offset initialization
	// if p.commitOffset == commitOffsetAtEnd {
	// 	return kgo.NewOffset().AtEnd()
	// }
	return kgo.NewOffset().AtStart()
}

func (p partitionState) hasRecords() bool {
	return p.endOffset > emptyPartitionEndOffset
}

func New(
	cfg Config,
	logger log.Logger,
	partitionRing ring.PartitionRingReader,
	overrides Overrides,
	store storage.Store,
) (*BlockBuilder, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	b := &BlockBuilder{
		logger:         logger,
		cfg:            cfg,
		partitionRing:  partitionRing,
		decoder:        ingest.NewDecoder(),
		overrides:      overrides,
		reader:         store,
		writer:         store,
		compactor:      store,
		consumeStopped: make(chan struct{}),
	}

	b.Service = services.NewBasicService(b.starting, b.running, b.stopping)
	return b, nil
}

func (b *BlockBuilder) starting(ctx context.Context) (err error) {
	level.Info(b.logger).Log("msg", "block builder starting")
	topic := b.cfg.IngestStorageConfig.Kafka.Topic
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

	b.kafkaClient, err = ingest.NewReaderClient(
		b.cfg.IngestStorageConfig.Kafka,
		ingest.NewReaderClientMetrics(blockBuilderServiceName, prometheus.DefaultRegisterer),
		b.logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create kafka reader client: %w", err)
	}

	err = ingest.WaitForKafkaBroker(ctx, b.kafkaClient, b.logger)
	if err != nil {
		return fmt.Errorf("failed to start blockbuilder: %w", err)
	}

	b.partitionOffsetClient = ingest.NewPartitionOffsetClient(b.kafkaClient, topic)
	b.kadm = kadm.NewClient(b.kafkaClient)

	ingest.ExportPartitionLagMetrics(
		ctx,
		b.kafkaClient,
		b.logger,
		b.cfg.IngestStorageConfig,
		b.getAssignedPartitions,
		b.kafkaClient.ForceMetadataRefresh)

	return nil
}

func (b *BlockBuilder) running(ctx context.Context) error {
	defer close(b.consumeStopped)
	for {
		// Create a detached context for consume
		consumeCtx, cancel := context.WithCancel(context.Background())

		waitTime, err := b.consume(consumeCtx)
		cancel() // Always cancel the context after consume completes

		if err != nil {
			level.Error(b.logger).Log("msg", "consumeCycle failed", "err", err)
		}

		select {
		case <-time.After(waitTime): // Continue with next cycle
		case <-ctx.Done():
			// Parent context canceled, return
			return nil
		}
	}
}

// It consumes records for all the asigneed partitions, priorizing the ones with more lag. It keeps consuming until
// all the partitions lag is less than the cycle duration. When that happen it returns time to wait before another consuming cycle, based on the last record timestamp
func (b *BlockBuilder) consume(ctx context.Context) (time.Duration, error) {
	partitions := b.getAssignedPartitions()

	ctx, span := tracer.Start(ctx, "blockbuilder.consume", trace.WithAttributes(attribute.String("active_partitions", formatActivePartitions(partitions))))
	defer span.End()

	if len(partitions) == 0 {
		return b.cfg.ConsumeCycleDuration, errNoPartitionsAssigned
	}

	level.Info(b.logger).Log("msg", "starting consume cycle", "active_partitions", formatActivePartitions(partitions))
	defer func(t time.Time) { metricConsumeCycleDuration.Observe(time.Since(t).Seconds()) }(time.Now())

	// Clear all previous remnants
	err := b.wal.Clear()
	if err != nil {
		return 0, err
	}

	ps, err := b.fetchPartitions(ctx, partitions)
	if err != nil {
		return 0, err
	}

	// First iteration over all the assigned partitions to get their current lag in time
	for i, p := range ps {
		if !p.hasRecords() { // No records, skip for the first iteration
			// We treat the partition as updated through now,
			// and will check it again after ConsumeCycleDuration has elapsed
			ps[i].lastRecordTs = time.Now()
			ps[i].commitOffset = 0 // always start at beginning
			level.Info(b.logger).Log("msg", "partition has no records", "partition", p.partition)
			continue
		}
		lastRecordTs, commitOffset, err := b.consumePartition(ctx, p)
		if err != nil {
			return 0, err
		}
		ps[i].lastRecordTs = lastRecordTs
		ps[i].commitOffset = commitOffset
	}

	// Iterate over the laggiest partition until the lag is less than the cycle duration or none of the partitions has records
	for {
		sort.Slice(ps, func(i, j int) bool {
			return ps[i].lastRecordTs.Before(ps[j].lastRecordTs)
		})

		laggiestPartition := ps[0]
		if laggiestPartition.lastRecordTs.IsZero() {
			return b.cfg.ConsumeCycleDuration, nil
		}

		lagTime := time.Since(laggiestPartition.lastRecordTs)
		if lagTime < b.cfg.ConsumeCycleDuration {
			return b.cfg.ConsumeCycleDuration - lagTime, nil
		}
		lastRecordTs, lastRecordOffset, err := b.consumePartition(ctx, laggiestPartition)
		if err != nil {
			return 0, err
		}
		ps[0].lastRecordTs = lastRecordTs
		ps[0].commitOffset = lastRecordOffset
	}
}

func (b *BlockBuilder) consumePartition(ctx context.Context, ps partitionState) (lastTs time.Time, commitOffset int64, err error) {
	ctx, span := tracer.Start(ctx, "blockbuilder.consumePartition",
		trace.WithAttributes(attribute.Int("partition", int(ps.partition)),
			attribute.String("last_record_ts", ps.lastRecordTs.String())))

	defer func(t time.Time) {
		metricProcessPartitionSectionDuration.WithLabelValues(strconv.Itoa(int(ps.partition))).Observe(time.Since(t).Seconds())
		span.End()
	}(time.Now())

	var (
		dur              = b.cfg.ConsumeCycleDuration
		topic            = b.cfg.IngestStorageConfig.Kafka.Topic
		group            = b.cfg.IngestStorageConfig.Kafka.ConsumerGroup
		maxBytesPerCycle = b.cfg.MaxBytesPerCycle
		partLabel        = strconv.Itoa(int(ps.partition))
		consumedBytes    uint64
		startOffset      kgo.Offset
		init             bool
		writer           *writer
		lastRec          *kgo.Record
		end              time.Time
		processedRecords int
	)

	startOffset = ps.getStartOffset()

	level.Info(b.logger).Log(
		"msg", "consuming partition",
		"partition", ps.partition,
		"commit_offset", ps.commitOffset,
		"start_offset", startOffset,
	)
	// We always rewind the partition's offset to the commit offset by reassigning the partition to the client (this triggers partition assignment).
	// This is so the cycle started exactly at the commit offset, and not at what was (potentially over-) consumed previously.
	// In the end, we remove the partition from the client (refer to the defer below) to guarantee the client always consumes
	// from one partition at a time. I.e. when this partition is consumed, we start consuming the next one.
	b.kafkaClient.AddConsumePartitions(map[string]map[int32]kgo.Offset{
		topic: {
			ps.partition: startOffset,
		},
	})
	defer b.kafkaClient.RemoveConsumePartitions(map[string][]int32{topic: {ps.partition}})

outer:
	for {
		fetches := func() kgo.Fetches {
			defer func(t time.Time) { metricFetchDuration.WithLabelValues(partLabel).Observe(time.Since(t).Seconds()) }(time.Now())
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
			metricFetchErrors.WithLabelValues(partLabel).Inc()
			return time.Time{}, commitOffsetAtEnd, err
		}

		if fetches.Empty() {
			break
		}

		for iter := fetches.RecordIter(); !iter.Done(); {
			rec := iter.Next()
			metricFetchBytesTotal.WithLabelValues(partLabel).Add(float64(len(rec.Value)))
			metricFetchRecordsTotal.WithLabelValues(partLabel).Inc()
			recordSizeBytes := uint64(len(rec.Value))

			level.Debug(b.logger).Log(
				"msg", "processing record",
				"partition", rec.Partition,
				"offset", rec.Offset,
				"timestamp", rec.Timestamp,
				"len", recordSizeBytes,
			)

			// Initialize on first record
			if !init {
				end = rec.Timestamp.Add(dur) // When block will be cut
				// Record lag at the start of the consumption
				ingest.SetPartitionLagSeconds(group, ps.partition, time.Since(rec.Timestamp))
				writer = newPartitionSectionWriter(
					b.logger,
					uint64(ps.partition),
					uint64(rec.Offset),
					rec.Timestamp,
					dur,
					b.cfg.WAL.IngestionSlack,
					b.cfg.BlockConfig,
					b.overrides,
					b.wal,
					b.enc)
				init = true

				// TODO(mapno): This call creates a link to the parent span in this trace.
				//  While this creates a redundant self-reference in the trace visualization,
				//  the functionality still works correctly. Low priority to fix.
				span.AddLink(trace.LinkFromContext(rec.Context))
			}

			if rec.Timestamp.After(end) {
				break outer
			}

			err := b.pushTraces(rec.Timestamp, rec.Key, rec.Value, writer)
			if err != nil {
				return time.Time{}, commitOffsetAtEnd, err
			}

			processedRecords++
			lastRec = rec
			consumedBytes += recordSizeBytes

			if maxBytesPerCycle > 0 && consumedBytes >= maxBytesPerCycle {
				level.Debug(b.logger).Log(
					"msg", "max bytes per cycle reached",
					"partition", ps.partition,
					"timestamp", rec.Timestamp,
				)
				span.AddEvent("max bytes per cycle reached", trace.WithAttributes(
					attribute.Int64("maxBytesPerCycle", int64(maxBytesPerCycle)),
					attribute.Int64("consumedBytes", int64(consumedBytes))),
				)
				break outer
			}
		}
	}

	if lastRec == nil {
		// Received no data
		level.Info(b.logger).Log(
			"msg", "no data",
			"partition", ps.partition,
			"commit_offset", ps.commitOffset,
			"start_offset", startOffset,
		)
		span.AddEvent("no data")
		// No data means we are caught up
		ingest.SetPartitionLagSeconds(group, ps.partition, 0)
		return time.Time{}, commitOffsetAtEnd, nil
	}

	// Record lag at the end of the consumption
	ingest.SetPartitionLagSeconds(group, ps.partition, time.Since(lastRec.Timestamp))

	err = writer.flush(ctx, b.reader, b.writer, b.compactor)
	if err != nil {
		return time.Time{}, commitOffsetAtEnd, err
	}

	offset := kadm.NewOffsetFromRecord(lastRec)
	err = b.commitOffset(ctx, offset, group, ps.partition)
	if err != nil {
		return time.Time{}, commitOffsetAtEnd, err
	}

	level.Info(b.logger).Log(
		"msg", "successfully committed offset to kafka",
		"partition", ps.partition,
		"commit_offset", offset.At,
		"processed_records", processedRecords,
	)

	writer.allowCompaction(ctx, b.writer)

	return lastRec.Timestamp, offset.At, nil
}

func (b *BlockBuilder) commitOffset(ctx context.Context, offset kadm.Offset, group string, partition int32) error {
	offsets := make(kadm.Offsets)
	offsets.Add(offset)

	trace.SpanFromContext(ctx).AddEvent("committing offset", trace.WithAttributes(attribute.Int64("at", offset.At)))

	boff := backoff.New(ctx, backoff.Config{
		MinBackoff: 100 * time.Millisecond,
		MaxBackoff: time.Minute,
		MaxRetries: 10,
	})
	for boff.Ongoing() {
		err := b.kadm.CommitAllOffsets(ctx, group, offsets)
		if err == nil {
			break
		}
		ingest.HandleKafkaError(err, b.kafkaClient.ForceMetadataRefresh)
		level.Warn(b.logger).Log(
			"msg", "failed to commit offset, retrying",
			"err", err,
			"partition", partition,
			"commit_offset", offset.At,
		)
		boff.Wait()
	}
	if err := boff.ErrCause(); err != nil {
		return fmt.Errorf("error committing offset %d for partition %d, it won't be retried: %w", offset.At, partition, err)
	}
	return nil
}

func formatActivePartitions(partitions []int32) string {
	var strArr []string
	for _, v := range partitions {
		strArr = append(strArr, strconv.Itoa(int(v)))
	}
	return strings.Join(strArr, ",")
}

// It fetches all the offsets for the blockbuilder topic, for each owned partitions it calculates their last committed records and the
// end record offset. Based on that it sort the partitions by lag
func (b *BlockBuilder) fetchPartitions(ctx context.Context, partitions []int32) ([]partitionState, error) {
	var (
		ps          = make([]partitionState, 0, len(partitions))
		commits     kadm.OffsetResponses
		endsOffsets kadm.ListedOffsets
		err         error
	)

	boff := backoff.New(ctx, backoff.Config{
		MinBackoff: 100 * time.Millisecond,
		MaxBackoff: time.Minute,
		MaxRetries: 5,
	})
	for boff.Ongoing() {
		commits, endsOffsets, err = b.getPartitionOffsets(ctx, partitions)
		if err == nil {
			break
		}
		ingest.HandleKafkaError(err, b.kafkaClient.ForceMetadataRefresh)
		boff.Wait()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch partition offsets: %w", err)
	}
	for _, partition := range partitions {
		p := b.getPartitionState(partition, commits, endsOffsets)
		ps = append(ps, p)
	}

	return ps, nil
}

// todo: this function fetches the offsets for all the partitions including the ones that are not assigned to this block builder.
// improve it to only fetch the offsets for the assigned partitions
func (b *BlockBuilder) getPartitionOffsets(ctx context.Context, partitionIDs []int32) (kadm.OffsetResponses, kadm.ListedOffsets, error) {
	var (
		topic = b.cfg.IngestStorageConfig.Kafka.Topic
		group = b.cfg.IngestStorageConfig.Kafka.ConsumerGroup
	)
	commits, err := b.kadm.FetchOffsetsForTopics(ctx, group, topic)
	if err != nil {
		return nil, nil, err
	}
	if err := commits.Error(); err != nil {
		return nil, nil, err
	}

	endsOffsets, err := b.partitionOffsetClient.FetchPartitionsLastProducedOffsets(ctx, partitionIDs)
	if err != nil {
		return nil, nil, err
	}
	if err := endsOffsets.Error(); err != nil {
		return nil, nil, err
	}

	return commits, endsOffsets, nil
}

// Returns the existing state of a partition. Including the last committed record and the last one
func (b *BlockBuilder) getPartitionState(partition int32, commits kadm.OffsetResponses, endsOffsets kadm.ListedOffsets) partitionState {
	var (
		topic = b.cfg.IngestStorageConfig.Kafka.Topic
		ps    = partitionState{partition: partition, commitOffset: commitOffsetAtEnd, endOffset: emptyPartitionEndOffset}
	)

	lastCommit, found := commits.Lookup(topic, partition)
	if found {
		ps.commitOffset = lastCommit.At
	}

	lastRecord, found := endsOffsets.Lookup(topic, partition)
	if found {
		ps.endOffset = lastRecord.Offset
	}

	return ps
}

func (b *BlockBuilder) stopping(err error) error {
	select {
	case <-b.consumeStopped:
	case <-time.After(60 * time.Second):
		// 60s is the default terminationGracePeriod for the BlockBuilder's statefulSet
		level.Error(b.logger).Log("msg", "failed to gracefully stop", "err", err)
	}
	if b.kafkaClient != nil {
		b.kafkaClient.Close()
	}
	return err
}

func (b *BlockBuilder) pushTraces(ts time.Time, tenantBytes, reqBytes []byte, p partitionSectionWriter) error {
	req, err := b.decoder.Decode(reqBytes)
	if err != nil {
		return fmt.Errorf("failed to decode trace: %w", err)
	}
	defer b.decoder.Reset()

	return p.pushBytes(ts, string(tenantBytes), req)
}

// Gets assigned partitions, these can be active or inactive. Pending partitions won't be included
func (b *BlockBuilder) getAssignedPartitions() []int32 {
	partitions := b.partitionRing.PartitionRing().Partitions()
	ringAssignedPartitions := make(map[int32]string, len(partitions))
	for _, p := range partitions {
		if p.IsActive() || p.IsInactive() {
			ringAssignedPartitions[p.Id] = p.GetState().String()
		}
	}
	assignedActivePartitions := make([]int32, 0, len(b.cfg.AssignedPartitions[b.cfg.InstanceID]))
	for _, partition := range b.cfg.AssignedPartitions[b.cfg.InstanceID] {
		if s, ok := ringAssignedPartitions[partition]; ok {
			metricOwnedPartitions.WithLabelValues(strconv.Itoa(int(partition)), s).Set(1)
			assignedActivePartitions = append(assignedActivePartitions, partition)
		}
	}
	return assignedActivePartitions
}
