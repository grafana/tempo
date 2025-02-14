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
)

const (
	blockBuilderServiceName = "block-builder"
	ConsumerGroup           = "block-builder"
	pollTimeout             = 2 * time.Second
	cutTime                 = 10 * time.Second
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
	metricPartitionLagSeconds = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Subsystem: "block_builder",
		Name:      "partition_lag_seconds",
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

	logger log.Logger
	cfg    Config

	kafkaClient   *kgo.Client
	kadm          *kadm.Client
	decoder       *ingest.Decoder
	partitionRing ring.PartitionRingReader

	overrides Overrides
	enc       encoding.VersionedEncoding
	wal       *wal.WAL // TODO - Shared between tenants, should be per tenant?
	writer    tempodb.Writer
}

type partitionStatus struct {
	partition              int32
	hasRecords             bool
	startOffset, endOffset int64
	lastRecordTs           time.Time
}

func (p partitionStatus) getStartOffset() kgo.Offset {
	if p.startOffset >= 0 {
		return kgo.NewOffset().At(p.startOffset)
	}
	return kgo.NewOffset().AtStart()
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
		logger:        logger,
		cfg:           cfg,
		partitionRing: partitionRing,
		decoder:       ingest.NewDecoder(),
		overrides:     overrides,
		writer:        store,
	}

	b.Service = services.NewBasicService(b.starting, b.running, b.stopping)
	return b, nil
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

	ingest.ExportPartitionLagMetrics(
		ctx,
		b.kadm,
		b.logger,
		b.cfg.IngestStorageConfig,
		b.getAssignedActivePartitions)

	return nil
}

func (b *BlockBuilder) running(ctx context.Context) error {
	for {
		waitTime, err := b.consume(ctx)
		if err != nil {
			level.Error(b.logger).Log("msg", "consumeCycle failed", "err", err)
		}
		select {
		case <-time.After(waitTime):
		case <-ctx.Done():
			return nil
		}
	}
}

// It consumes records for all the asigneed partitions, priorizing the ones with more lag. It keeps consuming until
// all the partitions lag is less than the cycle duration. When that happen it returns time to wait before another consuming cycle, based on the last record timestamp
func (b *BlockBuilder) consume(ctx context.Context) (time.Duration, error) {
	var (
		end        = time.Now()
		partitions = b.getAssignedActivePartitions()
	)
	level.Info(b.logger).Log("msg", "starting consume cycle", "cycle_end", end, "active_partitions", getActivePartitions(partitions))
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
		if !p.hasRecords { // No records, we can skip the partition
			continue
		}
		lastRecordTs, lastRecordOffset, err := b.consumePartition(ctx, p)
		if err != nil {
			return 0, err
		}
		ps[i].lastRecordTs = lastRecordTs
		ps[i].startOffset = lastRecordOffset
	}

	// Iterate over the laggiest partition until the lag is less than the cycle duration or none of the partitions has records
	for {
		sort.Slice(ps, func(i, j int) bool {
			return ps[i].lastRecordTs.After(ps[j].lastRecordTs)
		})

		laggiestPartition := ps[0]
		if laggiestPartition.lastRecordTs.IsZero() {
			return b.cfg.ConsumeCycleDuration, nil
		}

		lagTime := time.Since(laggiestPartition.lastRecordTs)
		if lagTime < b.cfg.ConsumeCycleDuration {
			return time.Second, nil
		}
		level.Info(b.logger).Log("msg", "consuming laggiest partition", "partition", laggiestPartition.partition, "lag", lagTime)
		lastRecordTs, lastRecordOffset, err := b.consumePartition(ctx, laggiestPartition)
		if err != nil {
			return 0, err
		}
		ps[0].lastRecordTs = lastRecordTs
		ps[0].startOffset = lastRecordOffset
	}
}

func (b *BlockBuilder) consumePartition(ctx context.Context, ps partitionStatus) (lastTs time.Time, lastOffset int64, err error) {
	defer func(t time.Time) {
		metricProcessPartitionSectionDuration.WithLabelValues(strconv.Itoa(int(ps.partition))).Observe(time.Since(t).Seconds())
	}(time.Now())

	var (
		dur              = b.cfg.ConsumeCycleDuration
		topic            = b.cfg.IngestStorageConfig.Kafka.Topic
		group            = b.cfg.IngestStorageConfig.Kafka.ConsumerGroup
		partLabel        = strconv.Itoa(int(ps.partition))
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
		"commit_offset", ps.startOffset,
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
			return time.Time{}, -1, err
		}

		if fetches.Empty() {
			break
		}

		for iter := fetches.RecordIter(); !iter.Done(); {
			rec := iter.Next()
			metricFetchBytesTotal.WithLabelValues(partLabel).Add(float64(len(rec.Value)))
			metricFetchRecordsTotal.WithLabelValues(partLabel).Inc()

			level.Debug(b.logger).Log(
				"msg", "processing record",
				"partition", rec.Partition,
				"offset", rec.Offset,
				"timestamp", rec.Timestamp,
				"len", len(rec.Value),
			)

			// Initialize on first record
			if !init {
				end = rec.Timestamp.Add(dur) // When block will be cut
				metricPartitionLagSeconds.WithLabelValues(partLabel).Set(time.Since(rec.Timestamp).Seconds())
				writer = newPartitionSectionWriter(b.logger, uint64(ps.partition), uint64(rec.Offset), rec.Timestamp, dur, b.cfg.WAL.IngestionSlack, b.cfg.BlockConfig, b.overrides, b.wal, b.enc)
				init = true
			}

			if rec.Timestamp.After(end) {
				break outer
			}

			err := b.pushTraces(rec.Timestamp, rec.Key, rec.Value, writer)
			if err != nil {
				return time.Time{}, -1, err
			}
			processedRecords++
			lastRec = rec
		}
	}

	if lastRec == nil {
		// Received no data
		level.Info(b.logger).Log(
			"msg", "no data",
			"partition", ps.partition,
			"commit_offset", ps.startOffset,
			"start_offset", startOffset,
		)
		return time.Time{}, -1, nil
	}

	err = writer.flush(ctx, b.writer)
	if err != nil {
		return time.Time{}, -1, err
	}

	// TODO - Retry commit
	resp, err := b.kadm.CommitOffsets(ctx, group, kadm.OffsetsFromRecords(*lastRec))
	if err != nil {
		return time.Time{}, -1, err
	}
	if err := resp.Error(); err != nil {
		return time.Time{}, -1, err
	}
	level.Info(b.logger).Log(
		"msg", "successfully committed offset to kafka",
		"partition", ps.partition,
		"last_record", lastRec.Offset,
		"processed_records", processedRecords,
	)

	return lastRec.Timestamp, lastRec.Offset, nil
}

func getActivePartitions(partitions []int32) string {
	var strArr []string
	for _, v := range partitions {
		strArr = append(strArr, strconv.Itoa(int(v)))
	}
	return strings.Join(strArr, ",")
}

// It fetches all the offsets for the blockbuilder topic, for each owned partitions it calculates their last committed records and the
// end record offset. Based on that it sort the partitions by lag
func (b *BlockBuilder) fetchPartitions(ctx context.Context, partitions []int32) ([]partitionStatus, error) {
	var (
		ps    = make([]partitionStatus, len(partitions))
		group = b.cfg.IngestStorageConfig.Kafka.ConsumerGroup
		topic = b.cfg.IngestStorageConfig.Kafka.Topic
	)

	commits, err := b.kadm.FetchOffsetsForTopics(ctx, group, topic)
	if err != nil {
		return nil, err
	}
	if err := commits.Error(); err != nil {
		return nil, err
	}

	endsOffsets, err := b.kadm.ListEndOffsets(ctx, topic)
	if err != nil {
		return nil, err
	}
	if err := endsOffsets.Error(); err != nil {
		return nil, err
	}

	for _, partition := range partitions {
		p := b.getPartitionStatus(partition, commits, endsOffsets)
		ps = append(ps, p)
	}

	return ps, nil
}

// Returns the existing status of a partition. Including the last committed record and the last one
func (b *BlockBuilder) getPartitionStatus(partition int32, commits kadm.OffsetResponses, endsOffsets kadm.ListedOffsets) partitionStatus {
	var (
		topic           = b.cfg.IngestStorageConfig.Kafka.Topic
		partitionStatus = partitionStatus{partition: partition, startOffset: -1, endOffset: -1}
	)

	lastCommit, found := commits.Lookup(topic, partition)
	if found {
		partitionStatus.startOffset = lastCommit.At
	}

	lastRecord, found := endsOffsets.Lookup(topic, partition)
	if found {
		partitionStatus.endOffset = lastRecord.Offset
		partitionStatus.hasRecords = true
	}

	return partitionStatus
}

func (b *BlockBuilder) stopping(err error) error {
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
