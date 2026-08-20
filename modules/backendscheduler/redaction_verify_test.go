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

	require.NotEmpty(t, s.verificationJobs(tenant, mustVerifyState(t, s, tenant)),
		"premise: there are in-scope blocks, so only the guard can stop the pass")

	require.False(t, s.runVerification(ctx, tenant), "a verified batch must not be re-verified")
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

	require.NotEmpty(t, s.verificationJobs(tenant, mustVerifyState(t, s, tenant)),
		"premise: there are in-scope blocks, so only the round limit can stop the pass")

	require.False(t, s.runVerification(ctx, tenant),
		"at the round limit the batch is released so compaction can resume")
	require.False(t, s.work.HasJobsForTenant(tenant, tempopb.JobType_JOB_TYPE_REDACTION))
}

// TestRunVerificationWithNoBatch guards the sweep against a batch removed between the drain check
// and the pass.
func TestRunVerificationWithNoBatch(t *testing.T) {
	ctx, s := newQuiescenceScheduler(t)
	require.False(t, s.runVerification(ctx, "t-absent"))
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

	jobs := s.verificationJobs(tenant, work.RedactionVerifyState{
		BatchID:           "b",
		Query:             `{resource.namespace = "checkout"}`,
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

	jobs := s.verificationJobs(tenant, work.RedactionVerifyState{
		BatchID:           "b",
		Query:             `{resource.namespace = "checkout"}`,
		StartTimeUnixNano: windowStart,
		EndTimeUnixNano:   windowEnd,
	})

	require.Len(t, jobs, 1, "only the block overlapping the window is scanned")
}

// TestVerificationJobsSkipBusyBlocks pins that a block already held by another job is left alone.
// Enqueueing a second job for a block mid-compaction would race it, and the next pass re-derives
// candidates so the block is re-checked rather than dropped.
func TestVerificationJobsSkipBusyBlocks(t *testing.T) {
	const tenant = "t-verify-busy"
	_, s, _ := newVerifyScheduler(t, tenant)

	state := work.RedactionVerifyState{
		BatchID:           "b",
		Query:             `{resource.namespace = "checkout"}`,
		CreatedAtUnixNano: time.Now().UnixNano(),
	}
	all := s.verificationJobs(tenant, state)
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

	remaining := s.verificationJobs(tenant, state)
	require.Len(t, remaining, 2, "the busy block is skipped")
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
