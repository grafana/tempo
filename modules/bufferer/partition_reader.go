package bufferer

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/backoff"
	"github.com/grafana/dskit/multierror"
	"github.com/grafana/dskit/services"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/plugin/kprom"
)

type record struct {
	tenantID string
	content  []byte
}

type consumeFn func(context.Context, []record) error

type PartitionReader struct {
	services.Service

	partitionID int32
	topic       string

	curCursor atomic.Int64

	client *kgo.Client
	adm    *kadm.Client

	consume consumeFn
	metrics partitionReaderMetrics

	logger log.Logger

	// TODO: Sync committer ?
}

func NewPartitionReaderForPusher(client *kgo.Client, partitionID int32, topic string, consume consumeFn, logger log.Logger, reg prometheus.Registerer) (*PartitionReader, error) {
	metrics := newPartitionReaderMetrics(partitionID, reg)
	return newPartitionReader(client, partitionID, topic, consume, logger, metrics)
}

func newPartitionReader(client *kgo.Client, partitionID int32, topic string, consume consumeFn, logger log.Logger, metrics partitionReaderMetrics) (*PartitionReader, error) {
	r := &PartitionReader{
		partitionID: partitionID,
		topic:       topic,
		curCursor:   atomic.Int64{},
		client:      client,
		adm:         kadm.NewClient(client),
		consume:     consume,
		metrics:     metrics,
		logger:      log.With(logger, "partition", partitionID),
	}

	r.curCursor.Store(-2) // TODO: Pass cursor from bufferer

	r.Service = services.NewBasicService(r.start, r.run, r.stop)
	return r, nil
}

func (r *PartitionReader) start(context.Context) error {
	return nil
}

func (r *PartitionReader) run(ctx context.Context) error {
	consumeCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	offset, err := r.fetchLastCommittedOffsetWithRetries(consumeCtx)
	if err != nil {
		return fmt.Errorf("failed to fetch last committed offset: %w", err)
	}
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
		r.consumeFetches(consumeCtx, fetches)
	}

	return nil
}

func (r *PartitionReader) stop(error) error {
	level.Info(r.logger).Log("msg", "stopping partition reader")

	r.client.Close()

	return nil
}

func collectFetchErrs(fetches kgo.Fetches) (_ error) {
	mErr := multierror.New()
	fetches.EachError(func(s string, i int32, err error) {
		// kgo advises to "restart" the kafka client if the returned error is a kerr.Error.
		// Recreating the client would cause duplicate metrics registration, so we don't do it for now.
		mErr.Add(err)
	})
	return mErr.Err()
}

func (r *PartitionReader) consumeFetches(ctx context.Context, fetches kgo.Fetches) {
	records := make([]record, 0, len(fetches.Records()))

	var (
		minOffset = int64(math.MaxInt64)
		maxOffset = int64(0)
	)
	fetches.EachRecord(func(r *kgo.Record) {
		minOffset = min(minOffset, r.Offset)
		maxOffset = max(maxOffset, r.Offset)
		records = append(records, record{
			content:  r.Value,
			tenantID: string(r.Key),
		})
	})

	err := r.consume(ctx, records)
	if err != nil {
		level.Error(r.logger).Log("msg", "encountered error processing records; skipping", "min_offset", minOffset, "max_offset", maxOffset, "err", err)
		// TODO abort ingesting & back off if it's a server error, ignore error if it's a client error
	}

	r.curCursor.Store(maxOffset)
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
	// TODO(mapno): Take the highest of either?
	cursor := r.curCursor.Load()
	if cursor > 0 {
		return kgo.NewOffset().At(cursor), nil
	}

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
	offsets, err := r.adm.FetchOffsets(ctx, ConsumerGroup)
	if errors.Is(err, kerr.UnknownTopicOrPartition) {
		// In case we are booting up for the first time ever against this topic.
		return kgo.NewOffset().AtStart(), nil
	}
	if err != nil {
		return kgo.NewOffset(), errors.Wrap(err, "unable to fetch group offsets")
	}
	offset, found := offsets.Lookup(r.topic, r.partitionID)
	if !found {
		// No committed offset found for this partition, start from the end
		return kgo.NewOffset().AtStart(), nil
	}
	return kgo.NewOffset().At(offset.At), nil
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
