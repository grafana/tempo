package frontend

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
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
