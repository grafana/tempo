package livestore

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/backoff"
	"github.com/grafana/dskit/multierror"
	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/plugin/kprom"
)

type recordIter interface {
	Next() *kgo.Record
	Done() bool
}

type consumeFn func(context.Context, recordIter, time.Time) (*kadm.Offset, error)

type PartitionReader struct {
	services.Service

	partitionID   int32
	consumerGroup string
	topic         string

	client *kgo.Client
	adm    *kadm.Client

	lookbackPeriod  time.Duration
	commitInterval  time.Duration
	wg              sync.WaitGroup
	offsetWatermark atomic.Pointer[kadm.Offset]

	consume consumeFn
	metrics partitionReaderMetrics

	logger log.Logger
}

func NewPartitionReaderForPusher(client *kgo.Client, partitionID int32, cfg ingest.KafkaConfig, commitInterval time.Duration, lookbackPeriod time.Duration, consume consumeFn, logger log.Logger, reg prometheus.Registerer) (*PartitionReader, error) {
	metrics := newPartitionReaderMetrics(partitionID, reg)
	return newPartitionReader(client, partitionID, cfg, commitInterval, lookbackPeriod, consume, logger, metrics)
}

func newPartitionReader(client *kgo.Client, partitionID int32, cfg ingest.KafkaConfig, commitInterval time.Duration, lookbackPeriod time.Duration, consume consumeFn, logger log.Logger, metrics partitionReaderMetrics) (*PartitionReader, error) {
	r := &PartitionReader{
		partitionID:    partitionID,
		consumerGroup:  cfg.ConsumerGroup,
		topic:          cfg.Topic,
		client:         client,
		adm:            kadm.NewClient(client),
		lookbackPeriod: lookbackPeriod,
		commitInterval: commitInterval,
		consume:        consume,
		metrics:        metrics,
		logger:         log.With(logger, "partition", partitionID),
	}

	r.Service = services.NewBasicService(r.start, r.running, r.stop)
	return r, nil
}

func (r *PartitionReader) start(context.Context) error {
	return nil
}

func (r *PartitionReader) running(ctx context.Context) error {
	offset, err := r.fetchLastCommittedOffsetWithRetries(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch last committed offset: %w", err)
	}

	r.wg.Add(1)
	go r.commitLoop(ctx)

	r.client.AddConsumePartitions(map[string]map[int32]kgo.Offset{r.topic: {r.partitionID: offset}})
	defer r.client.RemoveConsumePartitions(map[string][]int32{r.topic: {r.partitionID}})

	for ctx.Err() == nil {
		fetches := r.client.PollFetches(ctx)
		if fetches.Err() != nil {
			if errors.Is(fetches.Err(), context.Canceled) {
				return nil
			}
			err := collectFetchErrs(fetches)
			level.Error(r.logger).Log("msg", "encountered error while fetching", "err", err)
			continue
		}

		r.recordFetchesMetrics(fetches)
		if offset := r.consumeFetches(ctx, fetches); offset != nil {
			r.storeOffsetForCommit(ctx, offset)
		}
	}

	return nil
}

func (r *PartitionReader) storeOffsetForCommit(ctx context.Context, offset *kadm.Offset) {
	if r.commitInterval == 0 { // Sync commits
		if err := r.commitOffset(ctx, *offset); err != nil {
			level.Error(r.logger).Log("msg", "failed to commit offset", "offset", offset, "err", err)
		}
	}

	r.offsetWatermark.Store(offset)
}

func (r *PartitionReader) stop(error) error {
	level.Info(r.logger).Log("msg", "stopping partition reader")

	r.wg.Wait()

	r.client.Close()

	return nil
}

func (r *PartitionReader) commitLoop(ctx context.Context) {
	defer r.wg.Done()

	if r.commitInterval == 0 { // Sync commits
		return
	}

	t := time.NewTicker(r.commitInterval)
	defer t.Stop()

	var lastCommittedOffset kadm.Offset

	for {
		select {
		case <-ctx.Done():
			// Commit one last time before shutting down
			func() {
				// Detach context with a deadline
				ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(30*time.Second))
				defer cancel()
				r.commitHighWatermark(ctx, lastCommittedOffset)
			}()
			return
		case <-t.C:
			lastCommittedOffset = r.commitHighWatermark(ctx, lastCommittedOffset)
		}
	}
}

func collectFetchErrs(fetches kgo.Fetches) (_ error) {
	mErr := multierror.New()
	fetches.EachError(func(_ string, _ int32, err error) {
		// kgo advises to "restart" the kafka client if the returned error is a kerr.Error.
		// Recreating the client would cause duplicate metrics registration, so we don't do it for now.
		mErr.Add(err)
	})
	return mErr.Err()
}

func (r *PartitionReader) consumeFetches(ctx context.Context, fetches kgo.Fetches) *kadm.Offset {
	// Pass offset and byte information to the live-store
	offset, err := r.consume(ctx, fetches.RecordIter(), time.Now())
	if err != nil {
		// TODO abort ingesting & back off if it's a server error, ignore error if it's a client error
		level.Error(r.logger).Log("msg", "encountered error processing records; skipping", "err", err)
		return nil
	}

	return offset
}

func (r *PartitionReader) recordFetchesMetrics(fetches kgo.Fetches) {
	var (
		now        = time.Now()
		numRecords = 0
	)

	fetches.EachRecord(func(record *kgo.Record) {
		numRecords++
		r.metrics.receiveDelay.Observe(now.Sub(record.Timestamp).Seconds())
	})

	r.metrics.recordsPerFetch.Observe(float64(numRecords))
}

func (r *PartitionReader) fetchLastCommittedOffsetWithRetries(ctx context.Context) (offset kgo.Offset, err error) {
	retry := backoff.New(ctx, backoff.Config{
		MinBackoff: 100 * time.Millisecond,
		MaxBackoff: 2 * time.Second,
		MaxRetries: 10,
	})

	for retry.Ongoing() {
		offset, err = r.fetchLastCommittedOffset(ctx)
		if err == nil {
			return offset, nil
		}

		level.Warn(r.logger).Log("msg", "failed to fetch last committed offset", "err", err)
		retry.Wait()
	}

	// Handle the case the context was canceled before the first attempt.
	if err == nil {
		err = retry.Err()
	}

	return offset, err
}

func (r *PartitionReader) fetchLastCommittedOffset(ctx context.Context) (kgo.Offset, error) {
	offsets, err := r.adm.FetchOffsets(ctx, r.consumerGroup)
	if errors.Is(err, kerr.UnknownTopicOrPartition) {
		// In case we are booting up for the first time ever against this topic.
		return kgo.NewOffset().AtStart(), nil
	}
	if err != nil {
		return kgo.NewOffset(), errors.Wrap(err, "unable to fetch group offsets")
	}
	offset, found := offsets.Lookup(r.topic, r.partitionID)
	if !found { // No committed offset found for this partition
		if r.lookbackPeriod == 0 {
			return kgo.NewOffset().AtEnd(), nil
		}
		return kgo.NewOffset().AfterMilli(time.Now().Add(-r.lookbackPeriod).UnixMilli()), nil
	}
	return r.cutoffOffset(ctx, offset)
}

func (r *PartitionReader) cutoffOffset(ctx context.Context, offset kadm.OffsetResponse) (kgo.Offset, error) {
	offsets, err := r.adm.ListOffsetsAfterMilli(ctx, time.Now().Add(-r.lookbackPeriod).UnixMilli(), r.topic)
	if err != nil {
		return kgo.NewOffset(), err
	}
	cutoffOffset, found := offsets.Lookup(r.topic, r.partitionID)
	if !found || cutoffOffset.Offset < offset.At {
		return kgo.NewOffset().At(offset.At), nil
	}
	return kgo.NewOffset().At(cutoffOffset.Offset), nil
}

func (r *PartitionReader) commitHighWatermark(ctx context.Context, lastCommittedOffset kadm.Offset) kadm.Offset {
	offset := r.offsetWatermark.Load()
	if offset == nil {
		level.Debug(r.logger).Log("msg", "no offset found for committing offset")
		return lastCommittedOffset
	}

	if lastCommittedOffset.At >= offset.At {
		level.Debug(r.logger).Log("msg", "nothing to commit", "lastCommittedOffset", lastCommittedOffset.At, "offset", offset.At)
		return lastCommittedOffset
	}

	if err := r.commitOffset(ctx, *offset); err != nil {
		level.Error(r.logger).Log("msg", "failed to commit kafka offset", "offset", offset.At, "err", err)
		return lastCommittedOffset
	}

	level.Debug(r.logger).Log("msg", "committed kafka offset", "offset", offset.At, "topic", r.topic, "group", r.consumerGroup)

	return *offset
}

func (r *PartitionReader) commitOffset(ctx context.Context, offset kadm.Offset) error {
	// Use the admin client to commit the offset
	offsets := make(kadm.Offsets)
	offsets.Add(offset)

	_, err := r.adm.CommitOffsets(ctx, r.consumerGroup, offsets)
	return err
}

type partitionReaderMetrics struct {
	receiveDelay    prometheus.Histogram
	recordsPerFetch prometheus.Histogram
	kprom           *kprom.Metrics
}

func newPartitionReaderMetrics(partitionID int32, reg prometheus.Registerer) partitionReaderMetrics {
	factory := promauto.With(reg)

	return partitionReaderMetrics{
		receiveDelay: factory.NewHistogram(prometheus.HistogramOpts{
			Name:                        "tempo_ingest_storage_reader_receive_delay_seconds",
			Help:                        "Delay between producing a record and receiving it in the consumer.",
			NativeHistogramBucketFactor: 1.1,
		}),
		recordsPerFetch: factory.NewHistogram(prometheus.HistogramOpts{
			Name:                        "tempo_ingest_storage_reader_records_per_fetch",
			Help:                        "The number of records received by the consumer in a single fetch operation.",
			Buckets:                     prometheus.ExponentialBuckets(1, 2, 15),
			NativeHistogramBucketFactor: 1.1,
		}),
		kprom: kprom.NewMetrics("tempo_ingest_storage_reader",
			kprom.Registerer(prometheus.WrapRegistererWith(prometheus.Labels{"partition": strconv.Itoa(int(partitionID))}, reg)),
			// Do not export the client ID, because we use it to specify options to the backend.
			kprom.FetchAndProduceDetail(kprom.Batches, kprom.Records, kprom.CompressedBytes, kprom.UncompressedBytes)),
	}
}
