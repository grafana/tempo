package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestWindowBoundNano covers the resolution of a redaction window bound. The forms come from the
// shared parseTime helper (used by redact and five query commands); what this adds is the empty ==
// unbounded case and a guard on the representable range, because time.Time.UnixNano() is undefined
// outside 1678-2262 and silently wraps — a bound written as "far future" would otherwise resolve
// negative, which on a redaction means selecting every block and scanning each one unbounded.
func TestWindowBoundNano(t *testing.T) {
	for _, tc := range []struct {
		name    string
		spec    string
		wantErr bool
		// wantAbout, when non-zero, is the expected instant within a generous tolerance.
		wantAbout time.Time
	}{
		{name: "empty is unbounded", spec: ""},
		{name: "whitespace is unbounded", spec: "   "},
		{name: "now", spec: "now", wantAbout: time.Now()},
		{name: "relative hours", spec: "now-1h", wantAbout: time.Now().Add(-time.Hour)},
		{name: "relative days", spec: "now-7d", wantAbout: time.Now().Add(-7 * 24 * time.Hour)},
		{name: "relative years are supported by the shared parser", spec: "now-1y", wantAbout: time.Now().Add(-365 * 24 * time.Hour)},
		{name: "absolute RFC3339", spec: "2026-01-02T03:04:05Z", wantAbout: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)},
		{name: "garbage", spec: "yesterday", wantErr: true},
		{name: "future relative is rejected by the shared parser", spec: "now+1h", wantErr: true},
		{name: "beyond the representable range", spec: "2300-01-01T00:00:00Z", wantErr: true},
		{name: "before the representable range", spec: "1600-01-01T00:00:00Z", wantErr: true},
		{name: "absurd relative duration", spec: "now-99999999w", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := windowBoundNano(tc.spec)
			if tc.wantErr {
				require.Error(t, err, "a bound this command cannot represent must be an error, never a silent 0")
				return
			}
			require.NoError(t, err)
			if tc.wantAbout.IsZero() {
				require.Zero(t, got, "an omitted bound is unbounded")
				return
			}
			require.InDelta(t, tc.wantAbout.UnixNano(), got, float64(2*time.Minute), "resolved close to the expected instant")
		})
	}
}
