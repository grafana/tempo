package localblocks

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	gen "github.com/grafana/tempo/modules/generator/processor"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/pkg/errors"
)

type Processor struct {
	tenant  string
	Cfg     Config
	wal     *wal.WAL
	closeCh chan struct{}

	blocksMtx      sync.Mutex
	headBlock      common.WALBlock
	walBlocks      map[uuid.UUID]common.WALBlock
	completeBlocks map[uuid.UUID]common.BackendBlock
	lastCutTime    time.Time

	liveTracesMtx sync.Mutex
	liveTraces    *liveTraces
}

var _ gen.Processor = (*Processor)(nil)

func New(cfg Config, tenant string, wal *wal.WAL) (*Processor, error) {

	if wal == nil {
		return nil, errors.New("local blocks processor requires traces wal")
	}

	p := &Processor{
		Cfg:            cfg,
		tenant:         tenant,
		wal:            wal,
		walBlocks:      map[uuid.UUID]common.WALBlock{},
		completeBlocks: map[uuid.UUID]common.BackendBlock{},
		liveTraces:     NewLiveTraces(),
		closeCh:        make(chan struct{}),
	}

	err := p.reloadBlocks()
	if err != nil {
		return nil, errors.Wrap(err, "replaying blocks")
	}

	go p.flushLoop()
	go p.deleteLoop()
	go p.completeLoop()

	return p, nil
}

func (*Processor) Name() string {
	return "LocalBlocksProcessor"
}

func (p *Processor) PushSpans(ctx context.Context, req *tempopb.PushSpansRequest) {

	p.liveTracesMtx.Lock()
	defer p.liveTracesMtx.Unlock()

	for _, batch := range req.Batches {
		p.liveTraces.Push(batch)
	}
}

func (p *Processor) Shutdown(ctx context.Context) {
	close(p.closeCh)
}

func (p *Processor) flushLoop() {
	flushTicker := time.NewTicker(p.Cfg.FlushCheckPeriod)
	defer flushTicker.Stop()

	for {
		select {
		case <-flushTicker.C:
			err := p.cutIdleTraces(false)
			if err != nil {
				level.Error(log.WithUserID(p.tenant, log.Logger)).Log("msg", "local blocks processor failed to cut idle traces", "err", err)
			}

			err = p.cutBlocks()
			if err != nil {
				level.Error(log.WithUserID(p.tenant, log.Logger)).Log("msg", "local blocks processor failed to cut idle traces", "err", err)
			}

		case <-p.closeCh:
			// Immediately cut all traces from memory
			err := p.cutIdleTraces(true)
			if err != nil {
				level.Error(log.WithUserID(p.tenant, log.Logger)).Log("msg", "local blocks processor failed to cut idle traces", "err", err)
			}
			return
		}
	}
}

func (p *Processor) deleteLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := p.deleteOldBlocks()
			if err != nil {
				level.Error(log.WithUserID(p.tenant, log.Logger)).Log("msg", "local blocks processor failed to delete old blocks", "err", err)
			}

		case <-p.closeCh:
			return
		}
	}
}

func (p *Processor) completeLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := p.completeBlock()
			if err != nil {
				level.Error(log.WithUserID(p.tenant, log.Logger)).Log("msg", "local blocks processor failed to complete a block", "err", err)
			}

		case <-p.closeCh:
			return
		}
	}
}

func (p *Processor) completeBlock() error {
	p.blocksMtx.Lock()
	defer p.blocksMtx.Unlock()

	var (
		ctx    = context.Background()
		enc    = encoding.DefaultEncoding()
		reader = backend.NewReader(p.wal.LocalBackend())
		writer = backend.NewWriter(p.wal.LocalBackend())
		cfg    = p.Cfg.Block
	)

	for id, b := range p.walBlocks {

		iter, err := b.Iterator()
		if err != nil {
			return err
		}
		defer iter.Close()

		newMeta, err := enc.CreateBlock(ctx, cfg, b.BlockMeta(), iter, reader, writer)
		if err != nil {
			return err
		}

		newBlock, err := enc.OpenBlock(newMeta, reader)
		if err != nil {
			return err
		}

		p.completeBlocks[newMeta.BlockID] = newBlock

		err = b.Clear()
		if err != nil {
			return err
		}
		delete(p.walBlocks, id)

		// Only do 1 block per call
		return nil
	}

	return nil
}

func (p *Processor) deleteOldBlocks() (err error) {
	p.blocksMtx.Lock()
	defer p.blocksMtx.Unlock()

	before := time.Now().Add(-p.Cfg.CompleteBlockTimeout)

	for id, b := range p.walBlocks {
		if b.BlockMeta().EndTime.Before(before) {
			err = b.Clear()
			if err != nil {
				return err
			}
			delete(p.walBlocks, id)
		}
	}

	for id, b := range p.completeBlocks {
		if b.BlockMeta().EndTime.Before(before) {
			err = p.wal.LocalBackend().ClearBlock(id, p.tenant)
			if err != nil {
				return err
			}
			delete(p.walBlocks, id)
		}
	}

	return
}

func (p *Processor) cutIdleTraces(immediate bool) error {
	p.liveTracesMtx.Lock()
	defer p.liveTracesMtx.Unlock()

	since := time.Now().Add(-p.Cfg.MaxTraceIdle)
	if immediate {
		since = time.Time{}
	}

	tracesToCut := p.liveTraces.CutIdle(since)

	if len(tracesToCut) == 0 {
		return nil
	}

	segmentDecoder := model.MustNewSegmentDecoder(model.CurrentEncoding)

	// Sort by ID
	sort.Slice(tracesToCut, func(i, j int) bool {
		return bytes.Compare(tracesToCut[i].id, tracesToCut[j].id) == -1
	})

	for _, t := range tracesToCut {

		// TODO - This is dumb because the wal block will immediately
		// unmarshal the bytes, fix this.

		buf, err := segmentDecoder.PrepareForWrite(&tempopb.Trace{
			Batches: t.Batches,
		}, 0, 0)
		if err != nil {
			return err
		}

		out, err := segmentDecoder.ToObject([][]byte{buf})
		if err != nil {
			return err
		}

		err = p.writeHeadBlock(t.id, out)
		if err != nil {
			return err
		}

	}

	p.blocksMtx.Lock()
	defer p.blocksMtx.Unlock()
	return p.headBlock.Flush()
}

func (p *Processor) writeHeadBlock(id common.ID, b []byte) error {
	p.blocksMtx.Lock()
	defer p.blocksMtx.Unlock()

	if p.headBlock == nil {
		err := p.resetHeadBlock()
		if err != nil {
			return err
		}
	}

	now := uint32(time.Now().Unix())

	err := p.headBlock.Append(id, b, now, now)
	if err != nil {
		return err
	}

	return nil
}

func (p *Processor) resetHeadBlock() error {
	block, err := p.wal.NewBlock(uuid.New(), p.tenant, model.CurrentEncoding)
	if err != nil {
		return err
	}
	p.headBlock = block
	p.lastCutTime = time.Now()
	return nil
}

func (p *Processor) cutBlocks() error {
	p.blocksMtx.Lock()
	defer p.blocksMtx.Unlock()

	if p.headBlock == nil {
		return nil
	}

	if time.Since(p.lastCutTime) < p.Cfg.MaxBlockDuration && p.headBlock.DataLength() < p.Cfg.MaxBlockBytes {
		return nil
	}

	// Final flush
	err := p.headBlock.Flush()
	if err != nil {
		return fmt.Errorf("failed to flush head block: %w", err)
	}

	p.walBlocks[p.headBlock.BlockMeta().BlockID] = p.headBlock

	err = p.resetHeadBlock()
	if err != nil {
		return fmt.Errorf("failed to resetHeadBlock: %w", err)
	}

	return nil
}

func (p *Processor) reloadBlocks() error {
	var (
		ctx = context.Background()
		t   = p.tenant
		l   = p.wal.LocalBackend()
		r   = backend.NewReader(l)
	)

	// ------------------------------------
	// wal blocks
	// ------------------------------------
	walBlocks, err := p.wal.RescanBlocks(0, log.Logger)
	if err != nil {
		return err
	}
	for _, blk := range walBlocks {
		meta := blk.BlockMeta()
		if meta.TenantID == p.tenant {
			p.walBlocks[blk.BlockMeta().BlockID] = blk
		}
	}

	// ------------------------------------
	// Complete blocks
	// ------------------------------------

	// This is a quirk, we shouldn't try to list blocks until after we've made
	// sure the tenant folder exists.
	tenants, err := r.Tenants(ctx)
	if err != nil {
		return err
	}
	if len(tenants) == 0 {
		return nil
	}

	ids, err := r.Blocks(ctx, p.tenant)
	if err != nil {
		return err
	}

	for _, id := range ids {
		meta, err := r.BlockMeta(ctx, id, t)

		if err == backend.ErrDoesNotExist {
			// Partially written block, delete and continue
			err = l.ClearBlock(id, t)
			if err != nil {
				level.Error(log.WithUserID(p.tenant, log.Logger)).Log("msg", "local blocks processor failed to clear partially written block during replay", "err", err)
			}
			continue
		}

		if err != nil {
			return err
		}

		blk, err := encoding.OpenBlock(meta, r)
		if err != nil {
			return err
		}

		p.completeBlocks[id] = blk
	}

	return nil
}
