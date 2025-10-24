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
	maxBackoff       = 120 * time.Second
	initialBackoff   = 30 * time.Second
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
			s.completeQueues.Clear(op)
		}
	}
}

func (s *LiveStore) perTenantCutToWalLoop(instance *instance) {
	// ticker
	ticker := time.NewTicker(s.cfg.InstanceFlushPeriod)
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
		at: time.Now(),
		bo: initialBackoff,
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

func observeFailedOp(op *completeOp) {
	metricFailedCompletions.Inc()
	if op.attempts > 1 {
		metricCompletionFailedRetries.Inc()
	}
}

func (s *LiveStore) reloadBlocks() error {
	// ------------------------------------
	// wal blocks
	// ------------------------------------
	level.Info(s.logger).Log("msg", "reloading wal blocks")
	walBlocks, err := s.wal.RescanBlocks(0, s.logger)
	if err != nil {
		return fmt.Errorf("failed to rescan wal blocks: %w", err)
	}

	for _, blk := range walBlocks {
		err := func() error {
			meta := blk.BlockMeta()

			inst, err := s.getOrCreateInstance(meta.TenantID)
			if err != nil {
				return fmt.Errorf("failed to get or create instance for tenant %s: %w", meta.TenantID, err)
			}

			inst.blocksMtx.Lock()
			defer inst.blocksMtx.Unlock()

			level.Info(s.logger).Log("msg", "reloaded wal block", "block", meta.BlockID.String())
			inst.walBlocks[(uuid.UUID)(meta.BlockID)] = blk

			level.Info(s.logger).Log("msg", "queueing replayed wal block for completion", "block", meta.BlockID.String())
			if err := s.enqueueCompleteOp(meta.TenantID, uuid.UUID(meta.BlockID), true); err != nil {
				return fmt.Errorf("failed to enqueue wal block for completion for tenant %s: %w", meta.TenantID, err)
			}

			level.Info(s.logger).Log("msg", "reloaded wal blocks", "tenant", inst.tenantID, "count", len(inst.walBlocks))

			return nil
		}()
		if err != nil {
			return err
		}
	}

	// ------------------------------------
	// Complete blocks
	// ------------------------------------
	var (
		ctx = s.ctx
		l   = s.wal.LocalBackend()
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
