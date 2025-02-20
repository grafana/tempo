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

const completeQueueMaxBackoff = 5 * time.Minute

var (
	metricCompleteQueueLength = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "complete_queue_length",
		Help:      "Number of wal blocks waiting for completion",
	})

	// completeQueue is a shared priority queue that handles completing WAL blocks.
	// The queue provides backoff and retry capabilities for failed operations
	// The queue is shared across all processor instances to control concurrency.
	completeQueue     *flushqueues.PriorityQueue
	completeQueueMtx  = sync.Mutex{}
	completeQueueRefs = 0
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
	if f.bo > completeQueueMaxBackoff {
		f.bo = completeQueueMaxBackoff
	}

	return f.bo
}

var _ flushqueues.Op = (*completeOp)(nil)

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

// startCompleteQueue increments reference count of the queue and starts it
// on the first call with the given concurrency.
func startCompleteQueue(concurrency uint) {
	completeQueueMtx.Lock()
	defer completeQueueMtx.Unlock()

	completeQueueRefs++
	if completeQueueRefs == 1 {
		completeQueue = flushqueues.NewPriorityQueue(metricCompleteQueueLength)
		for i := uint(0); i < concurrency; i++ {
			go completeLoop(completeQueue)
		}
	}
}

// stopCompleteQueue decrements reference count of the queue and closes it
// when it reaches zero.
func stopCompleteQueue() {
	completeQueueMtx.Lock()
	defer completeQueueMtx.Unlock()

	completeQueueRefs--
	if completeQueueRefs == 0 {
		completeQueue.DiscardAndClose()
	}
}

func completeLoop(q *flushqueues.PriorityQueue) {
	for {
		o := q.Dequeue()
		if o == nil {
			// Queue is closed.
			return
		}

		op := o.(*completeOp)
		op.attempts++

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
				if _, err := q.Enqueue(op); err != nil {
					_ = level.Error(log.Logger).Log("msg", "failed to requeue block for flushing", "err", err)
				}
			}()
		}
	}
}
