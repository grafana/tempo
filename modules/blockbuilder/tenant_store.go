package blockbuilder

import (
	"context"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/modules/blockbuilder/util"
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

// TODO - This needs locking
type tenantStore struct {
	tenantID string
	ts       int64

	cfg    BlockConfig
	logger log.Logger

	overrides Overrides
	wal       *wal.WAL
	headBlock common.WALBlock
	walBlocks []common.WALBlock
	enc       encoding.VersionedEncoding
}

func newTenantStore(tenantID string, ts int64, cfg BlockConfig, logger log.Logger, wal *wal.WAL, enc encoding.VersionedEncoding, o Overrides) (*tenantStore, error) {
	s := &tenantStore{
		tenantID:  tenantID,
		ts:        ts,
		cfg:       cfg,
		logger:    logger,
		overrides: o,
		wal:       wal,
		enc:       enc,
	}

	return s, s.resetHeadBlock()
}

func (s *tenantStore) cutHeadBlock() error {
	// Flush the current head block if it exists
	if s.headBlock != nil {
		if err := s.headBlock.Flush(); err != nil {
			return err
		}
		s.walBlocks = append(s.walBlocks, s.headBlock)
		s.headBlock = nil
	}

	return nil
}

func (s *tenantStore) newUUID() backend.UUID {
	return backend.UUID(util.NewDeterministicID(s.ts, int64(len(s.walBlocks))))
}

func (s *tenantStore) resetHeadBlock() error {
	meta := &backend.BlockMeta{
		BlockID:           s.newUUID(),
		TenantID:          s.tenantID,
		DedicatedColumns:  s.overrides.DedicatedColumns(s.tenantID),
		ReplicationFactor: backend.MetricsGeneratorReplicationFactor,
	}
	block, err := s.wal.NewBlock(meta, model.CurrentEncoding)
	if err != nil {
		return err
	}
	s.headBlock = block
	return nil
}

func (s *tenantStore) AppendTrace(traceID []byte, tr *tempopb.Trace, start, end uint32) error {
	// TODO - Do this async? This slows down consumption, but we need to be precise
	if s.headBlock.DataLength() > s.cfg.MaxBlockBytes {
		if err := s.resetHeadBlock(); err != nil {
			return err
		}
	}

	return s.headBlock.AppendTrace(traceID, tr, start, end)
}

func (s *tenantStore) Flush(ctx context.Context, store tempodb.Writer) error {
	// TODO - Advance some of this work if possible

	// Cut head block
	if err := s.cutHeadBlock(); err != nil {
		return err
	}
	completeBlocks := make([]tempodb.WriteableBlock, 0, len(s.walBlocks))
	// Write all blocks
	for _, block := range s.walBlocks {
		completeBlock, err := s.buildWriteableBlock(ctx, block)
		if err != nil {
			return err
		}
		completeBlocks = append(completeBlocks, completeBlock)
	}

	level.Info(s.logger).Log("msg", "writing blocks to storage", "num_blocks", len(completeBlocks))
	// Write all blocks to the store
	for _, block := range completeBlocks {
		level.Info(s.logger).Log("msg", "writing block to storage", "block_id", block.BlockMeta().BlockID.String())
		if err := store.WriteBlock(ctx, block); err != nil {
			return err
		}
		metricBlockBuilderFlushedBlocks.WithLabelValues(s.tenantID).Inc()
	}

	// Clear the blocks
	s.walBlocks = s.walBlocks[:0]

	return s.resetHeadBlock()
}

func (s *tenantStore) buildWriteableBlock(ctx context.Context, b common.WALBlock) (tempodb.WriteableBlock, error) {
	iter, err := b.Iterator()
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	reader, writer := backend.NewReader(s.wal.LocalBackend()), backend.NewWriter(s.wal.LocalBackend())

	newMeta, err := s.enc.CreateBlock(ctx, &s.cfg.BlockCfg, b.BlockMeta(), iter, reader, writer)
	if err != nil {
		return nil, err
	}

	newBlock, err := s.enc.OpenBlock(newMeta, reader)
	if err != nil {
		return nil, err
	}

	return NewWriteableBlock(newBlock, reader, writer), nil
}
