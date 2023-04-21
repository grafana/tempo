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
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/pkg/errors"
)

type Processor struct {
	tenant  string
	Cfg     Config
	wal     *wal.WAL
	closeCh chan struct{}

	blocksMtx   sync.Mutex
	headBlock   common.WALBlock
	walBlocks   []common.WALBlock
	lastCutTime time.Time

	liveTracesMtx sync.Mutex
	liveTraces    *liveTraces
}

var _ gen.Processor = (*Processor)(nil)

func New(cfg Config, tenant string) (*Processor, error) {
	if cfg.WAL == nil {
		return nil, errors.New("wal config cannot be nil")
	}

	// Copy the wal config. This is because it contains runtime properties
	// that are set after creation like CompletedFilePath that break the
	// automatic configuration drift detection higher up in the metrics
	// generator module.
	walCfg := *cfg.WAL
	wal, err := wal.New(&walCfg)
	if err != nil {
		return nil, errors.Wrap(err, "creating local blocks processor wal")
	}

	p := &Processor{
		Cfg:        cfg,
		tenant:     tenant,
		wal:        wal,
		liveTraces: NewLiveTraces(),
		closeCh:    make(chan struct{}),
	}

	go p.loop()

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

func (p *Processor) loop() {
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

	err := p.headBlock.Append(id, b, 0, 0)
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

	if time.Since(p.lastCutTime) < p.Cfg.MaxBlockDuration || p.headBlock.DataLength() < p.Cfg.MaxBlockBytes {
		return nil
	}

	// Final flush
	err := p.headBlock.Flush()
	if err != nil {
		return fmt.Errorf("failed to flush head block: %w", err)
	}

	p.walBlocks = append(p.walBlocks, p.headBlock)

	err = p.resetHeadBlock()
	if err != nil {
		return fmt.Errorf("failed to resetHeadBlock: %w", err)
	}

	return nil
}
