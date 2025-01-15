package blockbuilder

import (
	"bytes"
	"context"
	"slices"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/tempo/modules/blockbuilder/util"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/pkg/livetraces"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/tracesizes"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/atomic"
)

var metricBlockBuilderFlushedBlocks = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "tempo",
		Subsystem: "block_builder",
		Name:      "flushed_blocks",
	}, []string{"tenant"},
)

const (
	reasonTraceTooLarge = "trace_too_large"
	flushConcurrency    = 4
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

	liveTraces *livetraces.LiveTraces
	traceSizes *tracesizes.Tracker
}

func newTenantStore(tenantID string, partitionID, endTimestamp uint64, cfg BlockConfig, logger log.Logger, wal *wal.WAL, enc encoding.VersionedEncoding, o Overrides) (*tenantStore, error) {
	s := &tenantStore{
		tenantID:     tenantID,
		idGenerator:  util.NewDeterministicIDGenerator(tenantID, partitionID, endTimestamp),
		cfg:          cfg,
		logger:       logger,
		overrides:    o,
		wal:          wal,
		headBlockMtx: sync.Mutex{},
		blocksMtx:    sync.Mutex{},
		enc:          enc,
		liveTraces:   livetraces.New(),
		traceSizes:   tracesizes.New(),
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

func (s *tenantStore) AppendTrace(traceID []byte, tr *tempopb.Trace, ts time.Time) error {
	maxSz := s.overrides.MaxBytesPerTrace(s.tenantID)

	for _, b := range tr.ResourceSpans {
		if maxSz > 0 && !s.traceSizes.Allow(traceID, b.Size(), maxSz) {
			// Record dropped spans due to trace too large
			count := 0
			for _, ss := range b.ScopeSpans {
				count += len(ss.Spans)
			}
			overrides.RecordDiscardedSpans(count, reasonTraceTooLarge, s.tenantID)
			continue
		}

		s.liveTraces.PushWithTimestamp(ts, traceID, b, 0)
	}
	return nil
}

func (s *tenantStore) CutIdle(since time.Time, immediate bool) error {
	idle := s.liveTraces.CutIdle(since, immediate)

	slices.SortFunc(idle, func(a, b *livetraces.LiveTrace) int {
		return bytes.Compare(a.ID, b.ID)
	})

	for _, e := range idle {
		tr := &tempopb.Trace{
			ResourceSpans: e.Batches,
		}

		// Get trace timestamp bounds
		var start, end uint64
		for _, b := range tr.ResourceSpans {
			for _, ss := range b.ScopeSpans {
				for _, s := range ss.Spans {
					if start == 0 || s.StartTimeUnixNano < start {
						start = s.StartTimeUnixNano
					}
					if s.EndTimeUnixNano > end {
						end = s.EndTimeUnixNano
					}
				}
			}
		}

		// Convert from unix nanos to unix seconds
		startSeconds := uint32(start / uint64(time.Second))
		endSeconds := uint32(end / uint64(time.Second))

		if err := s.headBlock.AppendTrace(e.ID, tr, startSeconds, endSeconds); err != nil {
			return err
		}
	}

	err := s.headBlock.Flush()
	if err != nil {
		return err
	}

	// Cut head block if needed
	return s.cutHeadBlock(false)
}

func (s *tenantStore) Flush(ctx context.Context, store tempodb.Writer) error {
	// TODO - Advance some of this work if possible

	// Cut head block
	if err := s.cutHeadBlock(true); err != nil {
		return err
	}

	s.blocksMtx.Lock()
	defer s.blocksMtx.Unlock()

	var (
		completeBlocks = make([]tempodb.WriteableBlock, len(s.walBlocks))
		jobErr         = atomic.NewError(nil)
		wg             = boundedwaitgroup.New(flushConcurrency)
	)

	// Convert WALs to backend blocks
	for i, block := range s.walBlocks {
		wg.Add(1)
		go func(i int, block common.WALBlock) {
			defer wg.Done()

			completeBlock, err := s.buildWriteableBlock(ctx, block)
			if err != nil {
				jobErr.Store(err)
				return
			}

			err = block.Clear()
			if err != nil {
				jobErr.Store(err)
				return
			}

			completeBlocks[i] = completeBlock
		}(i, block)
	}

	wg.Wait()
	if err := jobErr.Load(); err != nil {
		return err
	}

	// Write all blocks to the store
	level.Info(s.logger).Log("msg", "writing blocks to storage", "num_blocks", len(completeBlocks))
	for _, block := range completeBlocks {
		wg.Add(1)
		go func(block tempodb.WriteableBlock) {
			defer wg.Done()
			level.Info(s.logger).Log("msg", "writing block to storage", "block_id", block.BlockMeta().BlockID.String())
			if err := store.WriteBlock(ctx, block); err != nil {
				jobErr.Store(err)
				return
			}

			metricBlockBuilderFlushedBlocks.WithLabelValues(s.tenantID).Inc()

			if err := s.wal.LocalBackend().ClearBlock((uuid.UUID)(block.BlockMeta().BlockID), s.tenantID); err != nil {
				jobErr.Store(err)
			}
		}(block)
	}

	wg.Wait()
	if err := jobErr.Load(); err != nil {
		return err
	}

	// Clear the blocks
	s.walBlocks = s.walBlocks[:0]

	return nil
}

func (s *tenantStore) buildWriteableBlock(ctx context.Context, b common.WALBlock) (tempodb.WriteableBlock, error) {
	level.Debug(s.logger).Log("msg", "building writeable block", "block_id", b.BlockMeta().BlockID.String())

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
