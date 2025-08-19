package livestore

import (
	"context"
	"flag"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/tempo/modules/ingester"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/livetraces"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/tracesizes"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
)

type instance struct {
	tenantID string
	logger   log.Logger

	// Configuration
	Cfg Config

	// WAL and encoding
	wal *wal.WAL
	enc encoding.VersionedEncoding

	// Block management
	blocksMtx      sync.RWMutex
	headBlock      common.WALBlock
	walBlocks      map[uuid.UUID]common.WALBlock
	completeBlocks map[uuid.UUID]*ingester.LocalBlock
	lastCutTime    time.Time

	// Live traces
	liveTracesMtx sync.Mutex
	liveTraces    *livetraces.LiveTraces[*v1.ResourceSpans]
	traceSizes    *tracesizes.Tracker

	// Block offset metadata (set during coordinated cuts)
	// blockOffsetMeta map[uuid.UUID]offsetMetadata // TODO: Used for checking data integrity

	overrides overrides.Interface
}

func newInstance(instanceID string, cfg Config, wal *wal.WAL, overrides overrides.Interface, logger log.Logger) (*instance, error) {
	enc := encoding.DefaultEncoding()

	i := &instance{
		tenantID:       instanceID,
		logger:         log.With(logger, "tenant", instanceID),
		Cfg:            cfg,
		wal:            wal,
		enc:            enc,
		walBlocks:      map[uuid.UUID]common.WALBlock{},
		completeBlocks: map[uuid.UUID]*ingester.LocalBlock{},
		liveTraces:     livetraces.New[*v1.ResourceSpans](func(rs *v1.ResourceSpans) uint64 { return uint64(rs.Size()) }, 30*time.Second, 5*time.Minute),
		traceSizes:     tracesizes.New(),
		overrides:      overrides,
		// blockOffsetMeta:   make(map[uuid.UUID]offsetMetadata),
	}

	err := i.resetHeadBlock()
	if err != nil {
		return nil, err
	}

	return i, nil
}

func (i *instance) pushBytes(ts time.Time, req *tempopb.PushBytesRequest) {
	if len(req.Traces) != len(req.Ids) {
		level.Error(i.logger).Log("msg", "mismatched traces and ids length", "IDs", len(req.Ids), "traces", len(req.Traces))
		return
	}

	// Check tenant limits
	maxBytes := i.overrides.MaxBytesPerTrace(i.tenantID)

	// For each pre-marshalled trace, we need to unmarshal it and push to live traces
	for j, traceBytes := range req.Traces {
		traceID := req.Ids[j]

		// Unmarshal the trace
		trace := &tempopb.Trace{}
		if err := trace.Unmarshal(traceBytes.Slice); err != nil {
			level.Error(i.logger).Log("msg", "failed to unmarshal trace", "err", err)
			continue
		}

		i.liveTracesMtx.Lock()
		// Push each batch in the trace to live traces
		for _, batch := range trace.ResourceSpans {
			if len(batch.ScopeSpans) == 0 || len(batch.ScopeSpans[0].Spans) == 0 {
				continue
			}

			// Push to live traces with tenant-specific limits
			if !i.liveTraces.PushWithTimestampAndLimits(ts, traceID, batch, 0, uint64(maxBytes)) {
				level.Warn(i.logger).Log("msg", "dropped trace due to live traces limit", "tenant", i.tenantID)
				continue
			}
		}
		i.liveTracesMtx.Unlock()
	}
}

func (i *instance) cutIdleTraces(immediate bool) error {
	i.liveTracesMtx.Lock()
	tracesToCut := i.liveTraces.CutIdle(time.Now(), immediate)
	i.liveTracesMtx.Unlock()

	if len(tracesToCut) == 0 {
		return nil
	}

	// Collect the trace IDs that will be flushed
	for _, t := range tracesToCut {
		tr := &tempopb.Trace{
			ResourceSpans: t.Batches,
		}

		err := i.writeHeadBlock(t.ID, tr)
		if err != nil {
			return err
		}
	}

	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()
	if i.headBlock != nil {
		err := i.headBlock.Flush()
		if err != nil {
			return err
		}

		return nil
	}
	return nil
}

func (i *instance) writeHeadBlock(id []byte, tr *tempopb.Trace) error {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	if i.headBlock == nil {
		err := i.resetHeadBlock()
		if err != nil {
			return err
		}
	}

	// Get trace timestamp bounds
	var start, end uint64
	for _, batch := range tr.ResourceSpans {
		for _, ss := range batch.ScopeSpans {
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

	return i.headBlock.AppendTrace(id, tr, startSeconds, endSeconds, false)
}

func (i *instance) resetHeadBlock() error {
	dedicatedColumns := i.overrides.DedicatedColumns(i.tenantID)

	meta := &backend.BlockMeta{
		BlockID:           backend.NewUUID(),
		TenantID:          i.tenantID,
		DedicatedColumns:  dedicatedColumns,
		ReplicationFactor: backend.LiveStoreReplicationFactor,
	}
	block, err := i.wal.NewBlock(meta, model.CurrentEncoding)
	if err != nil {
		return err
	}
	i.headBlock = block
	i.lastCutTime = time.Now()
	return nil
}

func (i *instance) cutBlocks(immediate bool) (uuid.UUID, error) {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	if i.headBlock == nil || i.headBlock.DataLength() == 0 {
		return uuid.Nil, nil
	}

	// TODO: Configurable
	maxBlockDuration := 5 * time.Minute
	maxBlockBytes := uint64(100 * 1024 * 1024) // 100MB

	if !immediate && time.Since(i.lastCutTime) < maxBlockDuration && i.headBlock.DataLength() < maxBlockBytes {
		return uuid.Nil, nil
	}

	// Final flush
	err := i.headBlock.Flush()
	if err != nil {
		return uuid.Nil, err
	}

	id := (uuid.UUID)(i.headBlock.BlockMeta().BlockID)
	i.walBlocks[id] = i.headBlock

	level.Info(i.logger).Log("msg", "queueing wal block for completion", "block", id.String())

	err = i.resetHeadBlock()
	if err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

var blockConfig = common.BlockConfig{}

func init() {
	// TODO MRD this is a hack until we roll the config into the livestore
	blockConfig.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
}

func (i *instance) completeBlock(ctx context.Context, id uuid.UUID) error {
	i.blocksMtx.Lock()
	walBlock := i.walBlocks[id]
	i.blocksMtx.Unlock()

	if walBlock == nil {
		level.Warn(i.logger).Log("msg", "WAL block disappeared before being completed", "id", id)
		return nil
	}

	// Create completed block
	reader := backend.NewReader(i.wal.LocalBackend())
	writer := backend.NewWriter(i.wal.LocalBackend())

	iter, err := walBlock.Iterator()
	if err != nil {
		level.Error(i.logger).Log("msg", "failed to get WAL block iterator", "id", id, "err", err)
		return err
	}
	defer iter.Close()

	newMeta, err := i.enc.CreateBlock(ctx, &blockConfig, walBlock.BlockMeta(), iter, reader, writer)
	if err != nil {
		level.Error(i.logger).Log("msg", "failed to create complete block", "id", id, "err", err)
		return err
	}

	newBlock, err := i.enc.OpenBlock(newMeta, reader)
	if err != nil {
		level.Error(i.logger).Log("msg", "failed to open complete block", "id", id, "err", err)
		return err
	}

	i.blocksMtx.Lock()
	// Verify the WAL block still exists
	if _, ok := i.walBlocks[id]; !ok {
		level.Warn(i.logger).Log("msg", "WAL block disappeared while being completed, deleting complete block", "id", id)
		err := i.wal.LocalBackend().ClearBlock(id, i.tenantID)
		if err != nil {
			level.Error(i.logger).Log("msg", "failed to clear complete block after WAL disappeared", "block", id, "err", err)
		}
		i.blocksMtx.Unlock()
		return nil
	}

	i.completeBlocks[id] = ingester.NewLocalBlock(ctx, newBlock, i.wal.LocalBackend())

	err = walBlock.Clear()
	if err != nil {
		level.Error(i.logger).Log("msg", "failed to clear WAL block", "id", id, "err", err)
	}
	delete(i.walBlocks, (uuid.UUID)(walBlock.BlockMeta().BlockID))
	i.blocksMtx.Unlock()

	level.Info(i.logger).Log("msg", "completed block", "id", id.String())
	return nil
}

func (i *instance) deleteOldBlocks() error {
	i.blocksMtx.Lock()
	defer i.blocksMtx.Unlock()

	cutoff := time.Now().Add(i.Cfg.CompleteBlockTimeout) // Delete blocks older than Complete Block Timeout

	for id, walBlock := range i.walBlocks {
		if walBlock.BlockMeta().EndTime.Before(cutoff) {
			if _, ok := i.completeBlocks[id]; !ok {
				level.Warn(i.logger).Log("msg", "deleting WAL block that was never completed", "block", id.String())
			}
			err := walBlock.Clear()
			if err != nil {
				return err
			}
			delete(i.walBlocks, id)
		}
	}

	for id, completeBlock := range i.completeBlocks {
		if completeBlock.BlockMeta().EndTime.Before(cutoff) {
			flushedTime := completeBlock.FlushedTime()
			if !flushedTime.IsZero() { // Only delete if flushed
				level.Info(i.logger).Log("msg", "deleting complete block", "block", id.String())
				err := i.wal.LocalBackend().ClearBlock(id, i.tenantID)
				if err != nil {
					return err
				}
				delete(i.completeBlocks, id)
			}
		}
	}

	return nil
}
