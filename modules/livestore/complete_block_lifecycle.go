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
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricBlocksFlushed = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo_live_store",
		Name:      "local_blocks_flushed_total",
		Help:      "The total number of local complete blocks flushed",
	})
	metricFailedFlushes = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo_live_store",
		Name:      "local_failed_flushes_total",
		Help:      "The total number of failed local complete block flushes",
	})
	metricFlushRetries = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo_live_store",
		Name:      "local_flush_retries_total",
		Help:      "The total number of retries after a failed local complete block flush",
	})
	metricFlushFailedRetries = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempo_live_store",
		Name:      "local_flush_failed_retries_total",
		Help:      "The total number of failed retries after a failed local complete block flush",
	})
	metricFlushDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace:                       "tempo_live_store",
		Name:                            "local_flush_duration_seconds",
		Help:                            "Records the amount of time to flush a local complete block.",
		Buckets:                         prometheus.ExponentialBuckets(1, 2, 10),
		NativeHistogramBucketFactor:     1.1,
		NativeHistogramMaxBucketNumber:  100,
		NativeHistogramMinResetDuration: 1 * time.Hour,
	})
	metricFlushSize = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace:                       "tempo_live_store",
		Name:                            "local_flush_size_bytes",
		Help:                            "Size in bytes of local complete blocks flushed.",
		Buckets:                         prometheus.ExponentialBuckets(1024*1024, 2, 10),
		NativeHistogramBucketFactor:     1.1,
		NativeHistogramMaxBucketNumber:  100,
		NativeHistogramMinResetDuration: 1 * time.Hour,
	})
)

// completeBlockFlusher is the minimal write capability needed by the
// single-binary complete-block lifecycle.
type completeBlockFlusher interface {
	WriteBlock(ctx context.Context, block tempodb.WriteableBlock) error
}

// completeBlockLifecycle owns mode-specific handling for locally completed
// blocks. Kafka mode keeps the existing no-op behavior, while local mode
// flushes completed blocks in the background.
type completeBlockLifecycle interface {
	start(ctx context.Context)
	stop()
	onCompletedBlock(ctx context.Context, tenantID string, block *LocalBlock) error
	onReloadedBlock(ctx context.Context, tenantID string, block *LocalBlock) error
	shouldDeleteCompleteBlock(block *LocalBlock, cutoff time.Time) bool
}

func newCompleteBlockLifecycle(cfg Config, flusher completeBlockFlusher, logger log.Logger) (completeBlockLifecycle, error) {
	if cfg.ConsumeFromKafka {
		return kafkaCompleteBlockLifecycle{}, nil
	}

	if flusher == nil {
		return nil, fmt.Errorf("complete block flusher is required when kafka consumption is disabled")
	}

	return &localCompleteBlockLifecycle{
		flusher:            flusher,
		logger:             logger,
		flushConcurrency:   cfg.CompleteBlockConcurrency,
		retryDelay:         cfg.initialBackoff,
		completeBlockQueue: flushqueues.New[*localCompleteBlockOp](nil),
	}, nil
}

// The lifecycle for Kafka mode is noop

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

// The local implementation is used in the single-binary / monolithic mode for flushing blocks to the backend storage
type localCompleteBlockLifecycle struct {
	flusher completeBlockFlusher
	logger  log.Logger

	flushConcurrency   int
	retryDelay         time.Duration
	completeBlockQueue *flushqueues.ExclusiveQueues[*localCompleteBlockOp]
	wg                 sync.WaitGroup
	ctx                context.Context
	cancel             context.CancelFunc
}

type localCompleteBlockOp struct {
	tenantID string
	block    *LocalBlock
	at       time.Time
	attempts int
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
	for range l.flushConcurrency {
		l.wg.Go(func() {
			l.runFlushLoop()
		})
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

func (*localCompleteBlockLifecycle) shouldDeleteCompleteBlock(block *LocalBlock, cutoff time.Time) bool {
	if block == nil || block.FlushedTime().IsZero() {
		return false
	}

	return shouldDeleteCompleteBlockByAge(block, cutoff)
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

// Main loop. It dequeues items from the queue and tries to flush them to the backend storage.
// Failed ones are requeued after a short delay.
func (l *localCompleteBlockLifecycle) runFlushLoop() {
	for {
		op := l.completeBlockQueue.Dequeue()
		if op == nil {
			return
		}
		op.attempts++

		start := time.Now()
		err := l.flusher.WriteBlock(l.ctx, op.block)
		metricFlushDuration.Observe(time.Since(start).Seconds())
		metricFlushSize.Observe(float64(op.block.BlockMeta().Size_))
		if err != nil {
			l.observeFailedFlush(op, err)
			l.requeueAfter(op, l.retryDelay)
			continue
		}

		metricBlocksFlushed.Inc()
		l.completeBlockQueue.Clear(op)
	}
}

func (l *localCompleteBlockLifecycle) observeFailedFlush(op *localCompleteBlockOp, err error) {
	level.Error(l.logger).Log("msg", "failed to flush complete block", "tenant", op.tenantID, "block", op.block.BlockMeta().BlockID.String(), "attempts", op.attempts, "err", err)
	metricFailedFlushes.Inc()
	if op.attempts > 1 {
		metricFlushFailedRetries.Inc()
	}
}

// Failed blocks are requeued after a short delay
func (l *localCompleteBlockLifecycle) requeueAfter(op *localCompleteBlockOp, delay time.Duration) {
	op.at = time.Now().Add(delay)

	time.AfterFunc(delay, func() {
		if l.ctx.Err() != nil {
			return
		}

		metricFlushRetries.Inc()
		if err := l.completeBlockQueue.Requeue(op); err != nil {
			level.Error(l.logger).Log("msg", "failed to requeue complete block flush", "tenant", op.tenantID, "block", op.block.BlockMeta().BlockID.String(), "attempts", op.attempts, "err", err)
		}
	})
}

func shouldDeleteCompleteBlockByAge(block *LocalBlock, cutoff time.Time) bool {
	if block == nil {
		return false
	}

	return block.BlockMeta().EndTime.Before(cutoff)
}
