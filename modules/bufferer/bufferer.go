package bufferer

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/backoff"
	"github.com/grafana/dskit/kv"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/modules/ingester"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	ConsumerGroup       = "bufferer"
	buffererServiceName = "bufferer"
)

type Buffer struct {
	services.Service

	cfg    Config
	logger log.Logger
	reg    prometheus.Registerer

	subservicesWatcher *services.FailureWatcher

	ingestPartitionID         int32
	ingestPartitionLifecycler *ring.PartitionInstanceLifecycler

	client      *kgo.Client
	adminClient *kadm.Client
	decoder     *ingest.Decoder

	reader *PartitionReader
}

func New(cfg Config, logger log.Logger, reg prometheus.Registerer, singlePartition bool) (*Buffer, error) {
	b := &Buffer{
		cfg:     cfg,
		logger:  logger,
		reg:     reg,
		decoder: ingest.NewDecoder(),
	}

	var err error
	if singlePartition {
		// For single-binary don't require hostname to identify a partition.
		// Assume partition 0.
		b.ingestPartitionID = 0
	} else {
		b.ingestPartitionID, err = ingest.IngesterPartitionID(cfg.LifecyclerConfig.ID)
		if err != nil {
			return nil, fmt.Errorf("calculating ingester partition ID: %w", err)
		}
	}

	partitionRingKV := cfg.PartitionRing.KVStore.Mock
	if partitionRingKV == nil {
		partitionRingKV, err = kv.NewClient(cfg.PartitionRing.KVStore, ring.GetPartitionRingCodec(), kv.RegistererWithKVName(reg, ingester.PartitionRingName+"-lifecycler"), logger)
		if err != nil {
			return nil, fmt.Errorf("creating KV store for ingester partition ring: %w", err)
		}
	}

	b.ingestPartitionLifecycler = ring.NewPartitionInstanceLifecycler(
		b.cfg.PartitionRing.ToLifecyclerConfig(b.ingestPartitionID, cfg.LifecyclerConfig.ID),
		ingester.PartitionRingName,
		ingester.PartitionRingKey,
		partitionRingKV,
		logger,
		prometheus.WrapRegistererWithPrefix("tempo_", reg))

	b.subservicesWatcher = services.NewFailureWatcher()
	b.subservicesWatcher.WatchService(b.ingestPartitionLifecycler)

	b.Service = services.NewBasicService(b.starting, b.running, b.stopping)

	return b, nil
}

func (b *Buffer) starting(ctx context.Context) error {
	var err error
	b.client, err = ingest.NewReaderClient(
		b.cfg.IngestConfig.Kafka,
		ingest.NewReaderClientMetrics(buffererServiceName, b.reg),
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
		err := b.client.Ping(ctx)
		if err == nil {
			break
		}
		level.Warn(b.logger).Log("msg", "ping kafka; will retry", "err", err)
		boff.Wait()
	}
	if err := boff.ErrCause(); err != nil {
		return fmt.Errorf("failed to ping kafka: %w", err)
	}

	b.reader, err = NewPartitionReaderForPusher(b.client, b.ingestPartitionID, b.cfg.IngestConfig.Kafka.Topic, b.consume, b.logger, b.reg)
	if err != nil {
		return fmt.Errorf("failed to create partition reader: %w", err)
	}

	if err := b.reader.start(ctx); err != nil {
		return fmt.Errorf("failed to start partition reader: %w", err)
	}

	return nil
}

func (b *Buffer) running(ctx context.Context) error {
	go func() {
		_ = b.reader.run(ctx)
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-b.subservicesWatcher.Chan():
		return fmt.Errorf("bufferer subservice failed: %w", err)
	}
}

func (b *Buffer) stopping(err error) error {
	return b.reader.stop(err)
}

func (b *Buffer) consume(_ context.Context, rs []record) error {
	for _, record := range rs {
		_ = level.Info(b.logger).Log("msg", "consuming record", "tenant", record.tenantID)
		_, err := b.decoder.Decode(record.content)
		if err != nil {
			return fmt.Errorf("decoding record: %w", err)
		}
	}
	return nil
}
