package blockbuilder

import (
	"context"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var metricBlockBuilderFlushedBlocks = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "tempo",
		Subsystem: "block_builder",
		Name:      "flushed_blocks",
	},
	[]string{"tenant_id"},
)

type tenantStore struct {
	tenantID string
	cfg      BlockConfig
	logger   log.Logger

	overrides Overrides
	wal       *wal.WAL
	headBlock common.WALBlock
	walBlocks []common.WALBlock
	enc       encoding.VersionedEncoding
}

func newTenantStore(tenantID string, cfg BlockConfig, logger log.Logger, wal *wal.WAL, enc encoding.VersionedEncoding, o Overrides) (*tenantStore, error) {
	i := &tenantStore{
		tenantID:  tenantID,
		cfg:       cfg,
		logger:    logger,
		overrides: o,
		wal:       wal,
		enc:       enc,
	}

	return i, i.resetHeadBlock()
}

func (i *tenantStore) cutHeadBlock() error {
	// Flush the current head block if it exists
	if i.headBlock != nil {
		if err := i.headBlock.Flush(); err != nil {
			return err
		}
		i.walBlocks = append(i.walBlocks, i.headBlock)
		i.headBlock = nil
	}

	return nil
}

func (i *tenantStore) resetHeadBlock() error {
	meta := &backend.BlockMeta{
		BlockID:           backend.NewUUID(), // TODO - Deterministic UUID
		TenantID:          i.tenantID,
		DedicatedColumns:  i.overrides.DedicatedColumns(i.tenantID),
		ReplicationFactor: backend.MetricsGeneratorReplicationFactor,
	}
	block, err := i.wal.NewBlock(meta, model.CurrentEncoding)
	if err != nil {
		return err
	}
	i.headBlock = block
	return nil
}

func (i *tenantStore) AppendTrace(traceID []byte, tr *tempopb.Trace, start, end uint32) error {
	// TODO - Do this async? This slows down consumption, but we need to be precise
	if i.headBlock.DataLength() > i.cfg.MaxBlockBytes {
		if err := i.resetHeadBlock(); err != nil {
			return err
		}
	}

	return i.headBlock.AppendTrace(traceID, tr, start, end)
}

func (i *tenantStore) Flush(ctx context.Context, store tempodb.Writer) error {
	// TODO - Advance some of this work if possible

	// Cut head block
	if err := i.cutHeadBlock(); err != nil {
		return err
	}
	completeBlocks := make([]tempodb.WriteableBlock, 0, len(i.walBlocks))
	// Write all blocks
	for _, block := range i.walBlocks {
		completeBlock, err := i.buildWriteableBlock(ctx, block)
		if err != nil {
			return err
		}
		completeBlocks = append(completeBlocks, completeBlock)
	}

	level.Info(i.logger).Log("msg", "writing blocks to storage", "num_blocks", len(completeBlocks))
	// Write all blocks to the store
	for _, block := range completeBlocks {
		level.Info(i.logger).Log("msg", "writing block to storage", "block_id", block.BlockMeta().BlockID.String())
		if err := store.WriteBlock(ctx, block); err != nil {
			return err
		}
		metricBlockBuilderFlushedBlocks.WithLabelValues(i.tenantID).Inc()
	}

	// Clear the blocks
	i.walBlocks = i.walBlocks[:0]

	return i.resetHeadBlock()
}

func (i *tenantStore) buildWriteableBlock(ctx context.Context, b common.WALBlock) (tempodb.WriteableBlock, error) {
	iter, err := b.Iterator()
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	reader, writer := backend.NewReader(i.wal.LocalBackend()), backend.NewWriter(i.wal.LocalBackend())

	newMeta, err := i.enc.CreateBlock(ctx, &i.cfg.BlockCfg, b.BlockMeta(), iter, reader, writer)
	if err != nil {
		return nil, err
	}

	newBlock, err := i.enc.OpenBlock(newMeta, reader)
	if err != nil {
		return nil, err
	}

	return NewWriteableBlock(newBlock, reader, writer), nil
}
