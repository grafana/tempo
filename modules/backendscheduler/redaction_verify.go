package backendscheduler

import (
	"context"

	"github.com/go-kit/log/level"
	"github.com/google/uuid"

	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/log"
)

// auditDrainedBatch enqueues a read-only coverage audit for a batch whose rewrite jobs have drained.
//
// What it checks, and what it deliberately does not: the audit re-derives the batch's in-window
// blocks and scans any the batch holds no completed rewrite for. That catches a block left uncovered
// because its job failed or was never selected. It does NOT establish that the redaction was complete
// against object storage, because the scheduler's blocklist is refreshed on blocklist_poll (five
// minutes by default) while a batch lives for only a couple of maintenance ticks after draining, so a
// block created during the batch cannot be visible to the audit. Completeness against storage is an
// operator procedure: re-run the redaction with --dry-run over the same window once a poll has
// elapsed, where a non-zero match count means it is incomplete.
//
// The audit does not gate teardown. It enqueues scans and returns; the scans are ordinary redaction
// jobs, so redactionBatchActive counts them and the existing quiescence path removes the batch once
// they drain. There is no verdict, no round budget and no automatic repair: a finding is reported
// through redaction_verify_traces_found_total and a warning log, and acting on it is the operator's
// call. Everything here is derived from the job records, so it adds no durable state.
func (s *BackendScheduler) auditDrainedBatch(ctx context.Context, tenantID string) {
	state, ok := s.work.RedactionVerifyState(tenantID)
	if !ok {
		return
	}

	// Once per batch. A batch that already has scan jobs, in any state, has been audited; re-deriving
	// on each tick would re-scan the same blocks for as long as the batch exists.
	if s.batchHasAuditJobs(tenantID, state.BatchID) {
		return
	}

	jobs := s.auditJobs(tenantID, state)
	if len(jobs) == 0 {
		return
	}

	if err := s.work.AddPendingJobs(jobs); err != nil {
		level.Error(log.Logger).Log("msg", "redaction audit: failed to enqueue scans", "tenant", tenantID, "err", err)
		return
	}

	affectedIDs := make([]string, 0, len(jobs))
	for _, j := range jobs {
		affectedIDs = append(affectedIDs, j.ID)
	}
	if err := s.work.FlushToLocal(ctx, s.cfg.LocalWorkPath, affectedIDs); err != nil {
		// The scans are live in memory and will still run; they are simply not replayable across a
		// restart. Nothing downstream depends on them having run, so this is a lost audit rather than
		// an incorrect result.
		level.Warn(log.Logger).Log("msg", "redaction audit: failed to flush scans", "tenant", tenantID, "err", err)
	}

	level.Info(log.Logger).Log("msg", "redaction audit enqueued",
		"tenant", tenantID, "batch_id", state.BatchID, "blocks", len(jobs))
}

// batchHasAuditJobs reports whether this batch has already had an audit enqueued.
//
// Derived rather than stored: a scan job carries the batch ID and the verify flag, which is the same
// record the audit would otherwise have to persist. Prune retires job records only once they are
// terminal and older than PruneAge, by which point the batch itself is long gone.
//
// Both queues are searched because they are separate maps: ListJobs walks the active and terminal
// jobs, and a scan sitting in the pending queue appears only in ListAllPendingJobs. Checking just the
// active map would leave the guard relying on auditJobs' busy-block filter to suppress a second
// audit -- which it does, since a pending scan makes its block busy, but that is a filter three
// functions away rather than the thing this guard claims to do.
func (s *BackendScheduler) batchHasAuditJobs(tenantID, batchID string) bool {
	isAuditJob := func(j *work.Job) bool {
		if j.GetType() != tempopb.JobType_JOB_TYPE_REDACTION {
			return false
		}
		if j.Tenant() != tenantID || j.JobDetail.GetBatchId() != batchID {
			return false
		}
		return j.JobDetail.GetRedaction().GetVerify()
	}

	for _, j := range s.work.ListAllPendingJobs() {
		if isAuditJob(j) {
			return true
		}
	}
	for _, j := range s.work.ListJobs() {
		if isAuditJob(j) {
			return true
		}
	}
	return false
}

// auditJobs builds the read-only scan jobs for one audit.
//
// Blocks the batch already rewrote are skipped: a completed rewrite is the coverage the audit looks
// for. Busy blocks are skipped because a second job on a block in flight would race it; the audit runs
// once, so a block busy now is simply not audited, which is the honest outcome for a best-effort check.
func (s *BackendScheduler) auditJobs(tenantID string, state work.RedactionVerifyState) []*work.Job {
	startNano, endNano := auditWindow(state)

	busy := s.work.BusyBlocksForTenant(tenantID)
	covered := s.coveredBlocks(tenantID, state.BatchID)

	var jobs []*work.Job
	for _, meta := range s.store.BlockMetas(tenantID) {
		// Window first: blockOverlapsWindow is branch-only while the busy lookup formats a UUID. An
		// indeterminate range is taken on trust, matching submission -- a block whose range cannot be
		// judged is scanned rather than skipped.
		if overlaps, _ := blockOverlapsWindow(meta, startNano, endNano); !overlaps {
			continue
		}

		blockID := meta.BlockID.String()
		if _, isBusy := busy[blockID]; isBusy {
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
					// Verify is what makes this read-only: the dispatcher sends DRY_RUN for a job
					// carrying it rather than the batch's mode, and the worker forces dry-run again.
					Verify: true,
					// The resolved window travels on the job. Filtering candidates by it is not enough:
					// the scan itself has to be bounded, or it reports matches the request never covered.
					StartTimeUnixNano: startNano,
					EndTimeUnixNano:   endNano,
				},
			},
		})
	}

	return jobs
}

// coveredBlocks returns the blocks this batch holds a completed rewrite for.
//
// A FAILED job does not count: its block was not rewritten, so it is exactly what the audit should
// look at. Nor does a succeeded audit scan, which reads but never writes.
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

// auditWindow resolves the bounds a scan runs with.
//
// An unbounded re-scan would match data ingested after the request, which the operator never asked to
// remove. With no caller-specified window the scope is therefore everything up to submission. The
// explicit-ID selector is the exception and returns no bounds: it applies no time bound, and
// RedactBlock refuses an ID list combined with a window.
func auditWindow(state work.RedactionVerifyState) (startNano, endNano int64) {
	if state.HasTraceIDs {
		return 0, 0
	}
	if state.StartTimeUnixNano == 0 && state.EndTimeUnixNano == 0 {
		return 0, state.CreatedAtUnixNano
	}
	return state.StartTimeUnixNano, state.EndTimeUnixNano
}
