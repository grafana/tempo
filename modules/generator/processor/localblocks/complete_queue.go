package localblocks

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/flushqueues"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricCompleteQueueLength = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "complete_queue_length",
		Help:      "Number of wal blocks waiting for completion",
	})

	completeQueue          = flushqueues.NewPriorityQueue(metricCompleteQueueLength)
	startCompleteQueueOnce = &sync.Once{}
)

type completeOp struct {
	key      string
	p        *Processor
	blockID  uuid.UUID
	ctx      context.Context
	at       time.Time
	attempts int
	bo       time.Duration
}

func (f *completeOp) Key() string {
	return f.key
}

func (f *completeOp) Priority() int64 { return -f.at.Unix() }

func (f *completeOp) backoff() time.Duration {
	f.bo *= 2
	if f.bo > maxBackoff {
		f.bo = maxBackoff
	}

	return f.bo
}

func enqueueCompleteOp(ctx context.Context, p *Processor, blockID uuid.UUID) error {
	_, err := completeQueue.Enqueue(&completeOp{
		// Core fields
		key:     uuid.NewString(), // Instead of relying on queue idempotency we handle duplicates within the processor
		ctx:     ctx,
		p:       p,
		blockID: blockID,

		// Initial priority and backoff
		at: time.Now(),
		bo: 30 * time.Second,
	})
	return err
}

// startCompleteQueue consume routines once.
func startCompleteQueue(concurrency uint) {
	startCompleteQueueOnce.Do(func() {
		for i := uint(0); i < concurrency; i++ {
			go completeLoop()
		}
	})
}

func completeLoop() {
	for {
		op := completeQueue.Dequeue().(*completeOp)

		// The context is used to detect processors that have been shutdown.
		if err := op.ctx.Err(); err != nil {
			if !errors.Is(err, context.Canceled) {
				level.Error(log.Logger).Log("msg", "abandoning complete queue entry for ctx error", "err", err, "blockid", op.blockID)
			}
			continue
		}

		err := op.p.completeBlock(op.blockID)
		if err != nil {
			_ = level.Info(log.Logger).Log("msg", "re-queueing block for flushing", "block", op.blockID, "attempts", op.attempts, "err", err)

			delay := op.backoff()
			op.at = time.Now().Add(delay)

			go func() {
				time.Sleep(delay)
				if _, err := completeQueue.Enqueue(op); err != nil {
					_ = level.Error(log.Logger).Log("msg", "failed to requeue block for flushing", "err", err)
				}
			}()
		}
	}
}
