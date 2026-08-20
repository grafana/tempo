package backendscheduler

import (
	"context"

	"github.com/go-kit/log/level"
	"github.com/google/uuid"

	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/log"
)

// maxVerifyRounds bounds how many verification passes one batch may run.
//
// A pass that finds matches enqueues redaction jobs, which on completion trigger another pass, so
// without a bound a tenant whose in-window data keeps growing would re-verify forever and hold its
// compaction off indefinitely. A clean redaction needs one pass; one that hit the submission-time
// coverage gap needs two; beyond that the batch is not converging and an operator should look.
const maxVerifyRounds = 3

// runVerification enqueues a verification pass for a batch whose jobs have drained, returning
// whether it enqueued anything. A true result means the batch has outstanding work again and must
// not enter quiescence.
//
// Verification exists because completing every job the batch knew about is not the same as the data
// being gone. A compaction that registered between the submission's busy-block snapshot and the
// batch barrier is absent from the recorded skip list, so its output block never got a job and the
// rescan cannot find it either. Re-deriving candidates from the *current* blocklist rather than from
// the batch's own bookkeeping is what catches that class, whatever produced the block.
//
// The scan runs on workers, not here: this only filters block metadata already in memory and
// enqueues, so the singleton scheduler adds no I/O per pass.
func (s *BackendScheduler) runVerification(ctx context.Context, tenantID string) bool {
	state, ok := s.work.RedactionVerifyState(tenantID)
	if !ok {
		return false
	}

	if state.Verified {
		// The last pass came back clean, so there is nothing left to re-check and the batch may
		// quiesce. Without this the drained state after a clean pass is indistinguishable from a
		// batch never verified, and every sweep would launch another pass until the round budget
		// ran out -- reporting a successful redaction as one that failed to converge.
		return false
	}

	if state.VerifyRounds >= maxVerifyRounds {
		// Stop rather than keep the tenant's compaction paused. The batch proceeds to quiescence and
		// teardown, so this is reported as an unconverged redaction rather than a completed one.
		metricRedactionVerifyExhausted.WithLabelValues(tenantID).Inc()
		level.Warn(log.Logger).Log("msg", "redaction verification did not converge; releasing the batch without a clean pass -- operator should re-submit if traces are still present",
			"tenant", tenantID, "batch_id", state.BatchID, "rounds", state.VerifyRounds)
		return false
	}

	jobs := s.verificationJobs(tenantID, state)
	if len(jobs) == 0 {
		// Nothing in scope to re-check: treat as verified so the batch can quiesce.
		return false
	}

	if err := s.work.AddPendingJobs(jobs); err != nil {
		// Leave verify_rounds alone so the next tick retries rather than burning a round on a pass
		// that never ran.
		level.Error(log.Logger).Log("msg", "redaction verification: failed to enqueue jobs", "tenant", tenantID, "err", err)
		return false
	}

	// Optimistic: the pass is assumed clean and any job in it that finds a match clears this, which
	// both repairs the block and puts the batch back in line for a further pass.
	s.work.SetBatchVerified(tenantID, true)
	s.work.IncBatchVerifyRounds(tenantID)
	s.flushBatches(ctx)

	metricRedactionVerifyRounds.WithLabelValues(tenantID).Inc()
	level.Info(log.Logger).Log("msg", "redaction verification pass enqueued",
		"tenant", tenantID, "batch_id", state.BatchID, "round", state.VerifyRounds+1, "blocks", len(jobs))
	return true
}

// verificationJobs builds the dry-run scan jobs for a verification pass.
//
// Candidates come from the current blocklist filtered to the batch's window, not from the blocks the
// batch touched. A trace stamped inside the window but ingested late lands in a block the batch never
// produced, so a frozen block set would miss it and report the redaction done.
//
// Blocks already held by another job are skipped: enqueueing a second job for a block being
// compacted or redacted would race it, and a block still in flight is by definition not yet settled
// enough to verify. The next pass re-derives candidates, so a skipped block is re-checked then.
func (s *BackendScheduler) verificationJobs(tenantID string, state work.RedactionVerifyState) []*work.Job {
	// The explicit-ID selector applies no time bound -- RedactBlock refuses an ID list combined with
	// a window -- so its verification scan runs unwindowed, exactly as its original jobs did.
	var startNano, endNano int64
	if !state.HasTraceIDs {
		startNano, endNano = state.StartTimeUnixNano, state.EndTimeUnixNano
		if startNano == 0 && endNano == 0 {
			// No caller-specified window: the effective scope is everything up to submission. An
			// unbounded re-scan would match data ingested after the request, which the operator
			// never asked to remove and which would keep the loop from converging.
			endNano = state.CreatedAtUnixNano
		}
	}

	busy := s.work.BusyBlocksForTenant(tenantID)
	metas := s.store.BlockMetas(tenantID)
	jobs := make([]*work.Job, 0, len(metas))

	for _, meta := range metas {
		if _, isBusy := busy[meta.BlockID.String()]; isBusy {
			continue
		}
		if !state.HasTraceIDs {
			// An indeterminate range is taken on trust, matching submission: a block that cannot be
			// judged is scanned rather than skipped.
			if overlaps, _ := blockOverlapsWindow(meta, startNano, endNano); !overlaps {
				continue
			}
		}

		jobs = append(jobs, &work.Job{
			ID:   uuid.New().String(),
			Type: tempopb.JobType_JOB_TYPE_REDACTION,
			JobDetail: tempopb.JobDetail{
				Tenant:  tenantID,
				BatchId: state.BatchID,
				Redaction: &tempopb.RedactionDetail{
					BlockId: meta.BlockID.String(),
					// Verify is the only field Next() does not overwrite from the batch, which is
					// what keeps this a scan rather than a rewrite.
					Verify: true,
				},
			},
		})
	}

	return jobs
}

// enqueueRedactionForVerifiedBlock creates a real redaction job for a block a verification pass
// found still holding matches. Acting per result rather than tallying the whole pass first means a
// gap is repaired as soon as it is seen, and no barrier is needed across the pass.
func (s *BackendScheduler) enqueueRedactionForVerifiedBlock(ctx context.Context, tenantID, batchID, blockID string) {
	job := &work.Job{
		ID:   uuid.New().String(),
		Type: tempopb.JobType_JOB_TYPE_REDACTION,
		JobDetail: tempopb.JobDetail{
			Tenant:  tenantID,
			BatchId: batchID,
			Redaction: &tempopb.RedactionDetail{
				BlockId: blockID,
				// Verify deliberately unset: this job rewrites. Next() injects the batch's selector,
				// mode and window.
			},
		},
	}

	if err := s.work.AddPendingJobs([]*work.Job{job}); err != nil {
		level.Error(log.Logger).Log("msg", "redaction verification: failed to enqueue repair job",
			"tenant", tenantID, "block_id", blockID, "err", err)
		return
	}

	// This pass is not clean, so the batch must be verified again once the repair lands rather than
	// quiescing on the strength of the pass that found the gap.
	s.work.SetBatchVerified(tenantID, false)

	metricRedactionVerifyGaps.WithLabelValues(tenantID).Inc()
	level.Warn(log.Logger).Log("msg", "redaction verification found a block still holding matches; enqueued a redaction job",
		"tenant", tenantID, "batch_id", batchID, "block_id", blockID)

	if err := s.work.FlushToLocal(ctx, s.cfg.LocalWorkPath, []string{job.ID}); err != nil {
		level.Warn(log.Logger).Log("msg", "redaction verification: failed to flush repair job", "err", err)
	}
}
