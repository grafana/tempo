package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseTimeSpec(t *testing.T) {
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{"", 0, false}, // empty = unbounded (0)
		{"now", now.UnixNano(), false},
		{"now-7d", now.Add(-7 * 24 * time.Hour).UnixNano(), false},
		{"now-1h", now.Add(-time.Hour).UnixNano(), false},
		{"now-2w", now.Add(-14 * 24 * time.Hour).UnixNano(), false},
		{"now-1h30m", now.Add(-90 * time.Minute).UnixNano(), false},
		{"now+1h", now.Add(time.Hour).UnixNano(), false},
		{"2026-07-01T00:00:00Z", time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC).UnixNano(), false},
		{"garbage", 0, true},
		{"now-5x", 0, true},
		{"now-", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := parseTimeSpec(tc.in, now)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
