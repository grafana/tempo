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
	for _, tc := range []struct {
		name       string
		start, end int64
		wantReject bool
	}{
		{name: "no window is the whole tenant", start: 0, end: 0},
		{name: "both bounds set in order", start: hour, end: 2 * hour},
		{name: "start only leaves the scan unbounded", start: hour, wantReject: true},
		{name: "end only leaves the scan unbounded", end: hour, wantReject: true},
		{name: "negative start selects everything then scans unbounded", start: -hour, end: hour, wantReject: true},
		{name: "negative end", start: hour, end: -hour, wantReject: true},
		{name: "both negative passes an ordering check but still diverges", start: -2 * hour, end: -hour, wantReject: true},
		{name: "end before start", start: 2 * hour, end: hour, wantReject: true},
		{name: "degenerate range reports success while matching nothing", start: hour, end: hour, wantReject: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, s := newQuiescenceScheduler(t)
			// A tenant ID built from the subtest name would contain spaces and fail tenant
			// validation first, making every case pass for the wrong reason.
			tenant := fmt.Sprintf("t-window-%d", tc.start+tc.end)
			_, err := s.SubmitRedaction(user.InjectOrgID(ctx, tenant), &tempopb.SubmitRedactionRequest{
				TraceIds:          [][]byte{{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}},
				StartTimeUnixNano: tc.start,
				EndTimeUnixNano:   tc.end,
			})
			if tc.wantReject {
				require.Error(t, err, "this window must be rejected before any block work")
				require.Equal(t, codes.InvalidArgument, status.Code(err))
				return
			}
			// Accepted windows get past validation; the tenant has no blocks, so NotFound is the
			// expected outcome rather than InvalidArgument.
			require.Equal(t, codes.NotFound, status.Code(err), "window accepted, then no blocks to redact")
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
	writeTenantBlocksWithRanges(ctx, t, backend.NewWriter(ww), tenant, [][2]time.Time{
		{day(0), day(1)},
		{day(4), day(5)},
		{day(8), day(9)},
	})
	time.Sleep(150 * time.Millisecond) // let the blocklist poll pick them up

	limits, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{}}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	s, err := New(cfg, store, limits, rr, ww)
	require.NoError(t, err)

	resp, err := s.SubmitRedaction(user.InjectOrgID(ctx, tenant), &tempopb.SubmitRedactionRequest{
		TraceIds:          [][]byte{{0x01}},
		StartTimeUnixNano: day(3).UnixNano(),
		EndTimeUnixNano:   day(6).UnixNano(),
	})
	require.NoError(t, err)
	require.Equal(t, int32(1), resp.JobsCreated,
		"only the block overlapping the window gets a job; a window must never widen to the whole tenant")
}

// TestNextPropagatesWindowToJob verifies the batch's window reaches the per-block job, so the worker
// bounds its scan to the requested range. Without it the job carries 0/0 and each block is scanned in
// full — the window would narrow block selection but not the work done inside each block.
func TestNextPropagatesWindowToJob(t *testing.T) {
	ctx, s := newQuiescenceScheduler(t)
	s.cfg.JobTimeout = 200 * time.Millisecond

	tenant := "t-window-propagate"
	startNano := time.Now().Add(-48 * time.Hour).UnixNano()
	endNano := time.Now().Add(-24 * time.Hour).UnixNano()
	require.NoError(t, s.work.AddBatch(&tempopb.RedactionBatch{
		BatchId: "batch-window", TenantId: tenant, CreatedAtUnixNano: time.Now().UnixNano(),
		TraceIds:          [][]byte{{0x01}},
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
		TraceIds:          [][]byte{{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}},
		StartTimeUnixNano: day(2).UnixNano(),
		EndTimeUnixNano:   day(5).UnixNano(),
	})
	require.NoError(t, err)
	batch := s.work.GetBatch(tenant)
	require.NotNil(t, batch)
	require.Equal(t, []string{compJob.ID}, batch.SkippedCompactionJobIds, "the busy in-window block arms the rescan")

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
		TraceIds:          [][]byte{{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}},
		StartTimeUnixNano: day(20).UnixNano(),
		EndTimeUnixNano:   day(21).UnixNano(),
	})
	require.Error(t, err, "a window overlapping no block must fail, not report an empty success")
	require.Equal(t, codes.NotFound, status.Code(err))

	require.Nil(t, s.work.GetBatch(tenant), "no batch may be created: it would gate compaction for a quiescence cycle for no work")
	require.False(t, s.work.TenantPending(tenant), "the tenant's compaction must not be held")
}
