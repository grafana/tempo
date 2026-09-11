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

// advanceVerification moves verification forward for a batch whose jobs have drained, and reports
// whether verification is still OUTSTANDING.
//
// The return value answers the caller's question -- may this batch settle? -- rather than describing
// what happened here, and the two are not the same: the deferred case reports outstanding while
// enqueueing nothing at all. Outstanding is also the safe answer, because settling means teardown
// and re-enabling the tenant's compaction, so anything uncertain returns true:
//
//	true  -- a pass is now running, or nothing could be checked and the next tick must retry.
//	false -- verification has nothing left to do; the caller may quiesce the batch.
//
// Verification exists because completing every job the batch knew about is not the same as the data
// being gone. A compaction that registered between the submission's busy-block snapshot and the
// batch barrier is absent from the recorded skip list, so its output block never got a job and the
// rescan cannot find it either. Re-deriving candidates from the *current* blocklist rather than from
// the batch's own bookkeeping is what catches that class, whatever produced the block.
//
// The scan runs on workers, not here: this only filters block metadata already in memory and
// enqueues, so the singleton scheduler adds no I/O per pass.
func (s *BackendScheduler) advanceVerification(ctx context.Context, tenantID string) (outstanding bool) {
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
		// The counter is not incremented here: this is where convergence is abandoned, but the batch
		// is not released until quiescence ends, and a late result can mark it dirty in between. The
		// release itself is the single place that reports, so the count matches the docs' promise of
		// one signal per redaction released without a clean pass.
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
		return false
	}

	// Clean only if this pass covers every candidate it wanted. A pass that skipped busy blocks has
	// not checked them, and marking it clean would let the batch quiesce with those blocks unscanned --
	// the deferred-only branch above already refuses to settle for exactly that reason, and a pass
	// that both enqueued and deferred is no more complete than one that only deferred. Left dirty, the
	// next drain re-derives and picks them up, bounded by maxVerifyRounds.
	//
	// Marked before the jobs are published: AddPendingJobs is the point a worker can see them, and a
	// job that completes with a match calls SetBatchVerified(false). Setting the flag afterwards would
	// let the launcher overwrite that finding and lose the dirty record for this pass.
	//
	// Note: no test proves this ordering. Both orderings pass the suite, because the losing
	// interleaving needs a worker to complete a scan between the enqueue and the flag write, and a
	// sequential test cannot produce it while a stress test would only fail intermittently. It is
	// verified by inspection -- as with the in-flight dequeue atomicity in work/inflight_test.go.
	s.work.SetBatchVerified(tenantID, deferred == 0)

	if err := s.work.AddPendingJobs(jobs); err != nil {
		// Roll the optimistic mark back rather than leaving a pass that never ran looking clean, and
		// report outstanding: returning false here would quiesce and tear down a batch that was never
		// checked. No round was consumed, so the next tick simply retries.
		s.work.SetBatchVerified(tenantID, false)
		level.Error(log.Logger).Log("msg", "redaction verification: failed to enqueue jobs", "tenant", tenantID, "err", err)
		return true
	}

	// Counted only once the pass is actually running. Consuming a round before the enqueue would let
	// repeated failures exhaust the budget without a single block being scanned, and the batch would
	// then be released as unconverged having never been checked. Unlike the flag above, the counter
	// has no reason to precede publication: nothing a worker does touches it.
	s.work.IncBatchVerifyRounds(tenantID)

	// Persist the jobs. The manifest is left to the caller, which flushes once per tick rather than
	// once per settling tenant -- and that ordering is the safe one: jobs reach disk before the
	// manifest records the round, so a crash between them leaves work to redo rather than a consumed
	// round with nothing enqueued.
	affectedIDs := make([]string, len(jobs))
	for i, j := range jobs {
		affectedIDs[i] = j.ID
	}
	if err := s.work.FlushToLocal(ctx, s.cfg.LocalWorkPath, affectedIDs); err != nil {
		// The jobs are live in memory and will still run, but they are not on disk while the manifest
		// the caller is about to write says this round happened. Record the batch dirty so a restart
		// re-derives the pass instead of reading "round consumed, nothing enqueued" as clean. Costs an
		// extra pass in the surviving case, which is the safe direction.
		s.work.SetBatchVerified(tenantID, false)
		level.Warn(log.Logger).Log("msg", "redaction verification: failed to flush scan jobs; batch left unverified", "tenant", tenantID, "err", err)
	}

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
	covered := s.coveredBlocks(tenantID, state.BatchID)
	metas := s.store.BlockMetas(tenantID)
	jobs = make([]*work.Job, 0, len(metas))

	for _, meta := range metas {
		// Window first, busy second -- the same order SubmitRedaction uses, and it matters twice over.
		// blockOverlapsWindow is branch-only while the busy lookup formats a UUID, so filtering first
		// avoids that allocation for every rejected block. More importantly, `deferred` must count only
		// blocks this pass actually wanted: counting an out-of-window block that happens to be busy
		// would report the pass deferred, and since a deferred pass consumes no round the batch would
		// re-derive every tick forever with the tenant's compaction held off.
		//
		// An indeterminate range is taken on trust, matching submission: a block whose range cannot be
		// judged is scanned rather than skipped. blockOverlapsWindow already treats an unset window as
		// matching everything, so the explicit-ID selector needs no special case here.
		if overlaps, _ := blockOverlapsWindow(meta, startNano, endNano); !overlaps {
			continue
		}

		blockID := meta.BlockID.String()
		if _, isBusy := busy[blockID]; isBusy {
			deferred++
			continue
		}
		if _, done := covered[blockID]; done {
			continue
		}

		jobs = append(jobs, &work.Job{
			ID:   uuid.New().String(),
			Type: tempopb.JobType_JOB_TYPE_REDACTION,
			JobDetail: tempopb.JobDetail{
				Tenant:  tenantID,
				BatchId: state.BatchID,
				Redaction: &tempopb.RedactionDetail{
					BlockId: blockID,
					// Verify is what makes this a scan: the dispatcher sends DRY_RUN for a job
					// carrying it, rather than the batch's mode.
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

// coveredBlocks returns the blocks this batch already holds a succeeded redaction job for.
//
// Re-scanning them is the dominant cost of a pass and buys nothing: a block whose job succeeded was
// either rewritten -- in which case the input is retired and is not in the blocklist to be a
// candidate at all -- or was scanned and found clean. What verification is actually looking for is
// the block no job ever covered: the output of a compaction that registered between the submission's
// busy-block snapshot and the batch barrier, or a block ingested since. Without this filter a pass
// re-scans every in-window block, so verifying a redaction costs about as much as the redaction did,
// once per round.
//
// A FAILED job deliberately does not count as covered: its block was not scanned, so the pass must
// look at it. Nor does an in-flight one, which the busy-block check already defers.
//
// Nor does a succeeded *verification* job, even though it is a redaction job for this batch on this
// block. A scan that found matches succeeds and queues a repair; if that repair then fails, the batch
// is dirty and passes again -- and counting the scan as coverage would skip the one block known to
// still hold matches and report the batch clean. Only a completed rewrite proves coverage.
//
// Derived from the job records rather than stored on the batch, which keeps a block-ID set off the
// single global manifest -- at tens of thousands of blocks that would be megabytes rewritten on every
// flush. The trade-off is that Prune eventually retires the job records, after which a block looks
// uncovered and is re-scanned. That degrades to the unfiltered behaviour, which is correct and merely
// slower, so the failure direction is safe.
func (s *BackendScheduler) coveredBlocks(tenantID, batchID string) map[string]struct{} {
	covered := make(map[string]struct{})
	for _, j := range s.work.ListJobs() {
		if j.GetType() != tempopb.JobType_JOB_TYPE_REDACTION || !j.IsComplete() {
			continue
		}
		if j.Tenant() != tenantID || j.JobDetail.GetBatchId() != batchID {
			continue
		}
		if j.JobDetail.GetRedaction().GetVerify() {
			continue
		}
		if blockID := j.JobDetail.GetRedaction().GetBlockId(); blockID != "" {
			covered[blockID] = struct{}{}
		}
	}
	return covered
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
//
// This leaves a deliberate asymmetry for an unwindowed query batch: its original jobs scan with no
// bound and would remove a span stamped after the request, while a pass bounded at the submission
// instant will not look at a block whose whole range post-dates it -- which clock skew or
// future-dated spans can produce. blockOverlapsWindow is an overlap test, so only a block lying
// entirely in the future is excluded; one straddling the instant is still scanned. The alternative
// is worse: an unbounded pass over an active tenant keeps matching data that arrived after the
// request, so it never comes back clean, spends its round budget every time, and makes the
// exhausted signal routine rather than actionable. Documented with the redact command.
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
//
// Takes the verify job rather than its fields. Every value it needs comes from that job, and spelling
// them out gave a signature with three adjacent strings (tenant, batch, block) and two adjacent
// int64s (the window) -- transpose either pair and it compiles, then silently mis-scopes an
// irreversible rewrite. tempodb.RedactionWindow carries named fields for the same reason.
func (s *BackendScheduler) enqueueRedactionForVerifiedBlock(ctx context.Context, verified *work.Job) {
	tenantID := verified.Tenant()
	batchID := verified.JobDetail.GetBatchId()
	detail := verified.JobDetail.GetRedaction()
	blockID := detail.GetBlockId()
	startNano, endNano := detail.GetStartTimeUnixNano(), detail.GetEndTimeUnixNano()

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

	// Recorded before the job is published so the pass cannot be read as clean if the enqueue fails,
	// and persisted immediately: this runs on the completion path, where the caller only flushes the
	// manifest once the batch has drained -- which it has not, since a repair is about to be queued.
	// Flushed only on a transition, so a pass finding many dirty blocks writes the manifest once.
	if s.work.SetBatchVerifiedForBatch(tenantID, batchID, false) {
		s.flushBatches(ctx)
	}

	// A repair that already ran counts too, which the busy-block index below cannot see: it holds only
	// pending and running work, so a retry arriving after the first repair finished would look at an
	// idle block and queue a second rewrite. Rewriting an input another worker still lists as live
	// produces a duplicate output carrying every non-matching trace again. coveredBlocks answers this
	// from the job records, which are already durable, so there is no processed-result relationship to
	// store -- at the cost of a job walk per gap found, which is acceptable because a gap is the rare
	// case. Pruned job records degrade this to the busy check alone, and PruneAge is far longer than
	// any RPC retry.
	if _, alreadyRewritten := s.coveredBlocks(tenantID, batchID)[blockID]; alreadyRewritten {
		level.Info(log.Logger).Log("msg", "redaction verification: this batch has already rewritten the block, not queueing a second repair",
			"tenant", tenantID, "batch_id", batchID, "block_id", blockID)
		return
	}

	// One repair per block. The worker retries UpdateJob after an error or a lost response, and this
	// handler runs again on every retry -- with a fresh UUID, against an AddPendingJobs that
	// deduplicates nothing -- so a retried result would queue a second rewrite of the same block, and
	// two rewrites of one block can run concurrently. A block already held by a pending or running job
	// is either that first repair or something else mid-flight; either way the batch is marked dirty
	// above, so the next pass re-derives this block and re-checks it.
	if _, held := s.work.BusyBlocksForTenant(tenantID)[blockID]; held {
		level.Info(log.Logger).Log("msg", "redaction verification: a job already holds this block, not queueing a second repair",
			"tenant", tenantID, "batch_id", batchID, "block_id", blockID)
		return
	}

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

// releaseVerificationForTimedOutJobs marks a batch unverified when the timeout, rather than a worker,
// ends one of its verification scans.
//
// Prune force-fails a job that outran DeadJobTimeout by calling j.Fail() directly, so it never
// reaches UpdateJob -- and UpdateJob's failure path is what normally records that a scan did not run.
// Without this a pass whose scan was killed by the timeout keeps the optimistic clean mark it was
// launched with, and the batch quiesces on a block that was never looked at.
func (s *BackendScheduler) releaseVerificationForTimedOutJobs(ctx context.Context, timedOut []*work.Job) {
	for _, j := range timedOut {
		if j.GetType() != tempopb.JobType_JOB_TYPE_REDACTION || !j.JobDetail.GetRedaction().GetVerify() {
			continue
		}
		if s.work.SetBatchVerifiedForBatch(j.Tenant(), j.JobDetail.GetBatchId(), false) {
			s.flushBatches(ctx)
		}
		level.Warn(log.Logger).Log("msg", "redaction verification scan timed out; batch left unverified",
			"job_id", j.ID, "tenant", j.Tenant(), "block_id", j.JobDetail.GetRedaction().GetBlockId())
	}
}
