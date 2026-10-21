package distributor

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/gogo/status"
	"github.com/grafana/dskit/limiter"
	"github.com/grafana/dskit/services"
	"github.com/grafana/dskit/user"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/grafana/tempo/v3/modules/overrides"
	v1 "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/v3/pkg/util/test"
)

// prepareWithPerTenantOverrides builds a distributor with the given per-tenant overrides YAML on top of the flag defaults.
func prepareWithPerTenantOverrides(t *testing.T, tenantYAML string) *Distributor {
	t.Helper()

	limits := overrides.Config{}
	limits.RegisterFlagsAndApplyDefaults(&flag.FlagSet{})
	f := filepath.Join(t.TempDir(), "overrides.yaml")
	require.NoError(t, os.WriteFile(f, []byte(tenantYAML), 0o600))
	limits.PerTenantOverrideConfig = f
	limits.PerTenantOverridePeriod = model.Duration(time.Hour)

	distributorConfig, _, l, mw := setupDependencies(t, overrides.Config{})
	overridesSvc, err := overrides.NewOverrides(limits, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	require.NoError(t, services.StartAndAwaitRunning(context.Background(), overridesSvc))
	t.Cleanup(func() { _ = services.StopAndAwaitTerminated(context.Background(), overridesSvc) })

	d, err := New(distributorConfig, LocalPushTargets{}, nil, overridesSvc, mw, kitlog.NewNopLogger(), l, prometheus.NewPedanticRegistry())
	require.NoError(t, err)
	return d
}

func TestExtractBasicInfo(t *testing.T) {
	tcs := []struct {
		name          string
		orgID         string
		wantUserID    string
		wantLimitsKey string
		wantCode      codes.Code
	}{
		{name: "plain tenant", orgID: "test", wantUserID: "test", wantLimitsKey: "test", wantCode: codes.OK},
		{name: "tenant with metadata", orgID: "test:source=x", wantUserID: "test", wantLimitsKey: "test:source=x", wantCode: codes.OK},
		{name: "tenant with several metadata pairs", orgID: "test:env=prod:source=x", wantUserID: "test", wantLimitsKey: "test:env=prod:source=x", wantCode: codes.OK},
		{name: "malformed metadata", orgID: "test:garbage", wantCode: codes.InvalidArgument},
		{name: "unsorted metadata", orgID: "test:source=x:env=prod", wantCode: codes.InvalidArgument},
		{name: "empty tenant", orgID: "", wantCode: codes.InvalidArgument},
		{name: "multiple tenants", orgID: "a|b", wantCode: codes.InvalidArgument},
	}

	d := prepare(t, overrides.Config{}, nil)
	traces := batchesToTraces(t, []*v1.ResourceSpans{test.MakeBatch(3, nil)})

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			userID, limitsKey, spanCount, size, err := d.extractBasicInfo(user.InjectOrgID(context.Background(), tc.orgID), traces)
			if tc.wantCode != codes.OK {
				s, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, tc.wantCode, s.Code())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantUserID, userID)
			require.Equal(t, tc.wantLimitsKey, limitsKey)
			require.Equal(t, 3, spanCount)
			require.Positive(t, size)
		})
	}
}

func TestCheckForRateLimits_TenantMetadata(t *testing.T) {
	const tenantYAML = `
overrides:
  test-user:
    ingestion:
      rate_limit_bytes: 1000
      burst_size_bytes: 1000
  ":source=x":
    ingestion:
      rate_limit_bytes: 50
      burst_size_bytes: 50
  "test-user:source=y":
    ingestion:
      burst_size_bytes: 20
  "*":
    ingestion:
      rate_limit_bytes: 1
      burst_size_bytes: 1
`

	tcs := []struct {
		name       string
		userID     string
		limitsKey  string
		tracesSize int
		wantErr    string
	}{
		{
			name:       "metadata without an override inherits the tenant limits",
			userID:     "test-user",
			limitsKey:  "test-user:source=z",
			tracesSize: 500,
		},
		{
			name:       "metadata placeholder override applies to the tenant",
			userID:     "test-user",
			limitsKey:  "test-user:source=x",
			tracesSize: 100,
			wantErr:    "RATE_LIMITED: batch size (100 bytes) exceeds ingestion limit (local: 50 bytes/s, global: 0 bytes/s, burst: 50 bytes) while adding 100 bytes for user test-user (limits key test-user:source=x). consider reducing batch size or increasing rate limit.",
		},
		{
			name:       "tenant metadata override overlays only the fields it sets",
			userID:     "test-user",
			limitsKey:  "test-user:source=y",
			tracesSize: 100,
			wantErr:    "RATE_LIMITED: ingestion rate limit (local: 1000 bytes/s, global: 0 bytes/s, burst: 20 bytes) exceeded while adding 100 bytes for user test-user (limits key test-user:source=y). consider increasing the limit or reducing ingestion rate.",
		},
		{
			name:       "wildcard is the base for tenants without an entry",
			userID:     "other",
			limitsKey:  "other:source=z",
			tracesSize: 100,
			wantErr:    "RATE_LIMITED: batch size (100 bytes) exceeds ingestion limit (local: 1 bytes/s, global: 0 bytes/s, burst: 1 bytes) while adding 100 bytes for user other (limits key other:source=z). consider reducing batch size or increasing rate limit.",
		},
		{
			name:       "metadata placeholder override applies on top of the wildcard",
			userID:     "other",
			limitsKey:  "other:source=x",
			tracesSize: 100,
			wantErr:    "RATE_LIMITED: batch size (100 bytes) exceeds ingestion limit (local: 50 bytes/s, global: 0 bytes/s, burst: 50 bytes) while adding 100 bytes for user other (limits key other:source=x). consider reducing batch size or increasing rate limit.",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			d := prepareWithPerTenantOverrides(t, tenantYAML)

			err := d.checkForRateLimits(tc.tracesSize, 10, tc.userID, tc.limitsKey)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			s, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, codes.ResourceExhausted, s.Code())
			require.Equal(t, tc.wantErr, s.Message())
		})
	}

	t.Run("each limits key has its own token bucket", func(t *testing.T) {
		d := prepareWithPerTenantOverrides(t, tenantYAML)

		// drain the metadata key's bucket
		require.NoError(t, d.checkForRateLimits(1000, 10, "test-user", "test-user:source=z"))
		require.Error(t, d.checkForRateLimits(1000, 10, "test-user", "test-user:source=z"))

		// the tenant's own bucket and another metadata key's bucket are untouched
		require.NoError(t, d.checkForRateLimits(1000, 10, "test-user", "test-user"))
		require.NoError(t, d.checkForRateLimits(1000, 10, "test-user", "test-user:source=w"))
	})
}

func TestRateLimitRespected_TenantMetadata(t *testing.T) {
	overridesConfig := overrides.Config{
		Defaults: overrides.Overrides{
			Ingestion: overrides.IngestionOverrides{
				RateStrategy:   overrides.LocalIngestionRateStrategy,
				RateLimitBytes: 400,
				BurstSizeBytes: 200,
			},
		},
	}
	d := prepare(t, overridesConfig, nil)
	traces := batchesToTraces(t, []*v1.ResourceSpans{test.MakeBatch(10, nil)})

	t.Run("rate limited pushes are recorded against the tenant", func(t *testing.T) {
		before := discardedSpans(t, overrides.ReasonRateLimited, "test")

		_, err := d.PushTraces(user.InjectOrgID(context.Background(), "test:source=x"), traces)
		s, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.ResourceExhausted, s.Code())
		require.Contains(t, s.Message(), "for user test (limits key test:source=x)")

		require.Equal(t, before+10, discardedSpans(t, overrides.ReasonRateLimited, "test"))
	})

	t.Run("malformed metadata is rejected", func(t *testing.T) {
		_, err := d.PushTraces(user.InjectOrgID(context.Background(), "test:garbage"), traces)
		s, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.InvalidArgument, s.Code())
	})
}

func TestRetryInfoEnabled_TenantMetadata(t *testing.T) {
	d := prepareWithPerTenantOverrides(t, "overrides:\n  test-tenant:\n    ingestion:\n      retry_info_enabled: false\n")
	d.cfg.RetryAfterOnResourceExhausted = 10 * time.Second

	// the tenant override is found even when the org ID carries metadata
	enabled, err := d.RetryInfoEnabled(user.InjectOrgID(context.Background(), "test-tenant:source=x"))
	require.NoError(t, err)
	require.False(t, enabled)

	// defaults apply to other tenants
	enabled, err = d.RetryInfoEnabled(user.InjectOrgID(context.Background(), "other:source=x"))
	require.NoError(t, err)
	require.True(t, enabled)
}

func TestRemoveStaleRateLimiters(t *testing.T) {
	o, err := overrides.NewOverrides(overrides.Config{
		Defaults: overrides.Overrides{
			Ingestion: overrides.IngestionOverrides{
				RateStrategy:   overrides.LocalIngestionRateStrategy,
				RateLimitBytes: 100,
				BurstSizeBytes: 100,
			},
		},
	}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	l := limiter.NewRateLimiter(newLocalIngestionRateStrategy(o), 10*time.Second)

	now := time.Now()
	require.True(t, l.AllowN(now, "a", 1))
	require.True(t, l.AllowN(now, "a:source=x", 1))

	// nothing is stale yet
	require.Equal(t, 0, removeStaleRateLimiters(l, now.Add(time.Hour)))

	// one key is used again, the other goes stale
	later := now.Add(ingestionRateLimiterStaleness)
	require.True(t, l.AllowN(later, "a", 1))
	require.Equal(t, 1, removeStaleRateLimiters(l, later.Add(time.Hour)))

	// eventually both are removed
	require.Equal(t, 1, removeStaleRateLimiters(l, later.Add(ingestionRateLimiterStaleness+time.Hour)))
}
