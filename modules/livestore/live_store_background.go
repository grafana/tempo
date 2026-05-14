package livestore

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
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
	current := o.bo
	o.bo *= 2
	if o.bo > o.maxBackoff {
		o.bo = o.maxBackoff
	}

	return current
}

func (s *LiveStore) startAllBackgroundProcesses() {
	if s.cfg.holdAllBackgroundProcesses {
		level.Warn(s.logger).Log("msg", "live store has been started with all background processes suspended! this is meant for testing only")
		return
	}

	s.completeBlockLifecycle.start(s.ctx)
	close(s.startupComplete)
}

func (s *LiveStore) stopAllBackgroundProcesses() {
	s.cancel()              // this will cause the per tenant background processes to complete
	s.completeQueues.Stop() // this will cause the global complete loop by preventing additional enqueues
	s.wg.Wait()
	s.completeBlockLifecycle.stop()
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
	level.Info(s.logger).Log("msg", "starting completing loop", "index", idx)
	defer func() {
		level.Info(s.logger).Log("msg", "shutdown completing loop", "index", idx)
	}()
	for {
		op := s.completeQueues.Dequeue()
		if op == nil {
			return // queue is closed
		}
		op.attempts++

		if op.attempts > maxFlushAttempts {
			level.Error(s.logger).Log("msg", "failed to complete operation", "tenant", op.tenantID, "block", op.blockID, "attempts", op.attempts)
			observeFailedOp(op)
			continue
		}

		if err := s.processCompleteOp(op); err != nil {
			return
		}
	}
}

// processCompleteOp completes a single block. Returns an error if global loop should exit.
func (s *LiveStore) processCompleteOp(op *completeOp) error {
	ctx, span := tracer.Start(s.ctx, "LiveStore.processCompleteOp",
		oteltrace.WithAttributes(
			attribute.String("tenant", op.tenantID),
			attribute.String("blockID", op.blockID.String()),
			attribute.Int("attempt", op.attempts),
		))
	defer span.End()

	start := time.Now()
	inst, err := s.getOrCreateInstance(op.tenantID)
	if err != nil {
		level.Error(s.logger).Log("msg", "failed to retrieve instance for completion", "tenant", op.tenantID, "err", err)
		observeFailedOp(op)
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return err
	}

	// If the context is cancelled (shutdown), abandon the completion. The WAL block remains on
	// disk and will be re-enqueued by reloadBlocks() on next startup.
	if ctx.Err() != nil {
		level.Info(s.logger).Log("msg", "abandoning WAL block completion on shutdown, will replay on restart", "tenant", op.tenantID, "block", op.blockID)
		s.completeQueues.Clear(op)
		return nil
	}

	completeBlock, err := inst.completeBlock(ctx, op.blockID)
	if err != nil {
		metricCompletionDuration.Observe(time.Since(start).Seconds())
		s.retryCompleteOp(op, span, "failed to complete block", err)
		return nil
	}

	if completeBlock == nil {
		// completeBlock only returns a block when this call converts a WAL block.
		// On a retry after lifecycle handling fails, the WAL block may already be
		// gone while the completed block is still present in inst.completeBlocks.
		completeBlock = inst.blocks.Load().completeBlocks[op.blockID]
	}

	if completeBlock != nil {
		if err := s.completeBlockLifecycle.onCompletedBlock(ctx, op.tenantID, completeBlock); err != nil {
			metricCompletionDuration.Observe(time.Since(start).Seconds())
			s.retryCompleteOp(op, span, "failed to apply complete block lifecycle", err)
			return nil
		}
	}

	metricCompletionDuration.Observe(time.Since(start).Seconds())
	metricBlocksCompleted.Inc()
	s.completeQueues.Clear(op)
	return nil
}

func (s *LiveStore) retryCompleteOp(op *completeOp, span oteltrace.Span, msg string, err error) {
	level.Error(s.logger).Log("msg", msg, "tenant", op.tenantID, "block", op.blockID, "err", err)
	observeFailedOp(op)
	span.RecordError(err)

	delay := op.backoff()
	op.at = time.Now().Add(delay)

	metricCompletionRetries.Inc()

	go func() {
		time.Sleep(delay)

		if err := s.requeueOp(op); err != nil {
			_ = level.Error(s.logger).Log("msg", "failed to requeue block for flushing", "tenant", op.tenantID, "block", op.blockID, "err", err)
		}
	}()
}

func (s *LiveStore) startPerTenantCutToWalLoop(inst *instance) {
	s.cutToWalWg.Add(1)
	go func() {
		defer s.cutToWalWg.Done()

		// Wait for startup to finish; also listen on cutToWalStop so we can
		// exit if shutdown happens before startup completes.
		select {
		case <-s.startupComplete:
		case <-s.cutToWalStop:
			return
		}

		ticker := time.NewTicker(s.cfg.InstanceFlushPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.cutOneInstanceToWal(s.ctx, inst, false)
			case <-s.cutToWalStop:
				return
			case <-s.ctx.Done():
				return
			}
		}
	}()
}

func (s *LiveStore) stopAllCutToWalLoops() {
	close(s.cutToWalStop)
	s.cutToWalWg.Wait()
}

func (s *LiveStore) perTenantCleanupLoop(inst *instance) {
	// ticker
	ticker := time.NewTicker(s.cfg.InstanceCleanupPeriod)
	defer ticker.Stop()

	// Reclaim at a fraction of the grace window so blocks are deleted
	// soon after expiry, not at the next InstanceCleanupPeriod tick.
	reclaimInterval := s.cfg.BlockReclaimGrace / 4
	if reclaimInterval < time.Second {
		reclaimInterval = time.Second
	}
	reclaimTicker := time.NewTicker(reclaimInterval)
	defer reclaimTicker.Stop()

	for {
		select {
		case <-ticker.C:
			// dump any blocks that have been flushed for a while
			err := inst.deleteOldBlocks()
			if err != nil {
				level.Error(s.logger).Log("msg", "failed to delete old blocks", "err", err)
			}
		case <-reclaimTicker.C:
			for _, r := range inst.reclaim.reclaim() {
				if r.Err != nil {
					level.Error(s.logger).Log("msg", "reclaim failed", "tenant", r.Tenant, "block_id", r.BlockID.String(), "block_type", r.BlockType, "err", r.Err)
					continue
				}
				metricBlocksClearedTotal.WithLabelValues(r.BlockType).Inc()
				level.Info(s.logger).Log("msg", "reclaimed block", "tenant", r.Tenant, "block_id", r.BlockID.String(), "block_type", r.BlockType)
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
	// Reclaim tombstoned blocks left by an unclean shutdown before
	// reloading, so they don't get scanned as live.
	if n, err := s.wal.ClearTombstonedBlocks(); err != nil {
		level.Warn(s.logger).Log("msg", "failed to clear tombstoned wal blocks at startup", "err", err)
	} else if n > 0 {
		level.Info(s.logger).Log("msg", "cleared tombstoned wal blocks at startup", "count", n)
	}
	if n, err := s.wal.LocalBackend().ClearTombstonedBlocks(); err != nil {
		level.Warn(s.logger).Log("msg", "failed to clear tombstoned complete blocks at startup", "err", err)
	} else if n > 0 {
		level.Info(s.logger).Log("msg", "cleared tombstoned complete blocks at startup", "count", n)
	}

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
			inst.blocks.Store(inst.blocks.Load().withWALBlockAdded((uuid.UUID)(meta.BlockID), blk))

			level.Info(s.logger).Log("msg", "queueing replayed wal block for completion", "block", meta.BlockID.String(), "size", blk.DataLength())
			if err := s.enqueueCompleteOp(meta.TenantID, uuid.UUID(meta.BlockID), true); err != nil {
				return fmt.Errorf("failed to enqueue wal block for completion for tenant %s: %w", meta.TenantID, err)
			}

			level.Info(s.logger).Log("msg", "reloaded wal blocks", "tenant", inst.tenantID, "count", len(inst.blocks.Load().walBlocks))

			return nil
		}()
		if err != nil {
			return err
		}
	}

	level.Info(s.logger).Log("msg", "wal blocks to complete at startup", "count", len(walBlocks))

	// ------------------------------------
	// Complete blocks
	// ------------------------------------
	level.Info(s.logger).Log("msg", "reloading completed blocks")

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

			lb := NewLocalBlock(ctx, blk, l)

			inst, err := s.getOrCreateInstance(tenant)
			if err != nil {
				return fmt.Errorf("failed to get or create instance for tenant %s during complete block reload: %w", tenant, err)
			}

			inst.blocksMtx.Lock()
			inst.blocks.Store(inst.blocks.Load().withCompleteBlockAdded(id, lb))
			inst.blocksMtx.Unlock()

			if err := s.completeBlockLifecycle.onReloadedBlock(ctx, tenant, lb); err != nil {
				return fmt.Errorf("failed to apply complete block lifecycle to reloaded block %s in tenant %s: %w", id.String(), tenant, err)
			}
		}
	}

	level.Info(s.logger).Log("msg", "done reloading completed blocks")

	return nil
}
