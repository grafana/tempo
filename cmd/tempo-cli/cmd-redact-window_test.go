package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestWindowBoundNano covers the resolution of a redaction window bound. The accepted forms come from
// the shared parseTime family (used by redact and five query commands); what this adds is the
// omitted == unbounded case, a guard on the representable range, and the epoch collision.
//
// The range guard is bounded on the resolved instant rather than its year. UnixNano() is defined only
// up to 2262-04-11T23:47:16.854775807Z, so a year-granular check leaks the remaining ~8 months of 2262
// — those wrap to large negatives while preserving their order, which passes an ordering check.
func TestWindowBoundNano(t *testing.T) {
	// One fixed instant for every case, as validate() supplies.
	now := time.Now()

	for _, tc := range []struct {
		name    string
		spec    string
		wantErr bool
		wantSet bool
		// wantAbout, when non-zero, is the expected instant within a generous tolerance.
		wantAbout time.Time
	}{
		{name: "empty is unbounded", spec: ""},
		{name: "whitespace is unbounded", spec: "   "},
		{name: "now", spec: "now", wantSet: true, wantAbout: now},
		{name: "relative hours", spec: "now-1h", wantSet: true, wantAbout: now.Add(-time.Hour)},
		{name: "relative days", spec: "now-7d", wantSet: true, wantAbout: now.Add(-7 * 24 * time.Hour)},
		{name: "relative years are supported by the shared parser", spec: "now-1y", wantSet: true, wantAbout: now.Add(-365 * 24 * time.Hour)},
		{name: "absolute RFC3339", spec: "2026-01-02T03:04:05Z", wantSet: true, wantAbout: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)},
		{name: "garbage", spec: "yesterday", wantErr: true},
		{name: "future relative is rejected by the shared parser", spec: "now+1h", wantErr: true},
		{name: "absurd relative duration", spec: "now-99999999w", wantErr: true},

		// Boundary of what UnixNano() can express. The last valid nanosecond must be accepted and
		// the first invalid one refused; a year-granular guard accepts the whole of 2262.
		{name: "last representable instant", spec: "2262-04-11T23:47:16Z", wantSet: true, wantAbout: time.Date(2262, 4, 11, 23, 47, 16, 0, time.UTC)},
		{name: "first instant past the limit still inside year 2262", spec: "2262-04-12T00:00:00Z", wantErr: true},
		{name: "late 2262 wraps negative", spec: "2262-12-31T00:00:00Z", wantErr: true},
		{name: "far future", spec: "2300-01-01T00:00:00Z", wantErr: true},

		// Pre-epoch resolves negative, which every layer rejects.
		{name: "pre-epoch is refused", spec: "1900-01-01T00:00:00Z", wantErr: true},
		{name: "far past", spec: "1600-01-01T00:00:00Z", wantErr: true},

		// Exactly the epoch is the sentinel every layer reads as "no bound", so accepting it would
		// silently widen a single-instant window to the whole tenant.
		{name: "the epoch itself is refused", spec: "1970-01-01T00:00:00Z", wantErr: true},
		{name: "one nanosecond after the epoch is fine", spec: "1970-01-01T00:00:00.000000001Z", wantSet: true, wantAbout: time.Unix(0, 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok, err := windowBoundNano(tc.spec, now)
			if tc.wantErr {
				require.Error(t, err, "a bound this command cannot represent must be an error, never a silent 0")
				require.False(t, ok)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantSet, ok, "ok must report whether the flag was supplied")
			if !tc.wantSet {
				require.Zero(t, got, "an omitted bound is unbounded")
				return
			}
			require.Positive(t, got, "a supplied bound must never resolve to the unbounded sentinel")
			require.InDelta(t, tc.wantAbout.UnixNano(), got, float64(2*time.Minute), "resolved close to the expected instant")
		})
	}
}
