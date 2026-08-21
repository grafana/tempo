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

	jobs, deferred := s.verificationJobs(tenantID, state)
	if len(jobs) == 0 {
		if deferred > 0 {
			// Every candidate is held by another job, so nothing was actually checked. Quiescing here
			// would tear the batch down unverified -- and a block mid-compaction is the likeliest way
			// the uncovered block this pass exists to find came about. Report outstanding work so the
			// next tick re-derives candidates instead.
			metricRedactionVerifyDeferred.WithLabelValues(tenantID).Inc()
			level.Info(log.Logger).Log("msg", "redaction verification deferred: every candidate block is busy",
				"tenant", tenantID, "batch_id", state.BatchID, "deferred_blocks", deferred)
			return true
		}
		// Genuinely nothing in scope. Record it so the batch is not re-derived every tick.
		s.work.SetBatchVerified(tenantID, true)
		s.flushBatches(ctx)
		return false
	}

	// Marked before the jobs are published: AddPendingJobs is the point a worker can see them, and a
	// job that completes with a match calls SetBatchVerified(false). Setting the flag afterwards would
	// let the launcher overwrite that finding and lose the dirty record for this pass.
	s.work.SetBatchVerified(tenantID, true)
	s.work.IncBatchVerifyRounds(tenantID)

	if err := s.work.AddPendingJobs(jobs); err != nil {
		// Roll the optimistic mark back rather than leaving a pass that never ran looking clean.
		s.work.SetBatchVerified(tenantID, false)
		level.Error(log.Logger).Log("msg", "redaction verification: failed to enqueue jobs", "tenant", tenantID, "err", err)
		return false
	}

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
func (s *BackendScheduler) verificationJobs(tenantID string, state work.RedactionVerifyState) (jobs []*work.Job, deferred int) {
	startNano, endNano := verificationWindow(state)

	busy := s.work.BusyBlocksForTenant(tenantID)
	metas := s.store.BlockMetas(tenantID)
	jobs = make([]*work.Job, 0, len(metas))

	for _, meta := range metas {
		if _, isBusy := busy[meta.BlockID.String()]; isBusy {
			deferred++
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
					// Verify keeps this a scan rather than a rewrite: Next() injects the batch's
					// mode over everything else, and does not touch this field.
					Verify: true,
					// The resolved window travels on the job. Filtering candidates by it is not
					// enough -- the scan itself has to be bounded, or a pass over a block created
					// after submission matches traces the request never covered, and the repair job
					// that follows deletes them.
					StartTimeUnixNano: startNano,
					EndTimeUnixNano:   endNano,
				},
			},
		})
	}

	return jobs, deferred
}

// verificationWindow resolves the bounds a verification pass scans with.
//
// An unbounded re-scan would match data ingested after the request -- which the operator never asked
// to remove, and which keeps arriving, so the loop would never converge. With no caller-specified
// window the effective scope is therefore everything up to submission.
//
// The explicit-ID selector is the exception and returns no bounds at all: it applies no time bound,
// and RedactBlock refuses an ID list combined with a window, so its pass runs unwindowed exactly as
// its original jobs did.
func verificationWindow(state work.RedactionVerifyState) (startNano, endNano int64) {
	if state.HasTraceIDs {
		return 0, 0
	}
	if state.StartTimeUnixNano == 0 && state.EndTimeUnixNano == 0 {
		return 0, state.CreatedAtUnixNano
	}
	return state.StartTimeUnixNano, state.EndTimeUnixNano
}

// enqueueRedactionForVerifiedBlock creates a real redaction job for a block a verification pass
// found still holding matches. Acting per result rather than tallying the whole pass first means a
// gap is repaired as soon as it is seen, and no barrier is needed across the pass.
func (s *BackendScheduler) enqueueRedactionForVerifiedBlock(ctx context.Context, tenantID, batchID, blockID string, startNano, endNano int64) {
	// Fail closed on identity. A verify job can outlive its batch (Prune fails it, the batch
	// quiesces and is removed) and still report a match. Enqueueing then would either be dropped in
	// Next(), losing a confirmed surviving trace with nothing tracking it, or -- worse -- be matched
	// to a batch the operator submitted since, and rewritten under that batch's scope.
	if batch := s.work.GetBatch(tenantID); batch == nil || batch.BatchId != batchID {
		metricRedactionVerifyOrphanedGaps.WithLabelValues(tenantID).Inc()
		level.Warn(log.Logger).Log("msg", "redaction verification found matches but its batch is gone; not enqueueing a repair -- resubmit the redaction",
			"tenant", tenantID, "batch_id", batchID, "block_id", blockID)
		return
	}

	// Recorded before the job is published so the pass cannot be read as clean if the enqueue fails.
	s.work.SetBatchVerified(tenantID, false)
	s.flushBatches(ctx)

	job := &work.Job{
		ID:   uuid.New().String(),
		Type: tempopb.JobType_JOB_TYPE_REDACTION,
		JobDetail: tempopb.JobDetail{
			Tenant:  tenantID,
			BatchId: batchID,
			Redaction: &tempopb.RedactionDetail{
				BlockId: blockID,
				// Verify deliberately unset: this job rewrites. It carries the window the scan that
				// found the match ran under rather than the batch's, because a batch submitted
				// without a window is unbounded and this block may hold post-submission data that
				// the request never covered.
				StartTimeUnixNano: startNano,
				EndTimeUnixNano:   endNano,
			},
		},
	}

	if err := s.work.AddPendingJobs([]*work.Job{job}); err != nil {
		level.Error(log.Logger).Log("msg", "redaction verification: failed to enqueue repair job",
			"tenant", tenantID, "block_id", blockID, "err", err)
		return
	}

	metricRedactionVerifyGaps.WithLabelValues(tenantID).Inc()
	level.Warn(log.Logger).Log("msg", "redaction verification found a block still holding matches; enqueued a redaction job",
		"tenant", tenantID, "batch_id", batchID, "block_id", blockID)

	if err := s.work.FlushToLocal(ctx, s.cfg.LocalWorkPath, []string{job.ID}); err != nil {
		level.Warn(log.Logger).Log("msg", "redaction verification: failed to flush repair job", "err", err)
	}
}
