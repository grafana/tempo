package localblocks

import (
	"context"
	"errors"
	"fmt"
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
	completeQueue          *flushqueues.PriorityQueue
	startCompleteQueueOnce = &sync.Once{}

	metricCompleteQueueLength = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "complete_queue_length",
		Help:      "asdf",
	})
)

type completeOp struct {
	at       time.Time
	attempts int
	p        *Processor
	tenant   string
	blockID  uuid.UUID
	ctx      context.Context
	bo       time.Duration
}

func (f *completeOp) Key() string {
	// Address of processor is used in the key to ensure that
	// different instantiations of processor for the same tenant
	// are considered different.
	key := fmt.Sprintf("%p", f.p) + "-" + f.tenant + "-" + f.blockID.String()
	return key
}

func (f *completeOp) Priority() int64 { return -f.at.Unix() }

func (f *completeOp) backoff() time.Duration {
	f.bo *= 2
	if f.bo > maxBackoff {
		f.bo = maxBackoff
	}

	return f.bo
}

func startCompleteQueue(concurrency uint) {
	startCompleteQueueOnce.Do(func() {
		completeQueue = flushqueues.NewPriorityQueue(metricCompleteQueueLength)

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
