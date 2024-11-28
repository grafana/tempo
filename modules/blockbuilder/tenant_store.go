package blockbuilder

import (
	"context"
	"sync"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
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
	}, []string{"tenant_id"},
)

// TODO - This needs locking
type tenantStore struct {
	tenantID    string
	idGenerator util.IDGenerator

	cfg       BlockConfig
	logger    log.Logger
	overrides Overrides
	enc       encoding.VersionedEncoding

	wal *wal.WAL

	headBlockMtx sync.Mutex
	headBlock    common.WALBlock

	blocksMtx sync.Mutex
	walBlocks []common.WALBlock
}

func newTenantStore(tenantID string, partitionID, endTimestamp int64, cfg BlockConfig, logger log.Logger, wal *wal.WAL, enc encoding.VersionedEncoding, o Overrides) (*tenantStore, error) {
	s := &tenantStore{
		tenantID:     tenantID,
		idGenerator:  util.NewDeterministicIDGenerator(partitionID, endTimestamp),
		cfg:          cfg,
		logger:       logger,
		overrides:    o,
		wal:          wal,
		headBlockMtx: sync.Mutex{},
		blocksMtx:    sync.Mutex{},
		enc:          enc,
	}

	return s, s.resetHeadBlock()
}

// TODO - periodically flush
func (s *tenantStore) cutHeadBlock(immediate bool) error {
	s.headBlockMtx.Lock()
	defer s.headBlockMtx.Unlock()

	dataLen := s.headBlock.DataLength()

	if s.headBlock == nil || dataLen == 0 {
		return nil
	}

	if !immediate && dataLen < s.cfg.MaxBlockBytes {
		return nil
	}

	s.blocksMtx.Lock()
	defer s.blocksMtx.Unlock()

	if err := s.headBlock.Flush(); err != nil {
		return err
	}
	s.walBlocks = append(s.walBlocks, s.headBlock)
	s.headBlock = nil

	return s.resetHeadBlock()
}

func (s *tenantStore) resetHeadBlock() error {
	meta := &backend.BlockMeta{
		BlockID:           s.idGenerator.NewID(),
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
	// TODO - Do this async, it slows down consumption
	if err := s.cutHeadBlock(false); err != nil {
		return err
	}

	return s.headBlock.AppendTrace(traceID, tr, start, end)
}

func (s *tenantStore) Flush(ctx context.Context, store tempodb.Writer) error {
	// TODO - Advance some of this work if possible

	// Cut head block
	if err := s.cutHeadBlock(true); err != nil {
		return err
	}

	s.blocksMtx.Lock()
	defer s.blocksMtx.Unlock()

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
	for _, block := range s.walBlocks {
		if err := s.wal.LocalBackend().ClearBlock((uuid.UUID)(block.BlockMeta().BlockID), s.tenantID); err != nil {
			return err
		}
	}
	s.walBlocks = s.walBlocks[:0]

	return nil
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
