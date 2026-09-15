package backendscheduler

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
)

func newAuditScheduler(t *testing.T, tenant string) (context.Context, *BackendScheduler, func(int) time.Time) {
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
	// All three must be polled before any filter is exercised, or a count assertion can pass
	// without the filter ever running.
	require.Eventually(t, func() bool { return len(store.BlockMetas(tenant)) == 3 },
		5*time.Second, 50*time.Millisecond, "all three blocks must be polled")

	limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	s, err := New(cfg, store, limits, rr, ww)
	require.NoError(t, err)
	return ctx, s, day
}

func addQueryBatch(t *testing.T, s *BackendScheduler, tenant, batchID string) {
	t.Helper()
	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: batchID, TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
		Query: &tempopb.TraceQLSelector{Query: `{resource.namespace = "checkout"}`},
	}))
}

// drainPending pops every pending redaction job and splits it by kind, which is how a test tells an
// audit scan from a rewrite.
func drainPending(s *BackendScheduler) (scans, rewrites []*work.Job) {
	for {
		j := s.work.NextPendingJob(tempopb.JobType_JOB_TYPE_REDACTION)
		if j == nil {
			return scans, rewrites
		}
		if j.JobDetail.GetRedaction().GetVerify() {
			scans = append(scans, j)
			continue
		}
		rewrites = append(rewrites, j)
	}
}

// TestAuditOnlyEnqueuesScans is the safety property: the audit must never produce a job that
// rewrites. Next() overwrites a job's mode from the batch, so `verify` is the only field that
// survives dispatch to distinguish a scan from a rewrite.
func TestAuditOnlyEnqueuesScans(t *testing.T) {
	const tenant = "t-audit-scans-only"
	_, s, _ := newAuditScheduler(t, tenant)

	state := work.RedactionVerifyState{BatchID: "b", CreatedAtUnixNano: time.Now().UnixNano()}
	jobs := s.auditJobs(tenant, state)
	require.Len(t, jobs, 3, "premise: three in-window blocks")

	for _, j := range jobs {
		require.True(t, j.JobDetail.GetRedaction().GetVerify(), "every audit job must be read-only")
		require.Equal(t, tempopb.JobType_JOB_TYPE_REDACTION, j.Type)
		require.Equal(t, "b", j.JobDetail.GetBatchId(), "a scan carries its batch, so a stale one is dropped at dispatch")
	}
}

// TestAuditRespectsTheWindow keeps the audit inside what the operator asked to remove. A scan wider
// than the request would report matches the redaction was never meant to touch, which reads as an
// incomplete redaction.
func TestAuditRespectsTheWindow(t *testing.T) {
	const tenant = "t-audit-window"
	_, s, day := newAuditScheduler(t, tenant)

	state := work.RedactionVerifyState{
		BatchID:           "b",
		CreatedAtUnixNano: time.Now().UnixNano(),
		StartTimeUnixNano: day(4).UnixNano(),
		EndTimeUnixNano:   day(5).UnixNano(),
	}

	jobs := s.auditJobs(tenant, state)
	require.Len(t, jobs, 1, "only the block overlapping the window is scanned")
	require.Equal(t, state.StartTimeUnixNano, jobs[0].JobDetail.GetRedaction().GetStartTimeUnixNano(),
		"the resolved window travels on the job so the scan itself is bounded")
	require.Equal(t, state.EndTimeUnixNano, jobs[0].JobDetail.GetRedaction().GetEndTimeUnixNano())
}

// TestAuditSkipsBusyBlocks covers the block-exclusivity filter. A second job on a block already in
// flight would race it.
func TestAuditSkipsBusyBlocks(t *testing.T) {
	const tenant = "t-audit-busy"
	_, s, _ := newAuditScheduler(t, tenant)

	state := work.RedactionVerifyState{BatchID: "b", CreatedAtUnixNano: time.Now().UnixNano()}
	all := s.auditJobs(tenant, state)
	require.Len(t, all, 3)

	busyBlock := all[0].JobDetail.GetRedaction().GetBlockId()
	s.work.RegisterJob(&work.Job{
		ID:   "compaction-holding-a-block",
		Type: tempopb.JobType_JOB_TYPE_COMPACTION,
		JobDetail: tempopb.JobDetail{
			Tenant:     tenant,
			Compaction: &tempopb.CompactionDetail{Input: []string{busyBlock}},
		},
	})

	remaining := s.auditJobs(tenant, state)
	require.Len(t, remaining, 2, "the busy block is skipped")
	for _, j := range remaining {
		require.NotEqual(t, busyBlock, j.JobDetail.GetRedaction().GetBlockId())
	}
}

// TestAuditSkipsBlocksTheBatchRewrote is the coverage filter: a completed rewrite is exactly the
// coverage the audit is looking for, so re-scanning it buys nothing.
func TestAuditSkipsBlocksTheBatchRewrote(t *testing.T) {
	const tenant = "t-audit-covered"
	_, s, _ := newAuditScheduler(t, tenant)

	state := work.RedactionVerifyState{BatchID: "b", CreatedAtUnixNano: time.Now().UnixNano()}
	all := s.auditJobs(tenant, state)
	require.Len(t, all, 3, "premise: three in-window blocks")

	covered := all[0].JobDetail.GetRedaction().GetBlockId()
	done := &work.Job{
		ID:   "already-rewritten",
		Type: tempopb.JobType_JOB_TYPE_REDACTION,
		JobDetail: tempopb.JobDetail{
			Tenant:    tenant,
			BatchId:   "b",
			Redaction: &tempopb.RedactionDetail{BlockId: covered},
		},
	}
	done.SetWorkerID("w1")
	require.NoError(t, s.work.AddJob(done))
	s.work.StartJob(done.ID)
	s.work.CompleteJob(done.ID)

	remaining := s.auditJobs(tenant, state)
	require.Len(t, remaining, 2, "a block with a completed rewrite is not re-scanned")
	for _, j := range remaining {
		require.NotEqual(t, covered, j.JobDetail.GetRedaction().GetBlockId())
	}
}

// TestAuditScansABlockWhoseRewriteFailed is the other half, and the main thing the audit is for: a
// FAILED job did not rewrite its block, so treating failure as coverage would skip exactly the
// blocks least likely to have been redacted.
func TestAuditScansABlockWhoseRewriteFailed(t *testing.T) {
	const tenant = "t-audit-failed"
	_, s, _ := newAuditScheduler(t, tenant)

	state := work.RedactionVerifyState{BatchID: "b", CreatedAtUnixNano: time.Now().UnixNano()}
	all := s.auditJobs(tenant, state)
	require.Len(t, all, 3)

	failedBlock := all[0].JobDetail.GetRedaction().GetBlockId()
	j := &work.Job{
		ID:   "rewrite-that-failed",
		Type: tempopb.JobType_JOB_TYPE_REDACTION,
		JobDetail: tempopb.JobDetail{
			Tenant:    tenant,
			BatchId:   "b",
			Redaction: &tempopb.RedactionDetail{BlockId: failedBlock},
		},
	}
	j.SetWorkerID("w1")
	require.NoError(t, s.work.AddJob(j))
	s.work.StartJob(j.ID)
	s.work.FailJob(j.ID)

	blocks := make([]string, 0, 3)
	for _, job := range s.auditJobs(tenant, state) {
		blocks = append(blocks, job.JobDetail.GetRedaction().GetBlockId())
	}
	require.Len(t, blocks, 3, "a failed rewrite is not coverage")
	require.Contains(t, blocks, failedBlock)
}

// TestAuditScanIsNotCoverage separates "this block was scanned" from "this block was rewritten". A
// scan reads and never writes, so it can never stand in for a rewrite.
func TestAuditScanIsNotCoverage(t *testing.T) {
	const tenant = "t-audit-scan-not-coverage"
	_, s, _ := newAuditScheduler(t, tenant)

	state := work.RedactionVerifyState{BatchID: "b", CreatedAtUnixNano: time.Now().UnixNano()}
	all := s.auditJobs(tenant, state)
	require.Len(t, all, 3)

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

	blocks := make([]string, 0, 3)
	for _, j := range s.auditJobs(tenant, state) {
		blocks = append(blocks, j.JobDetail.GetRedaction().GetBlockId())
	}
	require.Len(t, blocks, 3, "a succeeded scan is not a rewrite and must not suppress the block")
	require.Contains(t, blocks, scanned)
}

// TestAuditCoverageIsScopedToTheBatch pins that another batch's rewrite does not count. Coverage is
// a claim about what *this* request removed.
func TestAuditCoverageIsScopedToTheBatch(t *testing.T) {
	const tenant = "t-audit-batch-scope"
	_, s, _ := newAuditScheduler(t, tenant)

	state := work.RedactionVerifyState{BatchID: "this-batch", CreatedAtUnixNano: time.Now().UnixNano()}
	all := s.auditJobs(tenant, state)
	require.Len(t, all, 3)

	other := &work.Job{
		ID:   "another-batches-rewrite",
		Type: tempopb.JobType_JOB_TYPE_REDACTION,
		JobDetail: tempopb.JobDetail{
			Tenant:    tenant,
			BatchId:   "some-other-batch",
			Redaction: &tempopb.RedactionDetail{BlockId: all[0].JobDetail.GetRedaction().GetBlockId()},
		},
	}
	other.SetWorkerID("w1")
	require.NoError(t, s.work.AddJob(other))
	s.work.StartJob(other.ID)
	s.work.CompleteJob(other.ID)

	require.Len(t, s.auditJobs(tenant, state), 3,
		"a different batch's rewrite says nothing about this batch's coverage")
}

// TestAuditRunsOncePerBatch pins the guard that keeps the audit from re-deriving on every tick for as
// long as the batch exists. Derived from the job records rather than a persisted flag.
func TestAuditRunsOncePerBatch(t *testing.T) {
	const tenant = "t-audit-once"
	ctx, s, _ := newAuditScheduler(t, tenant)
	addQueryBatch(t, s, tenant, "b")

	s.auditDrainedBatch(ctx, tenant)
	pending := s.work.ListAllPendingJobs()
	require.Len(t, pending, 3, "the first audit enqueues a scan per in-window block")
	for _, j := range pending {
		require.True(t, j.JobDetail.GetRedaction().GetVerify(), "an audit never enqueues a rewrite")
	}

	// Deliberately left in the pending queue rather than promoted. ListJobs covers only the active
	// and terminal maps, so a guard that consulted it alone would pass here and enqueue a second set.
	s.auditDrainedBatch(ctx, tenant)
	require.Len(t, s.work.ListAllPendingJobs(), 3,
		"a batch whose scans are still queued must not be audited again")

	// And once they are dispatched and terminal, the guard still holds.
	scans, rewrites := drainPending(s)
	require.Len(t, scans, 3)
	require.Empty(t, rewrites)
	for _, j := range scans {
		j.SetWorkerID("w1")
		require.NoError(t, s.work.AddJob(j))
		s.work.StartJob(j.ID)
		s.work.CompleteJob(j.ID)
	}

	s.auditDrainedBatch(ctx, tenant)
	again, _ := drainPending(s)
	require.Empty(t, again, "a batch whose scans have completed must not be audited again")
}

// TestAuditDoesNotGateTeardown is the load-bearing property of the narrowed feature: the audit
// reports, it does not decide. Quiescence is entered on the same tick, and the batch is removed once
// its scans drain through the ordinary path -- no verdict, no round budget.
func TestAuditDoesNotGateTeardown(t *testing.T) {
	const tenant = "t-audit-non-gating"
	ctx, s, _ := newAuditScheduler(t, tenant)
	addQueryBatch(t, s, tenant, "b")

	s.cleanupBatchIfDone(ctx, tenant)

	quiesceUntil, _, _, ok := s.work.BatchQuiescenceState(tenant)
	require.True(t, ok, "the batch is still present")
	require.NotZero(t, quiesceUntil, "quiescence is entered regardless of what the audit found")
	require.True(t, s.work.HasJobsForTenant(tenant, tempopb.JobType_JOB_TYPE_REDACTION),
		"the audit's scans are outstanding, which is what keeps the batch alive until they finish")
}

// TestAuditWindowDerivation pins the scan bounds, which decide what a scan can report.
//
// An unbounded re-scan would match data ingested after the request, which the operator never asked to
// remove. The explicit-ID selector is the exception: it applies no time bound, and RedactBlock refuses
// an ID list combined with a window, so its scans run unwindowed exactly as its rewrites did.
func TestAuditWindowDerivation(t *testing.T) {
	created := time.Now().UnixNano()
	start, end := created-1000, created-500

	for _, tc := range []struct {
		name               string
		state              work.RedactionVerifyState
		wantStart, wantEnd int64
	}{
		{
			name:      "no window falls back to everything up to submission",
			state:     work.RedactionVerifyState{CreatedAtUnixNano: created},
			wantStart: 0, wantEnd: created,
		},
		{
			name:      "a caller window is used as given",
			state:     work.RedactionVerifyState{CreatedAtUnixNano: created, StartTimeUnixNano: start, EndTimeUnixNano: end},
			wantStart: start, wantEnd: end,
		},
		{
			name:      "an explicit ID list scans unwindowed",
			state:     work.RedactionVerifyState{CreatedAtUnixNano: created, HasTraceIDs: true},
			wantStart: 0, wantEnd: 0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gotStart, gotEnd := auditWindow(tc.state)
			require.Equal(t, tc.wantStart, gotStart)
			require.Equal(t, tc.wantEnd, gotEnd)
		})
	}
}

// TestAuditWithNoBatch covers the guard for a tenant whose batch is gone. Blocks are in scope, so the
// guard is what has to stop the audit -- with an empty store it would return early via the
// nothing-to-scan path and pass with the guard deleted.
func TestAuditWithNoBatch(t *testing.T) {
	const tenant = "t-audit-no-batch"
	ctx, s, _ := newAuditScheduler(t, tenant)

	premise := s.auditJobs(tenant, work.RedactionVerifyState{BatchID: "b", CreatedAtUnixNano: time.Now().UnixNano()})
	require.Len(t, premise, 3, "premise: there are in-scope blocks, so only the guard can stop the audit")

	s.auditDrainedBatch(ctx, tenant)

	scans, rewrites := drainPending(s)
	require.Empty(t, scans, "no batch means nothing to audit")
	require.Empty(t, rewrites)
}

// TestBatchHasAuditJobsSeesPendingScans tests the guard's own contract rather than its effect.
//
// The end-to-end test above cannot distinguish the guard from auditJobs' busy-block filter: a pending
// scan makes its block busy, so a second audit enqueues nothing either way. That makes the outcome
// test blind to whether the guard works at all, which is why this asserts the guard directly. The two
// job queues are separate maps -- ListJobs covers active and terminal, ListAllPendingJobs covers
// queued -- so a guard consulting only one of them reports a batch as un-audited while its scans sit
// in the other.
func TestBatchHasAuditJobsSeesPendingScans(t *testing.T) {
	const tenant = "t-audit-guard"
	_, s, _ := newAuditScheduler(t, tenant)

	require.False(t, s.batchHasAuditJobs(tenant, "b"), "premise: nothing audited yet")

	scan := &work.Job{
		ID:   "a-queued-scan",
		Type: tempopb.JobType_JOB_TYPE_REDACTION,
		JobDetail: tempopb.JobDetail{
			Tenant:    tenant,
			BatchId:   "b",
			Redaction: &tempopb.RedactionDetail{BlockId: "blk", Verify: true},
		},
	}
	require.NoError(t, s.work.AddPendingJobs([]*work.Job{scan}))
	require.Empty(t, s.work.ListJobs(), "premise: the scan is queued only, so the active map is empty")

	require.True(t, s.batchHasAuditJobs(tenant, "b"),
		"a scan in the pending queue means this batch has been audited")

	// Scoping still holds: another batch's scan says nothing about this one.
	require.False(t, s.batchHasAuditJobs(tenant, "some-other-batch"))
}
