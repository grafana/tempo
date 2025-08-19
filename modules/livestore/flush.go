package livestore

import (
	"fmt"
	"time"

	"github.com/go-kit/log/level"
	"github.com/google/uuid"
)

const (
	maxBackoff       = 120 * time.Second
	maxFlushAttempts = 10
)

type completeOp struct {
	tenantID string
	blockID  uuid.UUID

	at       time.Time
	attempts int
	bo       time.Duration
}

func (o *completeOp) Key() string { return o.tenantID + "/" + o.blockID.String() }

func (o *completeOp) Priority() int64 { return -o.at.Unix() }

func (o *completeOp) backoff() time.Duration {
	o.bo *= 2
	if o.bo > maxBackoff {
		o.bo = maxBackoff
	}

	return o.bo
}

func (s *LiveStore) startAllBackgroundProcesses() {
	close(s.startupComplete)
}

func (s *LiveStore) stopAllBackgroundProcesses() {
	s.cancel()              // this will cause the per tenant background processes to complete
	s.completeQueues.Stop() // this will cause the global complete loop by preventing additional enqueues
	s.wg.Wait()
}

func (s *LiveStore) runInBackground(fn func()) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		select {
		case <-s.startupComplete:
		case <-s.ctx.Done():
			return
		}

		fn()
	}()
}

func (s *LiveStore) globalCompleteLoop(idx int) {
	for {
		o := s.completeQueues.Dequeue(idx)
		if o == nil {
			return // queue is closed
		}
		op := o.(*completeOp)
		op.attempts++

		if op.attempts > maxFlushAttempts {
			level.Error(s.logger).Log("msg", "failed to complete operation", "tenant", op.tenantID, "block", op.blockID, "attempts", op.attempts)
			observeFailedOp(op)
			continue
		}

		start := time.Now()
		inst, err := s.getOrCreateInstance(op.tenantID)
		if err != nil {
			level.Error(s.logger).Log("msg", "failed to retrieve instance for completion", "tenant", op.tenantID, "err", err)
			observeFailedOp(op)
			return
		}

		err = inst.completeBlock(s.ctx, op.blockID)
		duration := time.Since(start)
		metricCompletionDuration.Observe(duration.Seconds())

		if err != nil {
			level.Error(s.logger).Log("msg", "failed to complete block", "tenant", op.tenantID, "block", op.blockID, "err", err)
			observeFailedOp(op)

			delay := op.backoff()
			op.at = time.Now().Add(delay)

			metricCompletionRetries.Inc()

			go func() {
				time.Sleep(delay)

				if err := s.enqueueOp(op); err != nil {
					_ = level.Error(s.logger).Log("msg", "failed to requeue block for flushing", "tenant", op.tenantID, "block", op.blockID, "err", err)
				}
			}()
		} else {
			metricBlocksCompleted.Inc()
		}
	}
}

func (s *LiveStore) perTenantCutToWalLoop(instance *instance) {
	// ticker
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cutOneInstanceToWal(instance, false)
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *LiveStore) perTenantCleanupLoop(inst *instance) {
	// ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// dump any blocks that have been flushed for a while
			err := inst.deleteOldBlocks()
			if err != nil {
				level.Error(s.logger).Log("msg", "failed to delete old blocks", "err", err)
			}
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *LiveStore) enqueueCompleteOp(tenantID string, blockID uuid.UUID) error {
	return s.enqueueOp(&completeOp{
		tenantID: tenantID,
		blockID:  blockID,
		// Initial priority and backoff
		at: time.Now(),
		bo: 30 * time.Second,
	})
}

func (s *LiveStore) enqueueOp(op *completeOp) error {
	if s.completeQueues.IsStopped() {
		return fmt.Errorf("complete queues are stopped, cannot enqueue operation for block %s", op.blockID.String())
	}

	return s.completeQueues.Enqueue(op)
}

func observeFailedOp(op *completeOp) {
	metricFailedCompletions.Inc()
	if op.attempts > 1 {
		metricCompletionFailedRetries.Inc()
	}
}
