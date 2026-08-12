package backendscheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/tempodb/backend"
)

// TestCoveredRange covers the audit record that describes a redaction's blast radius.
//
// The hard cases are blocks whose recorded range is unusable (a zero StartTime or EndTime). Block
// selection deliberately INCLUDES those, so they are reachable by construction — and time.Time{}
// precedes every real timestamp, so letting one into a running minimum drags the reported start back
// to year 1. That understates the blast radius on exactly the blocks selection was least sure about,
// in the one record an operator has for an operation with no undo.
func TestCoveredRange(t *testing.T) {
	base := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	at := base.Add

	meta := func(start, end time.Time) *backend.BlockMeta {
		return &backend.BlockMeta{StartTime: start, EndTime: end}
	}

	for _, tc := range []struct {
		name      string
		metas     []*backend.BlockMeta
		wantOK    bool
		wantStart time.Time
		wantEnd   time.Time
	}{
		{name: "no blocks reports nothing", metas: nil},
		{
			name:      "a single usable block",
			metas:     []*backend.BlockMeta{meta(at(time.Hour), at(2*time.Hour))},
			wantOK:    true,
			wantStart: at(time.Hour),
			wantEnd:   at(2 * time.Hour),
		},
		{
			name: "min and max across usable blocks",
			metas: []*backend.BlockMeta{
				meta(at(2*time.Hour), at(3*time.Hour)),
				meta(at(time.Hour), at(90*time.Minute)),
				meta(at(4*time.Hour), at(5*time.Hour)),
			},
			wantOK:    true,
			wantStart: at(time.Hour),
			wantEnd:   at(5 * time.Hour),
		},
		{
			name: "a zero-time block trailing the list must not drag the start to year 1",
			metas: []*backend.BlockMeta{
				meta(at(time.Hour), at(2*time.Hour)),
				meta(at(3*time.Hour), at(4*time.Hour)),
				meta(time.Time{}, time.Time{}),
			},
			wantOK:    true,
			wantStart: at(time.Hour),
			wantEnd:   at(4 * time.Hour),
		},
		{
			name: "a zero-time block mid-list must not reset the accumulator",
			metas: []*backend.BlockMeta{
				meta(at(time.Hour), at(2*time.Hour)),
				meta(time.Time{}, time.Time{}),
				meta(at(3*time.Hour), at(4*time.Hour)),
			},
			wantOK:    true,
			wantStart: at(time.Hour),
			wantEnd:   at(4 * time.Hour),
		},
		{
			name: "a zero-time block leading the list",
			metas: []*backend.BlockMeta{
				meta(time.Time{}, time.Time{}),
				meta(at(time.Hour), at(2*time.Hour)),
			},
			wantOK:    true,
			wantStart: at(time.Hour),
			wantEnd:   at(2 * time.Hour),
		},
		{
			name: "a half-zero range contributes nothing",
			metas: []*backend.BlockMeta{
				meta(at(time.Hour), time.Time{}),
				meta(at(3*time.Hour), at(4*time.Hour)),
			},
			wantOK:    true,
			wantStart: at(3 * time.Hour),
			wantEnd:   at(4 * time.Hour),
		},
		{
			name: "only unusable blocks reports nothing rather than year 1",
			metas: []*backend.BlockMeta{
				meta(time.Time{}, time.Time{}),
				meta(time.Time{}, time.Time{}),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start, end, ok := coveredRange(tc.metas)
			require.Equal(t, tc.wantOK, ok, "ok must report whether any block had a usable range")
			if !tc.wantOK {
				return
			}
			require.Equal(t, tc.wantStart, start, "covered start")
			require.Equal(t, tc.wantEnd, end, "covered end")
		})
	}
}

// TestCoveredRangeLabel covers the rendering of a covered bound for the audit record. The unknown case
// is the point: without it, "no block reported a usable range" and a genuine year-1 timestamp print
// identically, so an operator reading the record cannot tell a missing range from a real one.
func TestCoveredRangeLabel(t *testing.T) {
	at := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	require.Equal(t, "2026-03-01T12:00:00Z", coveredRangeLabel(at, true))
	require.Equal(t, "unknown", coveredRangeLabel(at, false), "a bound with no usable range must not render as a timestamp")
	require.Equal(t, "unknown", coveredRangeLabel(time.Time{}, false))
}
