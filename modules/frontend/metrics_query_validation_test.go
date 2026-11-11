package frontend

import (
	"bytes"
	"cmp"
	"context"
	"math"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/go-kit/log" //nolint:all deprecated
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/v3/modules/overrides"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"
)

func TestValidateQueryRangeReqScannedBytesLimit(t *testing.T) {
	tcs := []struct {
		name            string
		start           time.Time
		maxScannedBytes uint64
		maxDuration     time.Duration
		maxIntervals    uint64
		wantErr         string
		wantResult      string
		wantLog         string
		wantCalls       int
	}{
		{
			name:            "over max_duration with a too small step keeps the max_duration error",
			start:           hoursAgo(8),
			maxScannedBytes: 350,
			maxDuration:     6 * time.Hour,
			maxIntervals:    10,
			wantErr:         "metrics query time range exceeds the maximum allowed duration of 6h0m0s",
			wantResult:      queryRejectedMaxDuration,
			wantCalls:       1,
			wantLog: `level=info msg="scanned bytes limit" tenant=foo op=metrics max_duration=21600 min_duration=0 max_scanned_bytes=350 requested_seconds=28800 estimated_scanned_bytes=900 estimated_range_seconds=0 result=rejected_max_duration
`,
		},
		{
			// The range limits replace the upstream max_duration check, which ran before the step check.
			name:            "step error comes after the range limits",
			start:           hoursAgo(2),
			maxScannedBytes: 350,
			maxIntervals:    10,
			wantErr:         "step of 1m0s is too small, minimum step for given range is 12m0s",
			wantResult:      queryAllowed,
			wantCalls:       1,
			wantLog: `level=info msg="scanned bytes limit" tenant=foo op=metrics max_duration=0 min_duration=0 max_scanned_bytes=350 requested_seconds=7200 estimated_scanned_bytes=300 estimated_range_seconds=0 result=allowed
`,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			cfg := Config{}
			cfg.Metrics.MaxIntervals = tc.maxIntervals
			cfg.Metrics.Sharder = QueryRangeSharderConfig{MaxDuration: tc.maxDuration, MaxScannedBytes: tc.maxScannedBytes}
			rdr := &countingMockReader{mockReader: mockReader{metas: hourlyMetricsBlocks(scannedBytesTestEnd)}}
			rl := newMetricsRangeLimiter(cfg, newTestOverrides(t, overrides.ReadOverrides{}), rdr)

			countBefore := queriesCount(metricsOp, tc.wantResult, "true")
			err := validateQueryRangeReq(user.InjectOrgID(context.Background(), "foo"), cfg, rl, log.NewLogfmtLogger(&buf), &tempopb.QueryRangeRequest{
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
			require.Equal(t, tc.wantLog, buf.String())
			if tc.wantResult != "" {
				require.Equal(t, 1.0, queriesCount(metricsOp, tc.wantResult, "true")-countBefore)
			}
		})
	}
}

func TestValidateQueryRangeReqRange(t *testing.T) {
	tcs := []struct {
		name       string
		noOrgID    bool
		start, end uint64
		wantErr    string
	}{
		{name: "start equal to end", start: 1, end: 1, wantErr: errEndMustBeGreaterThanStart.Error()},
		// An end past math.MaxInt64 nanoseconds must not wrap negative and slip under max_duration.
		{name: "huge end over max_duration", start: 1, end: math.MaxUint64, wantErr: "metrics query time range exceeds the maximum allowed duration of 1h0m0s"},
		// Both timestamps past math.MaxInt64 nanoseconds must not clamp to one instant and read as a 0s range.
		{name: "huge start and end over max_duration", start: math.MaxInt64, end: math.MaxUint64, wantErr: "metrics query time range exceeds the maximum allowed duration of 1h0m0s"},
		{name: "no org ID", noOrgID: true, start: 1, end: 2, wantErr: "no org id"},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{}
			cfg.Metrics.Sharder.MaxDuration = time.Hour
			rl := newMetricsRangeLimiter(cfg, newTestOverrides(t, overrides.ReadOverrides{}), &mockReader{})
			ctx := user.InjectOrgID(context.Background(), "foo")
			if tc.noOrgID {
				ctx = context.Background()
			}
			err := validateQueryRangeReq(ctx, cfg, rl, log.NewNopLogger(), &tempopb.QueryRangeRequest{Start: tc.start, End: tc.end, Step: uint64(time.Minute)})
			require.EqualError(t, err, tc.wantErr)
		})
	}
}

func TestMetricsRangeLimiter(t *testing.T) {
	metas := map[string][]*backend.BlockMeta{
		"foo":    hourlyMetricsBlocks(scannedBytesTestEnd),
		"a-over": hourlyMetricsBlocks(scannedBytesTestEnd),
		"b-fits": nil,
		"c-over": hourlyMetricsBlocks(scannedBytesTestEnd),
		"mixed":  append(hourlyMetricsBlocks(scannedBytesTestEnd), nonRF1Block),
		"e-dense": func() []*backend.BlockMeta {
			blocks := hourlyMetricsBlocks(scannedBytesTestEnd)
			for _, b := range blocks {
				b.Size_ = 200
			}
			return blocks
		}(),
	}

	tcs := []struct {
		name      string
		orgID     string
		cfg       QueryRangeSharderConfig
		overrides overrides.ReadOverrides
		// perTenant is a per-tenant overrides file, used instead of overrides.
		perTenant                  string
		multiTenantQueriesDisabled bool
		// start defaults to 4h before scannedBytesTestEnd.
		start, end uint64
		wantErr    string
		wantLog    string
		// wantSpans are the checked tenants' span results.
		wantSpans map[string]string
	}{
		{
			// tenant.TenantIDs sorts, so the over-limit tenant comes first and the fitting one is checked after the rejection.
			name:    "every tenant is logged",
			orgID:   "b-fits|a-over",
			cfg:     QueryRangeSharderConfig{MaxScannedBytes: 350},
			wantErr: "metrics query time range of 4h0m0s reads too much data, the allowed range for tenant a-over is 2h59m59s",
			wantLog: `level=info msg="scanned bytes limit" tenant=a-over op=metrics max_duration=0 min_duration=0 max_scanned_bytes=350 requested_seconds=14400 estimated_scanned_bytes=500 estimated_range_seconds=10799 result=rejected_scanned_bytes
level=info msg="scanned bytes limit" tenant=b-fits op=metrics max_duration=0 min_duration=0 max_scanned_bytes=350 requested_seconds=14400 estimated_scanned_bytes=0 estimated_range_seconds=0 result=allowed
`,
			wantSpans: map[string]string{"a-over": queryRejectedBytes, "b-fits": queryAllowed},
		},
		{
			name:    "every tenant over its max_duration is logged",
			orgID:   "b-fits|a-over",
			cfg:     QueryRangeSharderConfig{MaxScannedBytes: 350, MaxDuration: time.Hour},
			wantErr: "metrics query time range exceeds the maximum allowed duration of 1h0m0s\nmetrics query time range exceeds the maximum allowed duration of 1h0m0s",
			wantLog: `level=info msg="scanned bytes limit" tenant=a-over op=metrics max_duration=3600 min_duration=0 max_scanned_bytes=350 requested_seconds=14400 estimated_scanned_bytes=500 estimated_range_seconds=0 result=rejected_max_duration
level=info msg="scanned bytes limit" tenant=b-fits op=metrics max_duration=3600 min_duration=0 max_scanned_bytes=350 requested_seconds=14400 estimated_scanned_bytes=0 estimated_range_seconds=0 result=rejected_max_duration
`,
			wantSpans: map[string]string{"a-over": queryRejectedMaxDuration, "b-fits": queryRejectedMaxDuration},
		},
		{
			name:  "each tenant gets its own max_duration override",
			orgID: "b|a",
			cfg:   QueryRangeSharderConfig{MaxDuration: 24 * time.Hour},
			perTenant: `overrides:
  a:
    read:
      max_metrics_duration: 48h
  b:
    read:
      max_metrics_duration: 1h
`,
			wantErr:   "metrics query time range exceeds the maximum allowed duration of 1h0m0s",
			wantSpans: map[string]string{"a": "", "b": ""},
		},
		{
			name:      "every rejection is returned",
			orgID:     "c-over|a-over",
			cfg:       QueryRangeSharderConfig{MaxScannedBytes: 350},
			wantErr:   "metrics query time range of 4h0m0s reads too much data, the allowed range for tenant a-over is 2h59m59s\nmetrics query time range of 4h0m0s reads too much data, the allowed range for tenant c-over is 2h59m59s",
			wantSpans: map[string]string{"a-over": queryRejectedBytes, "c-over": queryRejectedBytes},
		},
		{
			name:      "a rejection after a fit is returned",
			orgID:     "c-over|b-fits",
			cfg:       QueryRangeSharderConfig{MaxScannedBytes: 350},
			wantErr:   "metrics query time range of 4h0m0s reads too much data, the allowed range for tenant c-over is 2h59m59s",
			wantSpans: map[string]string{"b-fits": queryAllowed, "c-over": queryRejectedBytes},
		},
		{
			// a-over allows 2h59m59s, which e-dense would reject, so both ranges are reported.
			name:      "each tenant's allowed range is returned",
			orgID:     "e-dense|a-over",
			cfg:       QueryRangeSharderConfig{MaxScannedBytes: 350},
			wantErr:   "metrics query time range of 4h0m0s reads too much data, the allowed range for tenant a-over is 2h59m59s\nmetrics query time range of 4h0m0s reads too much data, the allowed range for tenant e-dense is 59m59s",
			wantSpans: map[string]string{"a-over": queryRejectedBytes, "e-dense": queryRejectedBytes},
		},
		{
			name:      "a retry with the reported range passes every tenant",
			orgID:     "e-dense|a-over",
			cfg:       QueryRangeSharderConfig{MaxScannedBytes: 350},
			start:     uint64(scannedBytesTestEnd.Add(-(time.Hour - time.Second)).UnixNano()),
			wantSpans: map[string]string{"a-over": queryAllowed, "e-dense": queryAllowed},
		},
		{
			name:  "a max_duration rejection and a scanned bytes rejection are both returned",
			orgID: "b-fits|a-over",
			cfg:   QueryRangeSharderConfig{MaxScannedBytes: 350},
			perTenant: `overrides:
  b-fits:
    read:
      max_metrics_duration: 1h
`,
			wantErr:   "metrics query time range of 4h0m0s reads too much data, the allowed range for tenant a-over is 2h59m59s\nmetrics query time range exceeds the maximum allowed duration of 1h0m0s",
			wantSpans: map[string]string{"a-over": queryRejectedBytes, "b-fits": queryRejectedMaxDuration},
		},
		{
			name:  "a scanned bytes rejection after a max_duration rejection is returned too",
			orgID: "b-fits|e-dense",
			cfg:   QueryRangeSharderConfig{MaxScannedBytes: 350},
			perTenant: `overrides:
  b-fits:
    read:
      max_metrics_duration: 2h
`,
			wantErr:   "metrics query time range exceeds the maximum allowed duration of 2h0m0s\nmetrics query time range of 4h0m0s reads too much data, the allowed range for tenant e-dense is 59m59s",
			wantSpans: map[string]string{"b-fits": queryRejectedMaxDuration, "e-dense": queryRejectedBytes},
		},
		{
			// a-over is rejected without a check, so its span has no result.
			name:  "a max_duration rejection without a check has a span without a result",
			orgID: "b-fits|a-over",
			perTenant: `overrides:
  a-over:
    read:
      max_metrics_duration: 1h
  b-fits:
    read:
      max_metrics_scanned_bytes: 350
`,
			wantErr:   "metrics query time range exceeds the maximum allowed duration of 1h0m0s",
			wantSpans: map[string]string{"a-over": "", "b-fits": queryAllowed},
		},
		{
			name:      "every checked tenant gets a span",
			orgID:     "b-fits|d-fits",
			cfg:       QueryRangeSharderConfig{MaxScannedBytes: 350},
			wantSpans: map[string]string{"b-fits": queryAllowed, "d-fits": queryAllowed},
		},
		{
			name:    "invalid org ID",
			orgID:   "foo bar",
			cfg:     QueryRangeSharderConfig{MaxScannedBytes: 350},
			wantErr: `tenant ID 'foo bar' contains unsupported character ' '`,
		},
		{
			// The tenant validator rejects a|b later, so no tenant is checked or logged here.
			name:                       "multi-tenant queries disabled",
			orgID:                      "b-fits|a-over",
			cfg:                        QueryRangeSharderConfig{MaxScannedBytes: 350},
			multiTenantQueriesDisabled: true,
		},
		{
			name:      "max_duration override above the range allows it",
			cfg:       QueryRangeSharderConfig{MaxDuration: time.Hour},
			overrides: overrides.ReadOverrides{MaxMetricsDuration: model.Duration(48 * time.Hour)},
			wantSpans: map[string]string{"foo": ""},
		},
		{
			name:      "no limits allow any range",
			wantSpans: map[string]string{"foo": ""},
		},
		{
			// Each override must replace its config value.
			name:  "overrides beat the config",
			orgID: "mixed",
			cfg:   QueryRangeSharderConfig{MaxDuration: 24 * time.Hour, MinDuration: time.Hour, MaxScannedBytes: 10_000},
			overrides: overrides.ReadOverrides{
				MaxMetricsDuration: model.Duration(6 * time.Hour), MinMetricsDuration: model.Duration(5 * time.Hour), MaxMetricsScannedBytes: 150,
			},
			start:     uint64(hoursAgo(6).UnixNano()),
			wantErr:   "metrics query time range of 6h0m0s reads too much data, the allowed range for tenant mixed is 5h0m0s",
			wantSpans: map[string]string{"mixed": queryRejectedBytes},
			wantLog: `level=info msg="scanned bytes limit" tenant=mixed op=metrics max_duration=21600 min_duration=18000 max_scanned_bytes=150 requested_seconds=21600 estimated_scanned_bytes=700 estimated_range_seconds=18000 result=rejected_scanned_bytes
`,
		},
		{
			// A range past math.MaxInt64 nanoseconds must not wrap negative and slip under the allowed range.
			name:      "huge range",
			cfg:       QueryRangeSharderConfig{MaxScannedBytes: 1},
			start:     1,
			end:       math.MaxUint64,
			wantErr:   "metrics query time range of 2562047h47m16s reads too much data, the allowed range for tenant foo is 2562047h47m16s",
			wantSpans: map[string]string{"foo": queryRejectedBytes},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			frontendCfg := Config{MultiTenantQueriesEnabled: !tc.multiTenantQueriesDisabled}
			frontendCfg.Metrics.Sharder = tc.cfg
			o := newTestOverrides(t, tc.overrides)
			if tc.perTenant != "" {
				o = newTestPerTenantOverrides(t, tc.perTenant)
			}
			l := newMetricsRangeLimiter(frontendCfg, o, &perTenantMockReader{metas: metas})
			start, end := unixNanoTime(cmp.Or(tc.start, uint64(hoursAgo(4).UnixNano()))), unixNanoTime(cmp.Or(tc.end, uint64(scannedBytesTestEnd.UnixNano())))
			ctx, spans := recordCheckRangeSpans()

			err := checkMetricsRange(ctx, frontendCfg, l, log.NewLogfmtLogger(&buf), cmp.Or(tc.orgID, "foo"), start, end)
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tc.wantErr)
			}
			if strings.Contains(tc.wantErr, "maximum allowed duration") {
				require.ErrorIs(t, err, errMetricsMaxDuration)
			}
			if tc.wantLog != "" {
				require.Equal(t, tc.wantLog, buf.String())
			}
			wantSpans := tc.wantSpans
			if wantSpans == nil {
				wantSpans = map[string]string{}
			}
			require.Equal(t, wantSpans, spans())
		})
	}
}

func TestMetricsRangeLimiterCutoff(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// One large block straddles backendAfter at now-30m.
		now := time.Now()
		rdr := &mockReader{metas: []*backend.BlockMeta{{
			StartTime:         now.Add(-40 * time.Minute),
			EndTime:           now.Add(-time.Minute),
			Size_:             1000,
			ReplicationFactor: backend.MetricsGeneratorReplicationFactor,
		}}}
		cfg := Config{}
		cfg.Metrics.Sharder = QueryRangeSharderConfig{MaxScannedBytes: 500, QueryBackendAfter: 30 * time.Minute}
		l := newMetricsRangeLimiter(cfg, newTestOverrides(t, overrides.ReadOverrides{}), rdr)

		// The 28m between backendAfter and the end reach no backend block, so they are allowed.
		err := checkMetricsRange(context.Background(), cfg, l, log.NewNopLogger(), "foo", now.Add(-time.Hour), now.Add(-2*time.Minute))
		require.EqualError(t, err, "metrics query time range of 58m0s reads too much data, the allowed range for tenant foo is 28m0s")
	})
}
