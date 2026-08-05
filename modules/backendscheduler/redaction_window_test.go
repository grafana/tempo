package backendscheduler

import (
	"testing"
	"time"

	"github.com/gogo/status"
	"github.com/grafana/dskit/user"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
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
