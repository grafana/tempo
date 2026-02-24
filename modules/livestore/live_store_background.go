package livestore

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/tempo/modules/ingester"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
)

const (
	defaultInitialBackoff = 30 * time.Second
	defaultMaxBackoff     = 120 * time.Second
	maxFlushAttempts      = 10
)

type completeOp struct {
	tenantID string
	blockID  uuid.UUID

	at         time.Time
	attempts   int
	bo         time.Duration
	maxBackoff time.Duration
}

func (o *completeOp) Key() string { return o.tenantID + "/" + o.blockID.String() }

func (o *completeOp) Priority() int64 { return -o.at.Unix() }

func (o *completeOp) backoff() time.Duration {
	o.bo *= 2
	if o.bo > o.maxBackoff {
		o.bo = o.maxBackoff
	}

	return o.bo
}

func (s *LiveStore) startAllBackgroundProcesses() {
	if s.cfg.holdAllBackgroundProcesses {
		level.Warn(s.logger).Log("msg", "live store has been started with all background processes suspended! this is meant for testing only")
		return
	}

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
		op := s.completeQueues.Dequeue(idx)
		if op == nil {
			return // queue is closed
		}
		op.attempts++

		if op.attempts > maxFlushAttempts {
			level.Error(s.logger).Log("msg", "failed to complete operation", "tenant", op.tenantID, "attempts", op.attempts)
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

		err = inst.createBlockFromPending(s.ctx)
		duration := time.Since(start)
		metricCompletionDuration.Observe(duration.Seconds())

		if err != nil {
			level.Error(s.logger).Log("msg", "failed to create block from pending", "tenant", op.tenantID, "err", err)
			observeFailedOp(op)

			delay := op.backoff()
			op.at = time.Now().Add(delay)

			metricCompletionRetries.Inc()

			go func() {
				time.Sleep(delay)

				if err := s.requeueOp(op); err != nil {
					_ = level.Error(s.logger).Log("msg", "failed to requeue block for flushing", "tenant", op.tenantID, "err", err)
				}
			}()
		} else {
			metricBlocksCompleted.Inc()
			s.completeQueues.Clear(op)
		}
	}
}

func (s *LiveStore) perTenantCutLoop(instance *instance) {
	ticker := time.NewTicker(s.cfg.InstanceFlushPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cutOneInstance(instance, false)
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *LiveStore) perTenantCleanupLoop(inst *instance) {
	// ticker
	ticker := time.NewTicker(s.cfg.InstanceCleanupPeriod)
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

func (s *LiveStore) enqueueCompleteOp(tenantID string, blockID uuid.UUID, jitter bool) error {
	op := &completeOp{
		tenantID: tenantID,
		blockID:  blockID,
		// Initial priority and backoff
		at:         time.Now(),
		bo:         s.cfg.initialBackoff,
		maxBackoff: s.cfg.maxBackoff,
	}

	if jitter {
		return s.enqueueOpWithJitter(op)
	}

	return s.enqueueOp(op)
}

func (s *LiveStore) enqueueOpWithJitter(op *completeOp) error {
	delay := time.Duration(rand.Int64N(10_000) * int64(time.Millisecond)) //gosec:disable G404 — It doesn't require strong randomness
	go func() {
		time.Sleep(delay)
		if err := s.enqueueOp(op); err != nil {
			level.Error(s.logger).Log("msg", "failed to enqueue block", "tenant", op.tenantID, "block", op.blockID, "err", err)
		}
	}()
	return nil
}

func (s *LiveStore) enqueueOp(op *completeOp) error {
	if s.completeQueues.IsStopped() {
		return fmt.Errorf("complete queues are stopped, cannot enqueue operation for block %s", op.blockID.String())
	}

	level.Debug(s.logger).Log("msg", "enqueueing complete op", "tenant", op.tenantID, "block", op.blockID, "attempts", op.attempts)
	return s.completeQueues.Enqueue(op)
}

func (s *LiveStore) requeueOp(op *completeOp) error {
	if s.completeQueues.IsStopped() {
		return fmt.Errorf("complete queues are stopped, cannot requeue operation for block %s", op.blockID.String())
	}

	level.Debug(s.logger).Log("msg", "requeueing complete op", "tenant", op.tenantID, "block", op.blockID, "attempts", op.attempts)
	return s.completeQueues.Requeue(op)
}

func observeFailedOp(op *completeOp) {
	metricFailedCompletions.Inc()
	if op.attempts > 1 {
		metricCompletionFailedRetries.Inc()
	}
}

func (s *LiveStore) reloadBlocks() error {
	// ------------------------------------
	// Complete blocks
	// ------------------------------------
	var (
		ctx = s.ctx
		l   = s.localBackend
		r   = backend.NewReader(l)
	)

	tenants, err := r.Tenants(ctx)
	if err != nil {
		return fmt.Errorf("failed to get local tenants: %w", err)
	}

	for _, tenant := range tenants {
		ids, _, err := r.Blocks(ctx, tenant)
		if err != nil {
			return fmt.Errorf("failed to get local blocks for tenant %s: %w", tenant, err)
		}
		level.Info(s.logger).Log("msg", "reloading complete blocks", "tenant", tenant, "count", len(ids))

		for _, id := range ids {
			level.Info(s.logger).Log("msg", "reloading complete block", "block", id.String())
			meta, err := r.BlockMeta(ctx, id, tenant)

			// delete blocks that do not have a meta or a corrupt meta
			var clearBlock bool
			if err != nil {
				var vv *json.SyntaxError
				if errors.Is(err, backend.ErrDoesNotExist) || errors.As(err, &vv) {
					clearBlock = true
				}
			}

			if clearBlock {
				level.Info(s.logger).Log("msg", "clearing block", "block", id.String(), "err", err)
				// Partially written block, delete and continue
				err = l.ClearBlock(id, tenant)
				if err != nil {
					level.Error(s.logger).Log("msg", "failed to clear partially written block during replay", "err", err)
				}
				continue
			}

			if err != nil {
				return fmt.Errorf("failed to get block meta for block %s in tenant %s: %w", id.String(), tenant, err)
			}

			blk, err := encoding.OpenBlock(meta, r)
			if err != nil {
				return fmt.Errorf("failed to open block %s in tenant %s: %w", id.String(), tenant, err)
			}

			err = blk.Validate(ctx)
			if err != nil && !errors.Is(err, util.ErrUnsupported) {
				level.Error(s.logger).Log("msg", "local block failed validation, dropping", "block", id.String(), "error", err)

				err = l.ClearBlock(id, tenant)
				if err != nil {
					level.Error(s.logger).Log("msg", "failed to clear invalid block during replay", "err", err)
				}

				continue
			}

			level.Info(s.logger).Log("msg", "reloaded complete block", "block", id.String())

			lb := ingester.NewLocalBlock(ctx, blk, l)

			inst, err := s.getOrCreateInstance(tenant)
			if err != nil {
				return fmt.Errorf("failed to get or create instance for tenant %s during complete block reload: %w", tenant, err)
			}

			inst.blocksMtx.Lock()
			inst.completeBlocks[id] = lb
			inst.blocksMtx.Unlock()
		}
	}

	return nil
}
