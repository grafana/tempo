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

	// Clear all previous remnants
	err := b.wal.Clear()
	if err != nil {
		return err
	}

	for _, partition := range partitions {
		// Consume partition while data remains.
		// TODO - round-robin one consumption per partition instead to equalize catch-up time.
		for {
			more, err := b.consumePartition(ctx, partition, end)
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

func (b *BlockBuilder) consumePartition(ctx context.Context, partition int32, overallEnd time.Time) (more bool, err error) {
	defer func(t time.Time) {
		metricProcessPartitionSectionDuration.WithLabelValues(strconv.Itoa(int(partition))).Observe(time.Since(t).Seconds())
	}(time.Now())

	var (
		dur         = b.cfg.ConsumeCycleDuration
		topic       = b.cfg.IngestStorageConfig.Kafka.Topic
		group       = b.cfg.IngestStorageConfig.Kafka.ConsumerGroup
		partLabel   = strconv.Itoa(int(partition))
		startOffset kgo.Offset
		init        bool
		writer      *writer
		lastRec     *kgo.Record
		nextCut     time.Time
		end         time.Time
	)

	commits, err := b.kadm.FetchOffsetsForTopics(ctx, group, topic)
	if err != nil {
		return false, err
	}
	if err := commits.Error(); err != nil {
		return false, err
	}

	lastCommit, ok := commits.Lookup(topic, partition)
	if ok && lastCommit.At >= 0 {
		startOffset = kgo.NewOffset().At(lastCommit.At)
	} else {
		startOffset = kgo.NewOffset().AtStart()
	}

	ends, err := b.kadm.ListEndOffsets(ctx, topic)
	if err != nil {
		return false, err
	}
	if err := ends.Error(); err != nil {
		return false, err
	}
	lastPossibleMessage, lastPossibleMessageFound := ends.Lookup(topic, partition)

	level.Info(b.logger).Log(
		"msg", "consuming partition",
		"partition", partition,
		"commit_offset", lastCommit.At,
		"start_offset", startOffset,
	)

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
			return false, err
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
				writer = newPartitionSectionWriter(b.logger, uint64(partition), uint64(rec.Offset), b.cfg.BlockConfig, b.overrides, b.wal, b.enc)
				nextCut = rec.Timestamp.Add(cutTime)
				init = true
			}

			if rec.Timestamp.After(end) {
				// Cut this block but continue only if we have at least another full cycle
				if overallEnd.Sub(rec.Timestamp) >= dur {
					more = true
				}
				break outer
			}

			if rec.Timestamp.After(overallEnd) {
				break outer
			}

			if rec.Timestamp.After(nextCut) {
				// Cut before appending this trace
				err = writer.cutidle(rec.Timestamp.Add(-cutTime), false)
				if err != nil {
					return false, err
				}
				nextCut = rec.Timestamp.Add(cutTime)
			}

			err := b.pushTraces(rec.Timestamp, rec.Key, rec.Value, writer)
			if err != nil {
				return false, err
			}

			lastRec = rec

			if lastPossibleMessageFound && lastRec.Offset >= lastPossibleMessage.Offset-1 {
				// We reached the end so break now and avoid another poll which is expected to be empty.
				break outer
			}
		}
	}

	if lastRec == nil {
		// Received no data
		level.Info(b.logger).Log(
			"msg", "no data",
			"partition", partition,
		)
		return false, nil
	}

	// Cut any remaining
	err = writer.cutidle(time.Time{}, true)
	if err != nil {
		return false, err
	}

	err = writer.flush(ctx, b.writer)
	if err != nil {
		return false, err
	}

	// TODO - Retry commit
	resp, err := b.kadm.CommitOffsets(ctx, group, kadm.OffsetsFromRecords(*lastRec))
	if err != nil {
		return false, err
	}
	if err := resp.Error(); err != nil {
		return false, err
	}

	level.Info(b.logger).Log(
		"msg", "successfully committed offset to kafka",
		"partition", partition,
		"last_record", lastRec.Offset,
	)

	return more, nil
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
