package backendscheduler

import (
	"context"
	"flag"
	"fmt"
	"testing"
	"time"

	"github.com/gogo/status"
	"github.com/grafana/dskit/user"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
)

func TestBlockOverlapsWindow(t *testing.T) {
	ts := func(s string) int64 {
		tm, err := time.Parse(time.RFC3339, s)
		require.NoError(t, err)
		return tm.UnixNano()
	}
	day := func(d int) time.Time { return time.Date(2026, 1, d, 0, 0, 0, 0, time.UTC) }
	// Most cases use a block holding data spanning Jan 10-20.
	blk := &backend.BlockMeta{StartTime: day(10), EndTime: day(20)}

	cases := []struct {
		name              string
		meta              *backend.BlockMeta
		start, end        int64
		wantOverlap       bool
		wantIndeterminate bool
	}{
		{name: "unbounded 0/0 matches everything", meta: blk, wantOverlap: true},
		{name: "window entirely before block", meta: blk, start: ts("2026-01-01T00:00:00Z"), end: ts("2026-01-05T00:00:00Z")},
		{name: "window entirely after block", meta: blk, start: ts("2026-01-25T00:00:00Z"), end: ts("2026-01-30T00:00:00Z")},
		{name: "window overlaps block start", meta: blk, start: ts("2026-01-05T00:00:00Z"), end: ts("2026-01-15T00:00:00Z"), wantOverlap: true},
		{name: "window inside block", meta: blk, start: ts("2026-01-12T00:00:00Z"), end: ts("2026-01-14T00:00:00Z"), wantOverlap: true},
		{name: "unbounded start, ends inside block", meta: blk, end: ts("2026-01-15T00:00:00Z"), wantOverlap: true},
		{name: "unbounded start, ends before block", meta: blk, end: ts("2026-01-05T00:00:00Z")},
		{name: "starts inside block, unbounded end", meta: blk, start: ts("2026-01-15T00:00:00Z"), wantOverlap: true},
		{name: "starts after block, unbounded end", meta: blk, start: ts("2026-01-25T00:00:00Z")},

		// Block metadata is second-granularity: ObjectAdded builds the times from uint32 epoch
		// SECONDS, so the recorded EndTime is the truncated second of the block's latest data and
		// real spans can extend into the following second. A window opening inside that second
		// must still select the block, or those traces are silently skipped.
		{
			name:        "window opens inside the block's final truncated second",
			meta:        &backend.BlockMeta{StartTime: day(10), EndTime: time.Date(2026, 1, 20, 0, 0, 5, 0, time.UTC)},
			start:       ts("2026-01-20T00:00:05.400Z"),
			end:         ts("2026-01-21T00:00:00Z"),
			wantOverlap: true,
		},

		// A block whose recorded range is unusable cannot be judged. Include it and say so: the
		// per-block scan bound decides what is actually deleted, so an extra block costs I/O, while
		// excluding one silently leaves data the operator asked to delete. Reachable in practice —
		// ObjectAdded skips zero timestamps, so a block completed from a replayed WAL carries none.
		{name: "zero start time is indeterminate", meta: &backend.BlockMeta{EndTime: day(20)}, start: ts("2026-01-12T00:00:00Z"), end: ts("2026-01-14T00:00:00Z"), wantOverlap: true, wantIndeterminate: true},
		{name: "zero end time is indeterminate", meta: &backend.BlockMeta{StartTime: day(10)}, start: ts("2026-01-12T00:00:00Z"), end: ts("2026-01-14T00:00:00Z"), wantOverlap: true, wantIndeterminate: true},
		{name: "both zero is indeterminate", meta: &backend.BlockMeta{}, start: ts("2026-01-12T00:00:00Z"), end: ts("2026-01-14T00:00:00Z"), wantOverlap: true, wantIndeterminate: true},
		{name: "both zero with no window is not indeterminate", meta: &backend.BlockMeta{}, wantOverlap: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			overlaps, indeterminate := blockOverlapsWindow(tc.meta, tc.start, tc.end)
			require.Equal(t, tc.wantOverlap, overlaps)
			require.Equal(t, tc.wantIndeterminate, indeterminate, "an unusable block range must be reported, not silently assumed")
		})
	}
}

// TestSubmitRedactionWindowValidation covers every window a caller can submit.
//
// The rejections are not stylistic. A one-sided window is refused because the storage layer only
// installs its time predicate when both bounds are set (vparquet{3,4,5} gate on `start > 0 && end > 0`),
// so a single bound would narrow block selection while leaving the per-block scan unbounded — deleting
// out-of-window traces from every selected block, unrecoverably. A negative bound is refused because
// the layers disagree about it: block selection treats non-zero as a real bound while the scan bound
// treats non-positive as absent, so a negative value selects every block and then scans it unbounded.
// A degenerate range is refused because it matches almost nothing while reporting success.
func TestSubmitRedactionWindowValidation(t *testing.T) {
	hour := int64(time.Hour)
	const query = `{resource.namespace = "checkout"}`

	for i, tc := range []struct {
		name       string
		start, end int64
		traceIDs   bool
		// wantMsg, when set, must appear in the rejection. Asserting only codes.InvalidArgument
		// cannot tell six distinct guards apart, so a single blanket guard rejecting every window
		// would satisfy the table.
		wantMsg string
	}{
		{name: "no window is the whole tenant", start: 0, end: 0},
		{name: "both bounds set in order", start: hour, end: 2 * hour},
		{name: "start only", start: hour, wantMsg: "must both be set or both be omitted"},
		{name: "end only", end: hour, wantMsg: "must both be set or both be omitted"},
		{name: "negative start", start: -hour, end: hour, wantMsg: "non-negative"},
		{name: "negative end", start: hour, end: -hour, wantMsg: "non-negative"},
		{name: "both negative passes an ordering check but is still refused", start: -2 * hour, end: -hour, wantMsg: "non-negative"},
		{name: "end before start", start: 2 * hour, end: hour, wantMsg: "must be before"},
		{name: "degenerate range would report success while matching nothing", start: hour, end: hour, wantMsg: "must be before"},

		// A window cannot be combined with an explicit trace-ID list: the window is not applied
		// per-trace, so the listed traces would be deleted only from the blocks that happen to
		// overlap it, leaving the rest of each trace behind under a SUCCEEDED status.
		{name: "trace ids with a window", start: hour, end: 2 * hour, traceIDs: true, wantMsg: "cannot be combined"},
		{name: "trace ids without a window are fine", traceIDs: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, s := newQuiescenceScheduler(t)
			// Derived from the table index: a tenant built from the subtest name would contain
			// spaces and fail tenant validation first, and one built from the bounds collides
			// across cases (0/0 and -hour/+hour both sum to zero).
			tenant := fmt.Sprintf("t-window-%d", i)

			req := &tempopb.SubmitRedactionRequest{
				StartTimeUnixNano: tc.start,
				EndTimeUnixNano:   tc.end,
			}
			if tc.traceIDs {
				req.TraceIds = [][]byte{{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}}
			} else {
				req.Selector = &tempopb.SubmitRedactionRequest_Query{Query: &tempopb.TraceQLSelector{Query: query}}
			}

			_, err := s.SubmitRedaction(user.InjectOrgID(ctx, tenant), req)

			if tc.wantMsg != "" {
				require.Error(t, err, "this request must be rejected before any block work")
				require.Equal(t, codes.InvalidArgument, status.Code(err))
				require.ErrorContains(t, err, tc.wantMsg, "the rejection must name the guard that fired")
				return
			}
			// Accepted requests get past validation; the tenant has no blocks, so NotFound is the
			// expected outcome rather than InvalidArgument.
			require.Equal(t, codes.NotFound, status.Code(err), "request accepted, then no blocks to redact")
		})
	}
}

// writeTenantBlocksWithRanges writes one block per [start, end] data range, so a test can control
// which blocks a redaction window should select.
func writeTenantBlocksWithRanges(ctx context.Context, t *testing.T, w backend.Writer, tenant string, ranges [][2]time.Time) []backend.UUID {
	t.Helper()
	var blockIDs []backend.UUID
	for _, r := range ranges {
		meta := &backend.BlockMeta{
			BlockID:   backend.NewUUID(),
			TenantID:  tenant,
			Version:   encoding.DefaultEncoding().Version(),
			StartTime: r[0],
			EndTime:   r[1],
		}
		blockIDs = append(blockIDs, meta.BlockID)
		require.NoError(t, w.WriteBlockMeta(ctx, meta))
	}
	return blockIDs
}

// TestSubmitRedactionOnlySelectsBlocksInWindow verifies a windowed submission creates jobs only for
// blocks whose data range overlaps the window.
//
// This asserts the filter is wired into SubmitRedaction, not just that blockOverlapsWindow computes
// the right answer: with the filter removed, every block gets a job and a request to redact one day
// silently becomes a whole-tenant redaction. On a path that rewrites blocks with no recovery, that is
// the unrecoverable direction to be wrong in, and it is the entire purpose of the feature.
func TestSubmitRedactionOnlySelectsBlocksInWindow(t *testing.T) {
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	tmpDir := t.TempDir()
	cfg.LocalWorkPath = tmpDir

	ctx, cancel := context.WithCancel(context.Background())
	store, rr, ww := newStore(ctx, t, tmpDir)
	defer func() {
		cancel()
		store.Shutdown()
	}()

	tenant := "t-window-select"
	base := time.Now().Add(-10 * 24 * time.Hour)
	day := func(n int) time.Time { return base.Add(time.Duration(n) * 24 * time.Hour) }

	// Three blocks: one entirely before the window, one inside it, one entirely after.
	ids := writeTenantBlocksWithRanges(ctx, t, backend.NewWriter(ww), tenant, [][2]time.Time{
		{day(0), day(1)}, // before the window
		{day(4), day(5)}, // inside the window
		{day(8), day(9)}, // after the window
	})
	// All three blocks must be visible before submitting. A bare sleep can leave only the in-window
	// block polled, in which case a count-of-one assertion passes without the filter ever running.
	require.Eventually(t, func() bool { return len(store.BlockMetas(tenant)) == 3 },
		5*time.Second, 50*time.Millisecond, "all three blocks must be polled, or the filter is not exercised")

	limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	s, err := New(cfg, store, limits, rr, ww)
	require.NoError(t, err)

	resp, err := s.SubmitRedaction(user.InjectOrgID(ctx, tenant), &tempopb.SubmitRedactionRequest{
		Selector:          &tempopb.SubmitRedactionRequest_Query{Query: &tempopb.TraceQLSelector{Query: `{resource.namespace = "checkout"}`}},
		StartTimeUnixNano: day(3).UnixNano(),
		EndTimeUnixNano:   day(6).UnixNano(),
	})
	require.NoError(t, err)
	require.Equal(t, int32(1), resp.JobsCreated,
		"only the block overlapping the window gets a job; a window must never widen to the whole tenant")

	// Assert which block, not just how many: a filter selecting the wrong single block would also
	// produce a count of one.
	var selected []string
	for _, j := range s.work.ListAllPendingJobs() {
		if j.GetType() == tempopb.JobType_JOB_TYPE_REDACTION && j.JobDetail.Redaction != nil {
			selected = append(selected, j.JobDetail.Redaction.BlockId)
		}
	}
	require.Equal(t, []string{ids[1].String()}, selected, "the selected block is the one inside the window")
}

// TestNextPropagatesWindowToJob verifies the batch's window reaches the per-block job, so the worker
// bounds its scan to the requested range. Without it the job carries 0/0 and each block is scanned in
// full — the window would narrow block selection but not the work done inside each block.
func TestNextPropagatesWindowToJob(t *testing.T) {
	ctx, s := newQuiescenceScheduler(t)

	tenant := "t-window-propagate"
	startNano := time.Now().Add(-48 * time.Hour).UnixNano()
	endNano := time.Now().Add(-24 * time.Hour).UnixNano()
	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "batch-window", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
		Query:             &tempopb.TraceQLSelector{Query: `{resource.namespace = "checkout"}`},
		StartTimeUnixNano: startNano,
		EndTimeUnixNano:   endNano,
	}))

	s.mergedJobs <- &work.Job{
		ID:   "wj1",
		Type: tempopb.JobType_JOB_TYPE_REDACTION,
		JobDetail: tempopb.JobDetail{
			Tenant:    tenant,
			Redaction: &tempopb.RedactionDetail{BlockId: "blk1"},
		},
	}

	resp, err := s.Next(ctx, &tempopb.NextJobRequest{WorkerId: "w1"})
	require.NoError(t, err)
	require.Equal(t, startNano, resp.Detail.Redaction.StartTimeUnixNano, "the job carries the batch's window start")
	require.Equal(t, endNano, resp.Detail.Redaction.EndTimeUnixNano, "the job carries the batch's window end")
}

// TestRescanAppliesWindowToOutputBlocks verifies the rescan honours the batch's window.
//
// SubmitRedaction filters blocks by the window, but the rescan enqueues jobs for compaction OUTPUT
// blocks, which it discovers later by ID. A compaction merging an in-window input with an out-of-window
// one produces an output whose range exceeds the window, so without the same filter the rescan re-widens
// the redaction past what the operator asked for — and the submit-time filter had explicitly excluded
// that data.
func TestRescanAppliesWindowToOutputBlocks(t *testing.T) {
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	cfg.ProviderConfig.Redaction.RescanDelay = 0
	tmpDir := t.TempDir()
	cfg.LocalWorkPath = tmpDir

	ctx, cancel := context.WithCancel(context.Background())
	store, rr, ww := newStore(ctx, t, tmpDir)
	defer func() {
		cancel()
		store.Shutdown()
	}()

	tenant := "t-rescan-window"
	base := time.Now().Add(-20 * 24 * time.Hour)
	day := func(n int) time.Time { return base.Add(time.Duration(n) * 24 * time.Hour) }

	// An in-window block that compaction is holding, and an out-of-window block that will be the
	// compaction's output.
	ids := writeTenantBlocksWithRanges(ctx, t, backend.NewWriter(ww), tenant, [][2]time.Time{
		{day(3), day(4)},   // in window, busy
		{day(12), day(13)}, // the compaction output: outside the window
	})
	inWindow, outputBlock := ids[0].String(), ids[1].String()
	require.Eventually(t, func() bool { return len(store.BlockMetas(tenant)) == 2 },
		5*time.Second, 50*time.Millisecond, "both blocks must be polled before submitting")

	limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	s, err := New(cfg, store, limits, rr, ww)
	require.NoError(t, err)

	compJob := &work.Job{
		ID:   uuid.New().String(),
		Type: tempopb.JobType_JOB_TYPE_COMPACTION,
		JobDetail: tempopb.JobDetail{
			Tenant:     tenant,
			Compaction: &tempopb.CompactionDetail{Input: []string{inWindow}},
		},
	}
	s.work.RegisterJob(compJob)
	require.NoError(t, s.work.AddJob(compJob))
	s.work.StartJob(compJob.ID)

	// The in-window block is busy, so the submission arms a rescan for that compaction job.
	_, err = s.SubmitRedaction(user.InjectOrgID(ctx, tenant), &tempopb.SubmitRedactionRequest{
		Selector:          &tempopb.SubmitRedactionRequest_Query{Query: &tempopb.TraceQLSelector{Query: `{resource.namespace = "checkout"}`}},
		StartTimeUnixNano: day(2).UnixNano(),
		EndTimeUnixNano:   day(5).UnixNano(),
	})
	require.NoError(t, err)
	batch := s.work.GetBatch(tenant)
	require.NotNil(t, batch)
	require.ElementsMatch(t, []string{compJob.ID}, batch.SkippedCompactionJobIds, "the busy in-window block arms the rescan")

	// The compaction completes, producing an out-of-window output block.
	s.work.SetJobCompactionOutput(compJob.ID, []string{outputBlock})
	s.work.CompleteJob(compJob.ID)

	before := len(s.work.ListAllPendingJobs())
	s.performRescan(ctx, batch)

	for _, j := range s.work.ListAllPendingJobs() {
		if j.GetType() == tempopb.JobType_JOB_TYPE_REDACTION && j.JobDetail.Redaction != nil {
			require.NotEqual(t, outputBlock, j.JobDetail.Redaction.BlockId,
				"an output block outside the window must not be enqueued: the window excluded that data at submit")
		}
	}
	require.Equal(t, before, len(s.work.ListAllPendingJobs()),
		"the only output block is out of window, so the rescan should enqueue nothing")
}

// TestSubmitRedactionWindowMatchingNothing verifies a window that overlaps no block is refused rather
// than accepted as an empty batch.
//
// An empty batch reported jobs_created:0 with a success status, so a mistyped bound looked like a
// completed redaction; it also held the tenant's compaction off for a full quiescence cycle for no work,
// and rejected the corrected resubmission with AlreadyExists — so the operator's natural retry failed too.
func TestSubmitRedactionWindowMatchingNothing(t *testing.T) {
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	tmpDir := t.TempDir()
	cfg.LocalWorkPath = tmpDir

	ctx, cancel := context.WithCancel(context.Background())
	store, rr, ww := newStore(ctx, t, tmpDir)
	defer func() {
		cancel()
		store.Shutdown()
	}()

	tenant := "t-window-empty"
	base := time.Now().Add(-30 * 24 * time.Hour)
	day := func(n int) time.Time { return base.Add(time.Duration(n) * 24 * time.Hour) }

	writeTenantBlocksWithRanges(ctx, t, backend.NewWriter(ww), tenant, [][2]time.Time{
		{day(1), day(2)},
		{day(3), day(4)},
	})
	require.Eventually(t, func() bool { return len(store.BlockMetas(tenant)) == 2 },
		5*time.Second, 50*time.Millisecond, "blocks must be polled before submitting")

	limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	s, err := New(cfg, store, limits, rr, ww)
	require.NoError(t, err)

	// A window well clear of both blocks — the shape a mistyped bound produces.
	_, err = s.SubmitRedaction(user.InjectOrgID(ctx, tenant), &tempopb.SubmitRedactionRequest{
		Selector:          &tempopb.SubmitRedactionRequest_Query{Query: &tempopb.TraceQLSelector{Query: `{resource.namespace = "checkout"}`}},
		StartTimeUnixNano: day(20).UnixNano(),
		EndTimeUnixNano:   day(21).UnixNano(),
	})
	require.Error(t, err, "a window overlapping no block must fail, not report an empty success")
	require.Equal(t, codes.NotFound, status.Code(err))

	require.Nil(t, s.work.GetBatch(tenant), "no batch may be created: it would gate compaction for a quiescence cycle for no work")
	require.False(t, s.work.TenantPending(tenant), "the tenant's compaction must not be held")
}

// TestSubmitRedactionDryRunAllBlocksBusy verifies a dry-run whose every in-window block is mid-compaction
// is refused rather than turned into a batch that never scans anything.
//
// The zero-match guard exempts a submission with deferred blocks, because an apply-mode batch must
// survive for the rescan to pick those blocks up once compaction finishes. A dry-run arms no rescan, so
// that exemption leaves a batch with zero jobs and no rescan armed: nothing will ever be scanned, the
// call still reports success, the tenant's compaction is held off for a quiescence cycle, and the
// operator's corrected resubmission is rejected with AlreadyExists — every consequence the guard exists
// to prevent.
func TestSubmitRedactionDryRunAllBlocksBusy(t *testing.T) {
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	cfg.ProviderConfig.Redaction.RescanDelay = 0
	tmpDir := t.TempDir()
	cfg.LocalWorkPath = tmpDir

	ctx, cancel := context.WithCancel(context.Background())
	store, rr, ww := newStore(ctx, t, tmpDir)
	defer func() {
		cancel()
		store.Shutdown()
	}()

	tenant := "t-dryrun-busy"
	base := time.Now().Add(-20 * 24 * time.Hour)
	day := func(n int) time.Time { return base.Add(time.Duration(n) * 24 * time.Hour) }

	// One block, inside the window, held by a running compaction.
	ids := writeTenantBlocksWithRanges(ctx, t, backend.NewWriter(ww), tenant, [][2]time.Time{
		{day(3), day(4)},
	})
	require.Eventually(t, func() bool { return len(store.BlockMetas(tenant)) == 1 },
		5*time.Second, 50*time.Millisecond, "the block must be polled before submitting")

	limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	s, err := New(cfg, store, limits, rr, ww)
	require.NoError(t, err)

	compJob := &work.Job{
		ID:   uuid.New().String(),
		Type: tempopb.JobType_JOB_TYPE_COMPACTION,
		JobDetail: tempopb.JobDetail{
			Tenant:     tenant,
			Compaction: &tempopb.CompactionDetail{Input: []string{ids[0].String()}},
		},
	}
	s.work.RegisterJob(compJob)
	require.NoError(t, s.work.AddJob(compJob))
	s.work.StartJob(compJob.ID)

	req := &tempopb.SubmitRedactionRequest{
		Selector:          &tempopb.SubmitRedactionRequest_Query{Query: &tempopb.TraceQLSelector{Query: `{resource.namespace = "checkout"}`}},
		Mode:              tempopb.RedactionMode_REDACTION_MODE_DRY_RUN,
		StartTimeUnixNano: day(2).UnixNano(),
		EndTimeUnixNano:   day(5).UnixNano(),
	}

	_, err = s.SubmitRedaction(user.InjectOrgID(ctx, tenant), req)
	require.Error(t, err, "a dry-run that can never scan anything must be refused")
	// FailedPrecondition, not NotFound: the blocks did overlap the window. Reporting "no blocks overlap"
	// would send the operator to widen the window when the fix is to retry.
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
	require.ErrorContains(t, err, "being compacted", "the error must name the real cause, not the window")
	require.NotContains(t, err.Error(), "overlap", "must not misreport this as a window mismatch")
	require.Nil(t, s.work.GetBatch(tenant), "no batch may be left behind to block resubmission")

	// The same submission in apply mode IS accepted: its batch arms a rescan, so the deferred block
	// is picked up once the compaction finishes. This is the distinction the guard has to preserve.
	req.Mode = tempopb.RedactionMode_REDACTION_MODE_APPLY
	_, err = s.SubmitRedaction(user.InjectOrgID(ctx, tenant), req)
	require.NoError(t, err, "an apply-mode batch with deferred blocks must still be created")
	batch := s.work.GetBatch(tenant)
	require.NotNil(t, batch)
	require.ElementsMatch(t, []string{compJob.ID}, batch.SkippedCompactionJobIds)
	require.NotZero(t, batch.RescanAfterUnixNano, "the apply batch must arm a rescan")
}

// TestDropBatchesWithUnusableWindows verifies a persisted batch whose window cannot be honoured is
// discarded at load rather than acted on.
//
// SubmitRedaction validates on the way in, so an unusable window can only arrive already persisted. Both
// downstream consumers would otherwise have to cope and neither can do it safely: skipping the batch's
// blocks under-deletes silently, and treating the window as unbounded deletes every query match in those
// blocks regardless of time — unrecoverable over-deletion. Refusing the batch destroys nothing.
func TestDropBatchesWithUnusableWindows(t *testing.T) {
	hour := int64(time.Hour)

	for _, tc := range []struct {
		name       string
		start, end int64
		wantKept   bool
	}{
		{name: "unwindowed batch is kept", wantKept: true},
		{name: "ordered window is kept", start: hour, end: 2 * hour, wantKept: true},
		{name: "negative bounds are discarded", start: -1, end: -1},
		{name: "negative start is discarded", start: -hour, end: hour},
		{name: "inverted window is discarded", start: 2 * hour, end: hour},
		{name: "zero-width window is discarded", start: hour, end: hour},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, s := newQuiescenceScheduler(t)
			tenant := "t-load-window"

			require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
				BatchId:           "batch-1",
				TenantId:          tenant,
				StartTimeUnixNano: tc.start,
				EndTimeUnixNano:   tc.end,
			}))

			s.dropBatchesWithUnusableWindows()

			if tc.wantKept {
				require.NotNil(t, s.work.GetBatch(tenant), "a usable window must survive load")
				return
			}
			require.Nil(t, s.work.GetBatch(tenant),
				"an unusable persisted window must be discarded, not acted on with a guessed scope")
		})
	}
}

// TestSubmitRedactionDryRunArmsNoRescan pins the premise the dry-run zero-match guard rests on.
//
// That guard refuses a dry-run whose every in-window block is busy, justified by "a dry-run arms no
// rescan". Nothing else asserts that: making dry-runs arm rescans leaves every other test passing, and
// silently inverts the guard from correct to wrong (it would then refuse dry-runs that WOULD have been
// picked up later). The refusal test cannot cover this, because its submission is rejected and no batch
// exists to inspect — so this one needs a dry-run that succeeds, with one idle block and one busy block.
func TestSubmitRedactionDryRunArmsNoRescan(t *testing.T) {
	cfg := Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
	// Non-zero so an armed deadline is unmistakable, but below the default prune_age (config
	// validation requires prune_age > rescan_delay so jobs are still in memory when a rescan fires).
	cfg.ProviderConfig.Redaction.RescanDelay = time.Minute
	tmpDir := t.TempDir()
	cfg.LocalWorkPath = tmpDir

	ctx, cancel := context.WithCancel(context.Background())
	store, rr, ww := newStore(ctx, t, tmpDir)
	defer func() {
		cancel()
		store.Shutdown()
	}()

	base := time.Now().Add(-20 * 24 * time.Hour)
	day := func(n int) time.Time { return base.Add(time.Duration(n) * 24 * time.Hour) }

	for _, tc := range []struct {
		name      string
		mode      tempopb.RedactionMode
		wantArmed bool
	}{
		{name: "apply arms a rescan for the busy block", mode: tempopb.RedactionMode_REDACTION_MODE_APPLY, wantArmed: true},
		{name: "dry-run arms none", mode: tempopb.RedactionMode_REDACTION_MODE_DRY_RUN},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tenant := "t-dryrun-rescan-" + tc.mode.String()

			// Two in-window blocks: one held by a compaction, one free, so the submission succeeds
			// with a job AND a deferred block.
			ids := writeTenantBlocksWithRanges(ctx, t, backend.NewWriter(ww), tenant, [][2]time.Time{
				{day(3), day(4)},
				{day(3), day(4)},
			})
			require.Eventually(t, func() bool { return len(store.BlockMetas(tenant)) == 2 },
				5*time.Second, 50*time.Millisecond, "both blocks must be polled before submitting")

			limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.NewRegistry())
			require.NoError(t, err)
			s, err := New(cfg, store, limits, rr, ww)
			require.NoError(t, err)

			compJob := &work.Job{
				ID:   uuid.New().String(),
				Type: tempopb.JobType_JOB_TYPE_COMPACTION,
				JobDetail: tempopb.JobDetail{
					Tenant:     tenant,
					Compaction: &tempopb.CompactionDetail{Input: []string{ids[0].String()}},
				},
			}
			s.work.RegisterJob(compJob)
			require.NoError(t, s.work.AddJob(compJob))
			s.work.StartJob(compJob.ID)

			_, err = s.SubmitRedaction(user.InjectOrgID(ctx, tenant), &tempopb.SubmitRedactionRequest{
				Selector:          &tempopb.SubmitRedactionRequest_Query{Query: &tempopb.TraceQLSelector{Query: `{resource.namespace = "checkout"}`}},
				Mode:              tc.mode,
				StartTimeUnixNano: day(2).UnixNano(),
				EndTimeUnixNano:   day(5).UnixNano(),
			})
			require.NoError(t, err, "one block is free, so the submission must succeed")

			batch := s.work.GetBatch(tenant)
			require.NotNil(t, batch)

			if tc.wantArmed {
				require.NotZero(t, batch.RescanAfterUnixNano, "apply mode must arm a rescan for the deferred block")
				require.ElementsMatch(t, []string{compJob.ID}, batch.SkippedCompactionJobIds)
				return
			}
			require.Zero(t, batch.RescanAfterUnixNano,
				"a dry-run rewrites nothing, so there is no output block to re-cover and no rescan to arm")
			require.Empty(t, batch.SkippedCompactionJobIds)
		})
	}
}
