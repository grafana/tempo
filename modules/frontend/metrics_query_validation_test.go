package frontend

import (
	"bytes"
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/log" //nolint:all deprecated
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/v3/modules/overrides"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/stretchr/testify/require"
)

// mockOverridesMaxMetricsDuration satisfies overrides.Interface for the
// purpose of validateMetricsQueryMaxDuration tests. Embedding the interface
// means any method other than MaxMetricsDuration will panic if called — the
// tests below never trigger that.
type mockOverridesMaxMetricsDuration struct {
	overrides.Interface
	perTenant map[string]time.Duration
}

func (m *mockOverridesMaxMetricsDuration) MaxMetricsDuration(userID string) time.Duration {
	return m.perTenant[userID]
}

func TestValidateMetricsQueryMaxDuration(t *testing.T) {
	tcs := []struct {
		name      string
		perTenant map[string]time.Duration
		fallback  time.Duration
		rangeDur  time.Duration
		wantErr   bool
	}{
		{
			name:      "override below range rejects",
			perTenant: map[string]time.Duration{"foo": 24 * time.Hour},
			rangeDur:  25 * time.Hour,
			wantErr:   true,
		},
		{
			name:      "override above range accepts",
			perTenant: map[string]time.Duration{"foo": 48 * time.Hour},
			rangeDur:  24 * time.Hour,
			wantErr:   false,
		},
		{
			name:     "fallback applies when tenant has no override",
			fallback: 24 * time.Hour,
			rangeDur: 25 * time.Hour,
			wantErr:  true,
		},
		{
			name:     "no override, no fallback -> unlimited",
			rangeDur: 10 * 24 * time.Hour,
			wantErr:  false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ctx := user.InjectOrgID(context.Background(), "foo")
			req := &tempopb.QueryRangeRequest{
				Start: uint64(1 * time.Hour),
				End:   uint64(1*time.Hour + tc.rangeDur),
			}
			o := &mockOverridesMaxMetricsDuration{perTenant: tc.perTenant}

			err := validateMetricsQueryMaxDuration(ctx, o, tc.fallback, req)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateMetricsQueryMaxDuration_NegativeCapFailsClosed(t *testing.T) {
	// A negative max_metrics_duration is a misconfiguration. The prior sharder
	// check failed closed (every query rejected) because `positive > -1s` is
	// always true. The unsigned comparison must preserve that semantic instead
	// of silently bypassing the cap via uint64 underflow wraparound.
	tcs := []struct {
		name      string
		perTenant map[string]time.Duration
		fallback  time.Duration
	}{
		{
			name:      "per-tenant override is negative",
			perTenant: map[string]time.Duration{"foo": -1 * time.Second},
		},
		{
			name:     "config fallback is negative, no override",
			fallback: -1 * time.Second,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ctx := user.InjectOrgID(context.Background(), "foo")
			o := &mockOverridesMaxMetricsDuration{perTenant: tc.perTenant}
			req := &tempopb.QueryRangeRequest{
				Start: uint64(1 * time.Hour),
				End:   uint64(2 * time.Hour),
			}
			err := validateMetricsQueryMaxDuration(ctx, o, tc.fallback, req)
			require.Error(t, err, "negative cap must fail closed, not silently bypass")
		})
	}
}

func TestValidateMetricsQueryMaxDuration_OverflowSafe(t *testing.T) {
	// A range whose nanosecond delta exceeds math.MaxInt64 must not wrap to a
	// negative time.Duration and silently bypass the cap.
	ctx := user.InjectOrgID(context.Background(), "foo")
	o := &mockOverridesMaxMetricsDuration{perTenant: map[string]time.Duration{"foo": time.Hour}}

	req := &tempopb.QueryRangeRequest{
		Start: 0,
		End:   uint64(math.MaxInt64) + 1, // overflows int64 when cast directly
	}
	err := validateMetricsQueryMaxDuration(ctx, o, 0, req)
	require.Error(t, err, "huge range must be rejected, not silently bypassed via overflow")
}

func TestValidateQueryRangeReqScannedBytesLimit(t *testing.T) {
	tcs := []struct {
		name            string
		start           time.Time
		maxScannedBytes uint64
		maxDuration     time.Duration
		maxIntervals    uint64
		wantErr         string
		wantResult      string
		wantCalls       int
	}{
		{
			name:  "limit off reads no blocklist and logs nothing",
			start: hoursAgo(2),
		},
		{
			name:            "over max_duration with a too small step keeps the max_duration error",
			start:           hoursAgo(8),
			maxScannedBytes: 350,
			maxDuration:     6 * time.Hour,
			maxIntervals:    10,
			wantErr:         "metrics query time range exceeds the maximum allowed duration of 6h0m0s",
			wantResult:      scannedBytesResultRejectedMaxDuration,
			wantCalls:       1,
		},
		{
			name:            "negative max_duration fails closed without logging",
			start:           hoursAgo(2),
			maxScannedBytes: 350,
			maxDuration:     -time.Hour,
			wantErr:         "metrics query time range exceeds the maximum allowed duration of -1h0m0s",
		},
		{
			name:            "step error skips the blocklist walk",
			start:           hoursAgo(2),
			maxScannedBytes: 350,
			maxIntervals:    10,
			wantErr:         "step of 1m0s is too small, minimum step for given range is 12m0s",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			cfg := Config{}
			cfg.Metrics.MaxIntervals = tc.maxIntervals
			cfg.Metrics.Sharder = QueryRangeSharderConfig{MaxDuration: tc.maxDuration, MaxScannedBytes: tc.maxScannedBytes}
			o := newTestOverrides(t, overrides.ReadOverrides{})
			rdr := &countingMockReader{mockReader: mockReader{metas: hourlyMetricsBlocks(scannedBytesTestEnd)}}
			sl := newMetricsScannedBytesLimit(rdr, o, cfg.Metrics.Sharder, true, log.NewLogfmtLogger(&buf))

			countBefore := resultCount("foo", metricsOp, tc.wantResult)
			err := validateQueryRangeReq(user.InjectOrgID(context.Background(), "foo"), cfg, o, sl, &tempopb.QueryRangeRequest{
				Start: uint64(tc.start.UnixNano()),
				End:   uint64(scannedBytesTestEnd.UnixNano()),
				Step:  uint64(time.Minute),
			})

			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tc.wantErr)
			}
			if strings.Contains(tc.wantErr, "maximum allowed duration") {
				require.ErrorIs(t, err, errMetricsMaxDuration)
			}
			require.Equal(t, tc.wantCalls, rdr.callCount())
			if tc.wantResult == "" {
				require.Empty(t, buf.String())
			} else {
				require.Contains(t, buf.String(), "result="+tc.wantResult+" ")
				require.Equal(t, 1.0, resultCount("foo", metricsOp, tc.wantResult)-countBefore)
			}
		})
	}
}
