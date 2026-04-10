package livestore

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/pkg/flushqueues"
	"github.com/grafana/tempo/tempodb"
	"github.com/prometheus/client_golang/prometheus"
)

// completeBlockFlusher is the minimal write capability needed by the
// single-binary complete-block lifecycle.
type completeBlockFlusher interface {
	WriteBlock(ctx context.Context, block tempodb.WriteableBlock) error
}

// completeBlockLifecycle owns mode-specific handling for locally completed
// blocks. The initial implementation preserves the current Kafka behaviour;
// single-binary-specific background flushing will be added in a follow-up.
type completeBlockLifecycle interface {
	start(ctx context.Context)
	stop()
	onCompletedBlock(ctx context.Context, tenantID string, block *LocalBlock) error
	onReloadedBlock(ctx context.Context, tenantID string, block *LocalBlock) error
	shouldDeleteCompleteBlock(block *LocalBlock, cutoff time.Time) bool
}

func newCompleteBlockLifecycle(cfg Config, flusher completeBlockFlusher, logger log.Logger, reg prometheus.Registerer) (completeBlockLifecycle, error) {
	if cfg.ConsumeFromKafka {
		return kafkaCompleteBlockLifecycle{}, nil
	}

	if flusher == nil {
		return nil, fmt.Errorf("complete block flusher is required when kafka consumption is disabled")
	}

	return &localCompleteBlockLifecycle{
		flusher:            flusher,
		logger:             logger,
		reg:                reg,
		flushConcurrency:   cfg.CompleteBlockConcurrency,
		completeBlockQueue: flushqueues.New[*localCompleteBlockOp](cfg.CompleteBlockConcurrency, nil),
	}, nil
}

type kafkaCompleteBlockLifecycle struct{}

func (kafkaCompleteBlockLifecycle) start(context.Context) {}

func (kafkaCompleteBlockLifecycle) stop() {}

func (kafkaCompleteBlockLifecycle) onCompletedBlock(context.Context, string, *LocalBlock) error {
	return nil
}

func (kafkaCompleteBlockLifecycle) onReloadedBlock(context.Context, string, *LocalBlock) error {
	return nil
}

func (kafkaCompleteBlockLifecycle) shouldDeleteCompleteBlock(block *LocalBlock, cutoff time.Time) bool {
	return shouldDeleteCompleteBlockByAge(block, cutoff)
}

type localCompleteBlockLifecycle struct {
	flusher completeBlockFlusher
	logger  log.Logger
	reg     prometheus.Registerer

	flushConcurrency   int
	completeBlockQueue *flushqueues.ExclusiveQueues[*localCompleteBlockOp]
	wg                 sync.WaitGroup
	ctx                context.Context
	cancel             context.CancelFunc
}

type localCompleteBlockOp struct {
	tenantID string
	block    *LocalBlock
	at       time.Time
}

var _ flushqueues.Op = (*localCompleteBlockOp)(nil)

func (o *localCompleteBlockOp) Key() string {
	return o.tenantID + "/" + o.block.BlockMeta().BlockID.String()
}

func (o *localCompleteBlockOp) Priority() int64 {
	return -o.at.UnixNano()
}

func (l *localCompleteBlockLifecycle) start(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}

	l.ctx, l.cancel = context.WithCancel(ctx)

	for i := range l.flushConcurrency {
		idx := i
		l.wg.Add(1)
		go func() {
			defer l.wg.Done()
			l.runFlushLoop(idx)
		}()
	}
}

func (l *localCompleteBlockLifecycle) stop() {
	if l.cancel != nil {
		l.cancel()
	}
	l.completeBlockQueue.Stop()
	l.wg.Wait()
}

func (l *localCompleteBlockLifecycle) onCompletedBlock(_ context.Context, tenantID string, block *LocalBlock) error {
	return l.enqueueBlock(tenantID, block)
}

func (l *localCompleteBlockLifecycle) onReloadedBlock(_ context.Context, tenantID string, block *LocalBlock) error {
	if block == nil || !block.FlushedTime().IsZero() {
		return nil
	}

	return l.enqueueBlock(tenantID, block)
}

func (l *localCompleteBlockLifecycle) enqueueBlock(tenantID string, block *LocalBlock) error {
	if block == nil {
		return nil
	}

	op := &localCompleteBlockOp{
		tenantID: tenantID,
		block:    block,
		at:       time.Now(),
	}

	if err := l.completeBlockQueue.Enqueue(op); err != nil {
		return fmt.Errorf("enqueue complete block flush op: %w", err)
	}

	return nil
}

func (l *localCompleteBlockLifecycle) runFlushLoop(idx int) {
	for {
		op := l.completeBlockQueue.Dequeue(idx)
		if op == nil {
			return
		}

		if err := l.flusher.WriteBlock(l.ctx, op.block); err != nil {
			level.Error(l.logger).Log("msg", "failed to flush complete block", "tenant", op.tenantID, "block", op.block.BlockMeta().BlockID.String(), "err", err)
			l.completeBlockQueue.Clear(op)
			continue
		}

		l.completeBlockQueue.Clear(op)
	}
}

func (*localCompleteBlockLifecycle) shouldDeleteCompleteBlock(block *LocalBlock, cutoff time.Time) bool {
	if block == nil {
		return false
	}
	if block.FlushedTime().IsZero() {
		return false
	}

	return shouldDeleteCompleteBlockByAge(block, cutoff)
}

func shouldDeleteCompleteBlockByAge(block *LocalBlock, cutoff time.Time) bool {
	if block == nil {
		return false
	}

	return block.BlockMeta().EndTime.Before(cutoff)
}
