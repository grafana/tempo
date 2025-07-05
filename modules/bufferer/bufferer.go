package bufferer

import (
	"context"
	"fmt"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/kv"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/modules/ingester"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/prometheus/client_golang/prometheus"
)

type Buffer struct {
	services.Service

	cfg    Config
	logger log.Logger

	subservicesWatcher *services.FailureWatcher

	ingestPartitionID         int32
	ingestPartitionLifecycler *ring.PartitionInstanceLifecycler
}

func New(cfg Config, logger log.Logger, reg prometheus.Registerer, singlePartition bool) (*Buffer, error) {
	b := &Buffer{
		cfg:    cfg,
		logger: logger,
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

func (b *Buffer) starting(context.Context) error { return nil }

func (b *Buffer) running(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return nil
	case err := <-b.subservicesWatcher.Chan():
		return fmt.Errorf("bufferer subservice failed: %w", err)
	}
}

func (b *Buffer) stopping(err error) error { return err }
