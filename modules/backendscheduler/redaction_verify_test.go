package backendscheduler

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
)

// TestRunVerificationSkipsAVerifiedBatch pins the flag that makes the loop converge. A drained batch
// after a clean pass looks exactly like one that has never been verified, so without this the sweep
// launches a pass every tick until the round budget is spent -- reporting every successful redaction
// as one that failed to converge, and holding the tenant's compaction off for the whole budget.
func TestRunVerificationSkipsAVerifiedBatch(t *testing.T) {
	const tenant = "t-verify-clean"
	// Blocks must exist and be in scope: with an empty store the pass returns false via the
	// nothing-to-scan path and the test would pass with the guard deleted.
	ctx, s, _ := newVerifyScheduler(t, tenant)

	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "b", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
		Verified: true, VerifyRounds: 1,
	}))

	premise, _ := s.verificationJobs(tenant, mustVerifyState(t, s, tenant))
	require.NotEmpty(t, premise,
		"premise: there are in-scope blocks, so only the guard can stop the pass")

	require.False(t, s.advanceVerification(ctx, tenant), "a verified batch must not be re-verified")
	require.False(t, s.work.HasJobsForTenant(tenant, tempopb.JobType_JOB_TYPE_REDACTION),
		"no scan jobs may be enqueued for a batch already verified")

	state, ok := s.work.RedactionVerifyState(tenant)
	require.True(t, ok)
	require.Equal(t, int32(1), state.VerifyRounds, "a skipped pass must not consume a round")
}

// TestRunVerificationCircuitBreaker pins that a batch which keeps finding matches is eventually
// released rather than pausing the tenant's compaction indefinitely.
func TestRunVerificationCircuitBreaker(t *testing.T) {
	const tenant = "t-verify-exhausted"
	// Blocks in scope for the same reason as above: the guard must be what stops the pass.
	ctx, s, _ := newVerifyScheduler(t, tenant)

	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "b", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
		VerifyRounds: maxVerifyRounds,
	}))

	premise, _ := s.verificationJobs(tenant, mustVerifyState(t, s, tenant))
	require.NotEmpty(t, premise,
		"premise: there are in-scope blocks, so only the round limit can stop the pass")

	require.False(t, s.advanceVerification(ctx, tenant),
		"at the round limit the batch is released so compaction can resume")
	require.False(t, s.work.HasJobsForTenant(tenant, tempopb.JobType_JOB_TYPE_REDACTION))
}

// TestRunVerificationWithNoBatch guards the sweep against a batch removed between the drain check
// and the pass.
func TestRunVerificationWithNoBatch(t *testing.T) {
	ctx, s := newQuiescenceScheduler(t)
	require.False(t, s.advanceVerification(ctx, "t-absent"))
}

// newVerifyScheduler builds a scheduler over a store holding three blocks for tenant, one per day
// starting `daysAgo` back, so a window can select a strict subset of them.
func newVerifyScheduler(t *testing.T, tenant string) (context.Context, *BackendScheduler, func(int) time.Time) {
	t.Helper()
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	tmpDir := t.TempDir()
	cfg.LocalWorkPath = tmpDir
	cfg.MaintenanceInterval = time.Minute

	ctx, cancel := context.WithCancel(context.Background())
	store, rr, ww := newStore(ctx, t, tmpDir)
	t.Cleanup(func() {
		cancel()
		store.Shutdown()
	})

	base := time.Now().Add(-10 * 24 * time.Hour)
	day := func(n int) time.Time { return base.Add(time.Duration(n) * 24 * time.Hour) }
	writeTenantBlocksWithRanges(ctx, t, backend.NewWriter(ww), tenant, [][2]time.Time{
		{day(0), day(1)},
		{day(4), day(5)},
		{day(8), day(9)},
	})
	// All three must be polled before the filter is exercised, or a count assertion can pass
	// without the filter ever running.
	require.Eventually(t, func() bool { return len(store.BlockMetas(tenant)) == 3 },
		5*time.Second, 50*time.Millisecond, "all three blocks must be polled")

	limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	s, err := New(cfg, store, limits, rr, ww)
	require.NoError(t, err)
	return ctx, s, day
}

// TestVerificationJobsOnlyEnqueueScans is the safety property: a verification pass must never
// produce a job that rewrites. Next() overwrites a job's mode from the batch, so `verify` is the
// only field that survives dispatch to distinguish a scan from a rewrite.
func TestVerificationJobsOnlyEnqueueScans(t *testing.T) {
	const tenant = "t-verify-scans"
	_, s, _ := newVerifyScheduler(t, tenant)

	jobs, _ := s.verificationJobs(tenant, work.RedactionVerifyState{
		BatchID:           "b",
		CreatedAtUnixNano: time.Now().UnixNano(),
	})

	require.Len(t, jobs, 3, "every block up to submission is in scope")
	for _, j := range jobs {
		require.True(t, j.JobDetail.GetRedaction().GetVerify(),
			"a verification job must be a scan, never a rewrite")
		require.Equal(t, "b", j.JobDetail.GetBatchId())
		require.Equal(t, tempopb.JobType_JOB_TYPE_REDACTION, j.Type)
	}
}

// TestVerificationJobsRespectTheWindow pins that a pass scans only in-window blocks. Scanning
// everything would re-match data ingested after the request; scanning too little would let a block
// holding in-window matches pass as verified.
func TestVerificationJobsRespectTheWindow(t *testing.T) {
	const tenant = "t-verify-window"
	_, s, day := newVerifyScheduler(t, tenant)
	// Blocks span [d0,d1], [d4,d5], [d8,d9]; bracket only the middle one.
	windowStart, windowEnd := day(3).UnixNano(), day(6).UnixNano()

	jobs, _ := s.verificationJobs(tenant, work.RedactionVerifyState{
		BatchID:           "b",
		StartTimeUnixNano: windowStart,
		EndTimeUnixNano:   windowEnd,
	})

	require.Len(t, jobs, 1, "only the block overlapping the window is scanned")
	// The bound must travel on the job, not just filter candidates. Next() dispatches whatever the
	// job carries; a job with a zero window is scanned unbounded, which matches traces outside the
	// request and hands them to a rewrite.
	require.Equal(t, windowStart, jobs[0].JobDetail.GetRedaction().GetStartTimeUnixNano())
	require.Equal(t, windowEnd, jobs[0].JobDetail.GetRedaction().GetEndTimeUnixNano())
}

// TestVerificationJobsCarryTheDerivedWindow covers the case that can actually be violated: a batch
// submitted without a window. Its effective scope is everything up to submission, and that bound has
// to reach the worker -- filtering candidate blocks by it is not enough, because the scan of a block
// that straddles submission would still match data ingested after the request.
func TestVerificationJobsCarryTheDerivedWindow(t *testing.T) {
	const tenant = "t-verify-derived-window"
	_, s, _ := newVerifyScheduler(t, tenant)
	created := time.Now().UnixNano()

	jobs, _ := s.verificationJobs(tenant, work.RedactionVerifyState{
		BatchID:           "b",
		CreatedAtUnixNano: created,
	})

	require.NotEmpty(t, jobs)
	for _, j := range jobs {
		require.Zero(t, j.JobDetail.GetRedaction().GetStartTimeUnixNano())
		require.Equal(t, created, j.JobDetail.GetRedaction().GetEndTimeUnixNano(),
			"an unwindowed batch verifies only up to its submission instant")
	}
}

// TestVerificationJobsSkipBusyBlocks pins that a block already held by another job is left alone.
// Enqueueing a second job for a block mid-compaction would race it, and the next pass re-derives
// candidates so the block is re-checked rather than dropped.
func TestVerificationJobsSkipBusyBlocks(t *testing.T) {
	const tenant = "t-verify-busy"
	_, s, _ := newVerifyScheduler(t, tenant)

	state := work.RedactionVerifyState{
		BatchID:           "b",
		CreatedAtUnixNano: time.Now().UnixNano(),
	}
	all, _ := s.verificationJobs(tenant, state)
	require.Len(t, all, 3)

	// Register a compaction job holding one of those blocks.
	busyBlock := all[0].JobDetail.GetRedaction().GetBlockId()
	s.work.RegisterJob(&work.Job{
		ID:   "compaction-holding-a-block",
		Type: tempopb.JobType_JOB_TYPE_COMPACTION,
		JobDetail: tempopb.JobDetail{
			Tenant:     tenant,
			Compaction: &tempopb.CompactionDetail{Input: []string{busyBlock}},
		},
	})

	remaining, deferred := s.verificationJobs(tenant, state)
	require.Len(t, remaining, 2, "the busy block is skipped")
	require.Equal(t, 1, deferred, "the skipped block is reported, not silently dropped")
	for _, j := range remaining {
		require.NotEqual(t, busyBlock, j.JobDetail.GetRedaction().GetBlockId())
	}
}

// TestVerificationWindowDerivation pins the scan bounds, which decide what a pass can find.
//
// An unbounded re-scan would match data ingested after the request -- data the operator never asked
// to remove -- and the loop would never converge. The explicit-ID selector is the exception: it
// applies no time bound, and RedactBlock refuses an ID list combined with a window, so its pass runs
// unwindowed exactly as its original jobs did.
func TestVerificationWindowDerivation(t *testing.T) {
	created := time.Now().UnixNano()
	start := created - int64(48*time.Hour)
	end := created - int64(24*time.Hour)

	for _, tc := range []struct {
		name               string
		state              work.RedactionVerifyState
		wantStart, wantEnd int64
	}{
		{
			name:      "explicit window is used as given",
			state:     work.RedactionVerifyState{StartTimeUnixNano: start, EndTimeUnixNano: end},
			wantStart: start, wantEnd: end,
		},
		{
			name:      "no window falls back to everything up to submission",
			state:     work.RedactionVerifyState{CreatedAtUnixNano: created},
			wantStart: 0, wantEnd: created,
		},
		{
			name:      "trace-ID selector scans unwindowed",
			state:     work.RedactionVerifyState{HasTraceIDs: true, CreatedAtUnixNano: created},
			wantStart: 0, wantEnd: 0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gotStart, gotEnd := verificationWindow(tc.state)
			require.Equal(t, tc.wantStart, gotStart, "start bound")
			require.Equal(t, tc.wantEnd, gotEnd, "end bound")
		})
	}
}

func mustVerifyState(t *testing.T, s *BackendScheduler, tenant string) work.RedactionVerifyState {
	t.Helper()
	state, ok := s.work.RedactionVerifyState(tenant)
	require.True(t, ok)
	return state
}

// TestVerificationRunsOnNormalCompletion covers the path that matters and was missed: a batch whose
// jobs complete successfully.
//
// Completion arrives through UpdateJob, which calls cleanupBatchIfDone -- and that used to enter
// quiescence directly. Because the tick path only verifies a batch whose quiesce deadline is still
// unset, quiescence entered from the completion path skipped verification entirely. Verification
// therefore only ran for a batch that drained *without* a completion callback, i.e. the Prune-timeout
// case, which is the one case the other lifecycle test happens to cover. The feature was inert for
// every successful redaction and the suite was green.
func TestVerificationRunsOnNormalCompletion(t *testing.T) {
	const tenant = "t-verify-normal-completion"
	ctx, s, _ := newVerifyScheduler(t, tenant)

	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "b", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
		Query: &tempopb.TraceQLSelector{Query: `{resource.namespace = "checkout"}`},
	}))

	// One redaction job, taken to completion the way a worker would.
	job := &work.Job{
		ID:   "the-only-job",
		Type: tempopb.JobType_JOB_TYPE_REDACTION,
		JobDetail: tempopb.JobDetail{
			Tenant:    tenant,
			BatchId:   "b",
			Redaction: &tempopb.RedactionDetail{BlockId: "some-block"},
		},
	}
	require.NoError(t, s.work.AddPendingJobs([]*work.Job{job}))
	dequeued := s.work.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION)
	require.NotNil(t, dequeued)
	dequeued.SetWorkerID("w1")
	require.NoError(t, s.work.AddJob(dequeued))
	s.work.StartJob(dequeued.ID)
	s.work.CompleteJob(dequeued.ID)

	// This is what UpdateJob invokes once the last job reports success.
	s.cleanupBatchIfDone(ctx, tenant)

	state, ok := s.work.RedactionVerifyState(tenant)
	require.True(t, ok, "the batch must still exist -- it has not been verified yet")
	require.Equal(t, int32(1), state.VerifyRounds,
		"a batch completing normally must be verified, not quiesced unverified")

	quiesceUntil, _, _, ok := s.work.BatchQuiescenceState(tenant)
	require.True(t, ok)
	require.Zero(t, quiesceUntil,
		"quiescence must not start while verification is outstanding, or teardown races the scan")
	require.True(t, s.work.HasJobsForTenant(tenant, tempopb.JobType_JOB_TYPE_REDACTION),
		"the verification pass leaves scan jobs outstanding")
}

// completeVerifyJob drives a verify job through the production completion path -- UpdateJob, not
// work.CompleteJob -- because the two diverge: UpdateJob is what routes a verify result and what
// calls cleanupBatchIfDone. A test that shortcuts it cannot see either.
func completeVerifyJob(ctx context.Context, t *testing.T, s *BackendScheduler, tracesFound int32) string {
	t.Helper()
	j := s.work.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION)
	require.NotNil(t, j, "expected a pending verification job")
	require.True(t, j.JobDetail.GetRedaction().GetVerify())
	j.SetWorkerID("w1")
	require.NoError(t, s.work.AddJob(j))
	s.work.StartJob(j.ID)

	_, err := s.UpdateJob(ctx, &tempopb.UpdateJobStatusRequest{
		JobId:     j.ID,
		Status:    tempopb.JobStatus_JOB_STATUS_SUCCEEDED,
		Redaction: &tempopb.RedactionResult{TracesFound: tracesFound},
	})
	require.NoError(t, err)
	return j.JobDetail.GetRedaction().GetBlockId()
}

// TestVerifyResultEnqueuesRepairOnAMatch covers the half of the feature that acts on a finding: a
// verification scan reporting a match must queue a rewrite for that block and mark the pass dirty,
// so the batch cannot quiesce on the strength of the pass that found the gap.
func TestVerifyResultEnqueuesRepairOnAMatch(t *testing.T) {
	const tenant = "t-verify-repair"
	ctx, s, _ := newVerifyScheduler(t, tenant)

	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "b", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
		Query: &tempopb.TraceQLSelector{Query: `{resource.namespace = "checkout"}`},
	}))
	require.True(t, s.advanceVerification(ctx, tenant))

	// The window the pass derived, which the repair must inherit.
	var verifyEnd int64
	for _, j := range s.work.ListAllPendingJobs() {
		if j.JobDetail.GetRedaction().GetVerify() {
			verifyEnd = j.JobDetail.GetRedaction().GetEndTimeUnixNano()
			break
		}
	}
	require.NotZero(t, verifyEnd, "premise: the pass bounds its scan")

	blockID := completeVerifyJob(ctx, t, s, 3)

	state, ok := s.work.RedactionVerifyState(tenant)
	require.True(t, ok)
	require.False(t, state.Verified,
		"a pass that found a match is not clean; quiescing on it would report a redaction that missed a block")

	var repairs []string
	for _, j := range s.work.ListAllPendingJobs() {
		if !j.JobDetail.GetRedaction().GetVerify() {
			repairs = append(repairs, j.JobDetail.GetRedaction().GetBlockId())
		}
	}
	require.Contains(t, repairs, blockID, "the block that still matched must be queued for rewrite")

	// The repair must be scoped by the window the scan that found the match ran under. Inheriting the
	// batch's instead would be unbounded for a batch submitted without a window, and the rewrite would
	// delete traces ingested after the request.
	for _, j := range s.work.ListAllPendingJobs() {
		if j.JobDetail.GetRedaction().GetVerify() || j.JobDetail.GetRedaction().GetBlockId() != blockID {
			continue
		}
		require.Equal(t, verifyEnd, j.JobDetail.GetRedaction().GetEndTimeUnixNano(),
			"the repair carries the verification window, not the batch's")
	}
}

// TestVerifyResultCleanPassQuiesces is the converse: a pass finding nothing lets the batch settle,
// and must not queue a rewrite for a block that came back clean.
func TestVerifyResultCleanPassQuiesces(t *testing.T) {
	const tenant = "t-verify-clean-pass"
	ctx, s, _ := newVerifyScheduler(t, tenant)

	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "b", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
		Query: &tempopb.TraceQLSelector{Query: `{resource.namespace = "checkout"}`},
	}))
	require.True(t, s.advanceVerification(ctx, tenant))

	// Drain the whole pass with no findings.
	for s.work.HasJobsForTenant(tenant, tempopb.JobType_JOB_TYPE_REDACTION) {
		completeVerifyJob(ctx, t, s, 0)
	}

	state, ok := s.work.RedactionVerifyState(tenant)
	require.True(t, ok)
	require.True(t, state.Verified, "a pass that found nothing is clean")

	for _, j := range s.work.ListAllPendingJobs() {
		require.True(t, j.JobDetail.GetRedaction().GetVerify(), "a clean pass must queue no rewrites")
	}

	quiesceUntil, _, _, ok := s.work.BatchQuiescenceState(tenant)
	require.True(t, ok)
	require.NotZero(t, quiesceUntil, "the last clean result settles the batch into quiescence")
}

// TestVerifyJobFailureIsNotACleanPass pins that a scan which never ran cannot be read as clean. A
// failed verify job means its blocks were not looked at, and Prune reaching DeadJobTimeout is the
// most likely way that happens.
func TestVerifyJobFailureIsNotACleanPass(t *testing.T) {
	const tenant = "t-verify-failed"
	ctx, s, _ := newVerifyScheduler(t, tenant)

	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "b", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
		Query: &tempopb.TraceQLSelector{Query: `{resource.namespace = "checkout"}`},
	}))
	require.True(t, s.advanceVerification(ctx, tenant))

	state, _ := s.work.RedactionVerifyState(tenant)
	require.True(t, state.Verified, "premise: the pass starts optimistically clean")

	j := s.work.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION)
	require.NotNil(t, j)
	j.SetWorkerID("w1")
	require.NoError(t, s.work.AddJob(j))
	s.work.StartJob(j.ID)
	_, err := s.UpdateJob(ctx, &tempopb.UpdateJobStatusRequest{
		JobId:  j.ID,
		Status: tempopb.JobStatus_JOB_STATUS_FAILED,
		Error:  "worker died mid-scan",
	})
	require.NoError(t, err)

	state, ok := s.work.RedactionVerifyState(tenant)
	require.True(t, ok)
	require.False(t, state.Verified,
		"blocks that were never scanned cannot count as verified")
}

// TestReactivatedBatchLosesItsCleanVerdict covers the re-activation branch of advanceQuiescence.
//
// A batch that verified clean and entered quiescence can gain new rewrite work -- a rescan resolving,
// or a retried job. Those blocks have not been scanned, so the earlier clean verdict does not cover
// them. Leaving it set would let the batch quiesce a second time without verifying any of them.
func TestReactivatedBatchLosesItsCleanVerdict(t *testing.T) {
	const tenant = "t-verify-reactivated"
	ctx, s, _ := newVerifyScheduler(t, tenant)

	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "b", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
		Verified: true, VerifyRounds: 1,
		QuiesceUntilUnixNano: time.Now().Add(time.Hour).UnixNano(),
	}))

	// New rewrite work arrives while the batch is quiescing.
	require.NoError(t, s.work.AddPendingJobs([]*work.Job{{
		ID:   "late-rewrite",
		Type: tempopb.JobType_JOB_TYPE_REDACTION,
		JobDetail: tempopb.JobDetail{
			Tenant:    tenant,
			BatchId:   "b",
			Redaction: &tempopb.RedactionDetail{BlockId: "block-nobody-scanned"},
		},
	}}))

	require.True(t, s.advanceQuiescence(ctx, tenant), "re-activation changes batch state")

	state, ok := s.work.RedactionVerifyState(tenant)
	require.True(t, ok)
	require.False(t, state.Verified,
		"a batch that gained unscanned work must not keep an earlier clean verdict")

	quiesceUntil, _, _, ok := s.work.BatchQuiescenceState(tenant)
	require.True(t, ok)
	require.Zero(t, quiesceUntil, "re-activation leaves quiescence")
}

// TestVerificationDefersWhenEveryCandidateIsBusy covers the deferred branch. A block held by another
// job cannot be scanned, and a block mid-compaction is the likeliest origin of the uncovered block
// this pass exists to find -- so a pass that checked nothing must not let the batch tear down.
func TestVerificationDefersWhenEveryCandidateIsBusy(t *testing.T) {
	const tenant = "t-verify-all-busy"
	ctx, s, _ := newVerifyScheduler(t, tenant)

	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "b", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
		Query: &tempopb.TraceQLSelector{Query: `{resource.namespace = "checkout"}`},
	}))

	// Hold every block the pass would scan.
	candidates, _ := s.verificationJobs(tenant, mustVerifyState(t, s, tenant))
	require.NotEmpty(t, candidates, "premise: there are candidates to hold")
	for i, c := range candidates {
		s.work.RegisterJob(&work.Job{
			ID:   fmt.Sprintf("compaction-%d", i),
			Type: tempopb.JobType_JOB_TYPE_COMPACTION,
			JobDetail: tempopb.JobDetail{
				Tenant:     tenant,
				Compaction: &tempopb.CompactionDetail{Input: []string{c.JobDetail.GetRedaction().GetBlockId()}},
			},
		})
	}

	require.True(t, s.advanceVerification(ctx, tenant),
		"a pass that checked nothing must report outstanding work, not let the batch quiesce")

	state, ok := s.work.RedactionVerifyState(tenant)
	require.True(t, ok)
	require.False(t, state.Verified, "nothing was scanned, so nothing is verified")
	require.Zero(t, state.VerifyRounds, "a pass that ran no jobs must not consume a round")
}

// TestRepairRefusedWhenTheBatchIsGone covers the identity guard. A verify job can outlive its batch:
// Prune fails it, the batch quiesces and is removed, and the worker still reports a match. Queueing
// then would either be dropped in Next(), losing a confirmed surviving trace, or be matched to a
// batch submitted since and rewritten under that batch's scope.
func TestRepairRefusedWhenTheBatchIsGone(t *testing.T) {
	const tenant = "t-verify-orphaned"
	ctx, s, _ := newVerifyScheduler(t, tenant)

	// A batch exists, but under a different ID than the one the stale job names.
	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "the-new-batch", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
	}))

	// A verify job from the batch that has since been removed.
	stale := &work.Job{
		ID:   "stale-verify-job",
		Type: tempopb.JobType_JOB_TYPE_REDACTION,
		JobDetail: tempopb.JobDetail{
			Tenant:  tenant,
			BatchId: "the-batch-that-is-gone",
			Redaction: &tempopb.RedactionDetail{
				BlockId: "some-block",
				Verify:  true,
			},
		},
	}
	s.enqueueRedactionForVerifiedBlock(ctx, stale)

	require.Empty(t, s.work.ListAllPendingJobs(),
		"a repair must not be queued against a batch that is not the one it came from")
}

// TestAdvanceQuiescenceWithNoBatch and the dry-run path round out the sweep's branches.
func TestAdvanceQuiescenceEdges(t *testing.T) {
	ctx, s, _ := newVerifyScheduler(t, "t-quiesce-edges")

	require.False(t, s.advanceQuiescence(ctx, "tenant-without-a-batch"),
		"a tenant with no batch is not a state change")

	const dryTenant = "t-quiesce-dryrun"
	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "b", TenantId: dryTenant, CreatedAtUnixNano: time.Now().UnixNano(),
		Mode: tempopb.RedactionMode_REDACTION_MODE_DRY_RUN,
	}))
	require.True(t, s.advanceQuiescence(ctx, dryTenant))
	require.Nil(t, s.work.GetBatch(dryTenant),
		"a drained dry-run wrote nothing and never blocked compaction, so it is removed outright")
}

// TestVerificationSkipsBlocksAlreadyCovered pins the filter that makes a pass affordable. A block
// whose redaction job succeeded was either rewritten or found clean, so re-scanning it costs a full
// block read and tells us nothing. Without this a pass re-scans every in-window block, making
// verification cost about as much as the redaction it is verifying, once per round.
func TestVerificationSkipsBlocksAlreadyCovered(t *testing.T) {
	const tenant = "t-verify-covered"
	_, s, _ := newVerifyScheduler(t, tenant)

	state := work.RedactionVerifyState{BatchID: "b", CreatedAtUnixNano: time.Now().UnixNano()}
	all, _ := s.verificationJobs(tenant, state)
	require.Len(t, all, 3, "premise: three in-window blocks")

	// One block already has a succeeded redaction job for this batch.
	coveredBlock := all[0].JobDetail.GetRedaction().GetBlockId()
	done := &work.Job{
		ID:   "already-redacted",
		Type: tempopb.JobType_JOB_TYPE_REDACTION,
		JobDetail: tempopb.JobDetail{
			Tenant:    tenant,
			BatchId:   "b",
			Redaction: &tempopb.RedactionDetail{BlockId: coveredBlock},
		},
	}
	done.SetWorkerID("w1")
	require.NoError(t, s.work.AddJob(done))
	s.work.StartJob(done.ID)
	s.work.CompleteJob(done.ID)

	remaining, _ := s.verificationJobs(tenant, state)
	require.Len(t, remaining, 2, "a block with a succeeded job is not re-scanned")
	for _, j := range remaining {
		require.NotEqual(t, coveredBlock, j.JobDetail.GetRedaction().GetBlockId())
	}
}

// TestVerificationRescansAFailedBlock is the other half: a job that FAILED did not scan its block, so
// the pass must still look at it. Treating failure as coverage would skip exactly the blocks least
// likely to have been redacted.
func TestVerificationRescansAFailedBlock(t *testing.T) {
	const tenant = "t-verify-failed-not-covered"
	_, s, _ := newVerifyScheduler(t, tenant)

	state := work.RedactionVerifyState{BatchID: "b", CreatedAtUnixNano: time.Now().UnixNano()}
	all, _ := s.verificationJobs(tenant, state)
	require.Len(t, all, 3)

	failedBlock := all[0].JobDetail.GetRedaction().GetBlockId()
	failed := &work.Job{
		ID:   "redaction-that-failed",
		Type: tempopb.JobType_JOB_TYPE_REDACTION,
		JobDetail: tempopb.JobDetail{
			Tenant:    tenant,
			BatchId:   "b",
			Redaction: &tempopb.RedactionDetail{BlockId: failedBlock},
		},
	}
	failed.SetWorkerID("w1")
	require.NoError(t, s.work.AddJob(failed))
	s.work.StartJob(failed.ID)
	s.work.FailJob(failed.ID)

	remaining, _ := s.verificationJobs(tenant, state)
	require.Len(t, remaining, 3, "a failed job leaves its block unscanned, so it stays a candidate")
}

// TestVerificationCoverageIsScopedToTheBatch guards against a previous batch's completed work
// suppressing this batch's pass.
func TestVerificationCoverageIsScopedToTheBatch(t *testing.T) {
	const tenant = "t-verify-covered-other-batch"
	_, s, _ := newVerifyScheduler(t, tenant)

	state := work.RedactionVerifyState{BatchID: "b", CreatedAtUnixNano: time.Now().UnixNano()}
	all, _ := s.verificationJobs(tenant, state)
	require.Len(t, all, 3)

	other := &work.Job{
		ID:   "older-batch-job",
		Type: tempopb.JobType_JOB_TYPE_REDACTION,
		JobDetail: tempopb.JobDetail{
			Tenant:    tenant,
			BatchId:   "a-previous-batch",
			Redaction: &tempopb.RedactionDetail{BlockId: all[0].JobDetail.GetRedaction().GetBlockId()},
		},
	}
	other.SetWorkerID("w1")
	require.NoError(t, s.work.AddJob(other))
	s.work.StartJob(other.ID)
	s.work.CompleteJob(other.ID)

	remaining, _ := s.verificationJobs(tenant, state)
	require.Len(t, remaining, 3, "another batch's completed work does not cover this batch's blocks")
}

// TestVerificationScanIsNotCoverage separates "this block was scanned" from "this block was
// rewritten". A scan that finds matches succeeds and queues a repair; if that repair fails, the batch
// is dirty and passes again -- and if the succeeded scan counted as coverage, that pass would skip the
// one block known to still hold matches and report the batch clean.
func TestVerificationScanIsNotCoverage(t *testing.T) {
	const tenant = "t-verify-scan-not-coverage"
	_, s, _ := newVerifyScheduler(t, tenant)

	state := work.RedactionVerifyState{BatchID: "b", CreatedAtUnixNano: time.Now().UnixNano()}
	all, _ := s.verificationJobs(tenant, state)
	require.Len(t, all, 3, "premise: three in-window blocks")

	// A verification scan for this batch on one of those blocks, succeeded.
	scanned := all[0].JobDetail.GetRedaction().GetBlockId()
	scan := &work.Job{
		ID:   "a-succeeded-scan",
		Type: tempopb.JobType_JOB_TYPE_REDACTION,
		JobDetail: tempopb.JobDetail{
			Tenant:    tenant,
			BatchId:   "b",
			Redaction: &tempopb.RedactionDetail{BlockId: scanned, Verify: true},
		},
	}
	scan.SetWorkerID("w1")
	require.NoError(t, s.work.AddJob(scan))
	s.work.StartJob(scan.ID)
	s.work.CompleteJob(scan.ID)

	remaining, _ := s.verificationJobs(tenant, state)
	require.Len(t, remaining, 3, "a succeeded scan is not a rewrite and must not suppress the block")
	blocks := make([]string, 0, len(remaining))
	for _, j := range remaining {
		blocks = append(blocks, j.JobDetail.GetRedaction().GetBlockId())
	}
	require.Contains(t, blocks, scanned)
}

// TestPassThatDeferredBlocksIsNotClean pins that a partly-run pass is not a clean verdict.
//
// The deferred-only branch already refuses to settle a pass that checked nothing. A pass that
// enqueued some jobs and skipped others is no more complete: those blocks were not scanned, so
// marking the batch clean lets it quiesce with them unchecked. Left dirty, the next drain re-derives
// and picks them up, bounded by maxVerifyRounds.
func TestPassThatDeferredBlocksIsNotClean(t *testing.T) {
	const tenant = "t-verify-partial"
	ctx, s, _ := newVerifyScheduler(t, tenant)

	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "b", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
		Query: &tempopb.TraceQLSelector{Query: `{resource.namespace = "checkout"}`},
	}))

	// Hold one of the three candidates with a compaction so the pass both enqueues and defers.
	candidates, _ := s.verificationJobs(tenant, mustVerifyState(t, s, tenant))
	require.Len(t, candidates, 3, "premise: three in-window blocks")
	s.work.RegisterJob(&work.Job{
		ID:   "compaction-holding-a-candidate",
		Type: tempopb.JobType_JOB_TYPE_COMPACTION,
		JobDetail: tempopb.JobDetail{
			Tenant:     tenant,
			Compaction: &tempopb.CompactionDetail{Input: []string{candidates[0].JobDetail.GetRedaction().GetBlockId()}},
		},
	})

	jobs, deferred := s.verificationJobs(tenant, mustVerifyState(t, s, tenant))
	require.Len(t, jobs, 2, "premise: the pass enqueues the two free blocks")
	require.Equal(t, 1, deferred, "premise: and defers the busy one")

	require.True(t, s.advanceVerification(ctx, tenant), "a pass with work outstanding reports outstanding")

	state := mustVerifyState(t, s, tenant)
	require.False(t, state.Verified,
		"a pass that skipped a busy block has not checked it, so the batch is not clean")
}

// TestVerifySuccessWithNoResultIsNotClean covers the scan that answers nothing. A worker reporting
// SUCCEEDED without a redaction result has said nothing about its block, so the pass cannot be read
// as clean -- the batch must verify again rather than quiescing on an unanswered scan.
func TestVerifySuccessWithNoResultIsNotClean(t *testing.T) {
	const tenant = "t-verify-no-result"
	ctx, s, _ := newVerifyScheduler(t, tenant)

	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "b", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
		Query: &tempopb.TraceQLSelector{Query: `{resource.namespace = "checkout"}`},
	}))
	require.True(t, s.advanceVerification(ctx, tenant))
	require.True(t, mustVerifyState(t, s, tenant).Verified, "premise: the pass starts optimistically clean")

	j := s.work.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION)
	require.NotNil(t, j)
	require.True(t, j.JobDetail.GetRedaction().GetVerify())
	j.SetWorkerID("w1")
	require.NoError(t, s.work.AddJob(j))
	s.work.StartJob(j.ID)

	// SUCCEEDED with no Redaction result at all.
	_, err := s.UpdateJob(ctx, &tempopb.UpdateJobStatusRequest{
		JobId:  j.ID,
		Status: tempopb.JobStatus_JOB_STATUS_SUCCEEDED,
	})
	require.NoError(t, err)

	require.False(t, mustVerifyState(t, s, tenant).Verified,
		"a scan that reported no result cannot leave the batch verified")
}

// TestFlushFailureLeavesBatchUnverified covers the durability edge on a destructive path.
//
// The scan jobs are published in memory before they are persisted, and the caller writes the batch
// manifest once per tick afterwards. If the job flush fails, the manifest would otherwise record
// "round consumed, batch clean" while no jobs exist on disk -- so a restart reads a redaction that
// verified cleanly and never scanned anything. Recording the batch dirty costs an extra pass in the
// surviving case, which is the safe direction.
func TestFlushFailureLeavesBatchUnverified(t *testing.T) {
	const tenant = "t-verify-flush-fails"
	ctx, s, _ := newVerifyScheduler(t, tenant)

	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "b", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
		Query: &tempopb.TraceQLSelector{Query: `{resource.namespace = "checkout"}`},
	}))

	// FlushToLocal starts with MkdirAll, so pointing the work path at a regular file makes every
	// flush fail with ENOTDIR.
	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o600))
	s.cfg.LocalWorkPath = blocker
	require.Error(t, s.work.FlushToLocal(ctx, s.cfg.LocalWorkPath, nil), "premise: flushing must fail")

	require.True(t, s.advanceVerification(ctx, tenant), "the pass still runs; its jobs are live in memory")

	state := mustVerifyState(t, s, tenant)
	require.False(t, state.Verified,
		"a pass whose jobs never reached disk must not leave the batch recorded clean")
}

// TestTimedOutScanLeavesBatchUnverified covers the failure that never reports itself.
//
// Prune force-fails a job past DeadJobTimeout by calling j.Fail() directly, so it bypasses UpdateJob
// -- and UpdateJob's failure path is what records that a scan did not run. Without the Prune
// reaction, a pass whose only scan was killed by the timeout keeps the optimistic clean mark it was
// launched with, and the batch quiesces on a block nothing ever looked at.
func TestTimedOutScanLeavesBatchUnverified(t *testing.T) {
	const tenant = "t-verify-timeout"
	ctx, s, _ := newVerifyScheduler(t, tenant)

	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "b", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
		Query: &tempopb.TraceQLSelector{Query: `{resource.namespace = "checkout"}`},
	}))
	require.True(t, s.advanceVerification(ctx, tenant))
	require.True(t, mustVerifyState(t, s, tenant).Verified, "premise: the pass starts optimistically clean")

	// Take a scan job to RUNNING and age it past the dead-job timeout.
	j := s.work.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION)
	require.NotNil(t, j)
	require.True(t, j.JobDetail.GetRedaction().GetVerify())
	j.SetWorkerID("w1")
	require.NoError(t, s.work.AddJob(j))
	s.work.StartJob(j.ID)
	// Back-date past the default 24h DeadJobTimeout, as the Prune test in this package does.
	s.work.GetJob(j.ID).StartTime = time.Now().Add(-25 * time.Hour)

	timedOut := s.work.Prune(ctx)
	require.Len(t, timedOut, 1, "premise: Prune must report the scan it force-failed")

	s.releaseVerificationForTimedOutJobs(timedOut)

	require.False(t, mustVerifyState(t, s, tenant).Verified,
		"a scan killed by the timeout did not check its block, so the batch is not verified")
}

// TestRetriedVerifyResultQueuesOneRepair covers the retried RPC. UpdateJob is retried by the worker
// after an error or a lost response, and the result handler runs again each time with a fresh job
// UUID against an AddPendingJobs that deduplicates nothing -- so without a per-block guard a single
// confirmed gap becomes two concurrent rewrites of the same block.
func TestRetriedVerifyResultQueuesOneRepair(t *testing.T) {
	const tenant = "t-verify-retry"
	ctx, s, _ := newVerifyScheduler(t, tenant)

	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "b", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
		Query: &tempopb.TraceQLSelector{Query: `{resource.namespace = "checkout"}`},
	}))
	require.True(t, s.advanceVerification(ctx, tenant))

	j := s.work.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION)
	require.NotNil(t, j)
	require.True(t, j.JobDetail.GetRedaction().GetVerify())
	j.SetWorkerID("w1")
	require.NoError(t, s.work.AddJob(j))
	s.work.StartJob(j.ID)

	result := &tempopb.UpdateJobStatusRequest{
		JobId:     j.ID,
		Status:    tempopb.JobStatus_JOB_STATUS_SUCCEEDED,
		Redaction: &tempopb.RedactionResult{TracesFound: 3},
	}
	// The same result delivered twice, as a retry after a lost response.
	_, err := s.UpdateJob(ctx, result)
	require.NoError(t, err)
	_, err = s.UpdateJob(ctx, result)
	require.NoError(t, err)

	repairs := 0
	for {
		q := s.work.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION)
		if q == nil {
			break
		}
		if !q.JobDetail.GetRedaction().GetVerify() {
			repairs++
		}
	}
	require.Equal(t, 1, repairs, "a retried result must not queue a second rewrite of the same block")
}

// TestSettleRechecksTheBatchIsStillDrained covers the second caller into the settle path.
//
// Both the completion path and the maintenance tick settle a drained batch, and each arrives having
// checked liveness before taking the lock. If the first caller launches a verification pass, those
// scans make the batch active again -- so the second must not act on its stale check and quiesce a
// batch that is mid-verification.
func TestSettleRechecksTheBatchIsStillDrained(t *testing.T) {
	const tenant = "t-verify-settle-recheck"
	ctx, s, _ := newVerifyScheduler(t, tenant)

	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "b", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
		Query: &tempopb.TraceQLSelector{Query: `{resource.namespace = "checkout"}`},
	}))

	// First caller: launches a pass, leaving scan jobs outstanding.
	s.settleDrainedBatch(ctx, tenant)
	require.True(t, s.work.HasJobsForTenant(tenant, tempopb.JobType_JOB_TYPE_REDACTION),
		"premise: the pass left scan jobs outstanding, so the batch is active again")
	rounds := mustVerifyState(t, s, tenant).VerifyRounds
	require.Equal(t, int32(1), rounds, "premise: one pass ran")

	// Second caller, arriving with a check made before that pass existed.
	s.settleDrainedBatch(ctx, tenant)

	quiesceUntil, _, _, ok := s.work.BatchQuiescenceState(tenant)
	require.True(t, ok)
	require.Zero(t, quiesceUntil, "a batch mid-verification must not be quiesced")
	require.Equal(t, rounds, mustVerifyState(t, s, tenant).VerifyRounds,
		"the second settle must not consume another round")
}
