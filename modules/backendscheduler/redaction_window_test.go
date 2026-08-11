package backendscheduler

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/gogo/status"
	"github.com/grafana/dskit/user"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

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
	// Block holds data spanning Jan 10–20.
	blk := &backend.BlockMeta{
		StartTime: time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC),
	}

	cases := []struct {
		name        string
		start, end  int64
		wantOverlap bool
	}{
		{"unbounded 0/0 matches everything", 0, 0, true},
		{"window entirely before block", ts("2026-01-01T00:00:00Z"), ts("2026-01-05T00:00:00Z"), false},
		{"window entirely after block", ts("2026-01-25T00:00:00Z"), ts("2026-01-30T00:00:00Z"), false},
		{"window overlaps block start", ts("2026-01-05T00:00:00Z"), ts("2026-01-15T00:00:00Z"), true},
		{"window inside block", ts("2026-01-12T00:00:00Z"), ts("2026-01-14T00:00:00Z"), true},
		{"unbounded start, ends inside block", 0, ts("2026-01-15T00:00:00Z"), true},
		{"unbounded start, ends before block", 0, ts("2026-01-05T00:00:00Z"), false},
		{"starts inside block, unbounded end", ts("2026-01-15T00:00:00Z"), 0, true},
		{"starts after block, unbounded end", ts("2026-01-25T00:00:00Z"), 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.wantOverlap, blockOverlapsWindow(blk, tc.start, tc.end))
		})
	}
}

// TestSubmitRedactionRejectsInvalidWindow verifies a window with end <= start is rejected before
// any block work; 0 on either side is unbounded and allowed.
func TestSubmitRedactionRejectsInvalidWindow(t *testing.T) {
	ctx, s := newQuiescenceScheduler(t)
	_, err := s.SubmitRedaction(user.InjectOrgID(ctx, "t-badwindow"), &tempopb.SubmitRedactionRequest{
		TraceIds:          [][]byte{{0x01}},
		StartTimeUnixNano: 2000,
		EndTimeUnixNano:   1000, // end before start
	})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
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
