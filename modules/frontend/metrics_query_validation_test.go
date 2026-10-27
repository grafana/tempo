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
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
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
			wantResult:      scannedBytesResultRejectedMaxDuration,
			wantCalls:       1,
			wantLog: `level=info msg="scanned bytes limit" tenant=foo op=metrics requested_seconds=28800 estimated_scanned_bytes=900 max_scanned_bytes=350 derived_seconds=10799 allowed_seconds=10799 min_seconds=0 max_seconds=21600 result=rejected_max_duration passes_scanned_bytes=false
`,
		},
		{
			// The range limits replace the upstream max_duration check, which ran before the step check.
			name:            "step error comes after the range limits",
			start:           hoursAgo(2),
			maxScannedBytes: 350,
			maxIntervals:    10,
			wantErr:         "step of 1m0s is too small, minimum step for given range is 12m0s",
			wantResult:      scannedBytesResultWithin,
			wantCalls:       1,
			wantLog: `level=info msg="scanned bytes limit" tenant=foo op=metrics requested_seconds=7200 estimated_scanned_bytes=300 max_scanned_bytes=350 fits=true min_seconds=0 max_seconds=0 result=within passes_scanned_bytes=true
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
			rl := newMetricsRangeLimiter(cfg, newTestOverrides(t, overrides.ReadOverrides{}), rdr, log.NewLogfmtLogger(&buf))

			countBefore := resultCount("foo", metricsOp, tc.wantResult)
			err := validateQueryRangeReq(user.InjectOrgID(context.Background(), "foo"), cfg, rl, &tempopb.QueryRangeRequest{
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
				require.Equal(t, 1.0, resultCount("foo", metricsOp, tc.wantResult)-countBefore)
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
		{name: "no org ID", noOrgID: true, start: 1, end: 2, wantErr: "no org id"},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{}
			cfg.Metrics.Sharder.MaxDuration = time.Hour
			rl := newMetricsRangeLimiter(cfg, newTestOverrides(t, overrides.ReadOverrides{}), &mockReader{}, log.NewNopLogger())
			ctx := user.InjectOrgID(context.Background(), "foo")
			if tc.noOrgID {
				ctx = context.Background()
			}
			err := validateQueryRangeReq(ctx, cfg, rl, &tempopb.QueryRangeRequest{Start: tc.start, End: tc.end, Step: uint64(time.Minute)})
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
		// wantSpanResult is the span's result attribute, empty when the span has none.
		wantSpanResult string
	}{
		{
			// tenant.TenantIDs sorts, so the over-budget tenant comes first and the fitting one is checked after the rejection.
			name:    "every tenant is logged",
			orgID:   "b-fits|a-over",
			cfg:     QueryRangeSharderConfig{MaxScannedBytes: 350},
			wantErr: "metrics query time range of 4h0m0s exceeds the maximum allowed range of 2h59m59s for tenant a-over (limited by scanned bytes), reduce the time range to 2h59m59s or less",
			wantLog: `level=info msg="scanned bytes limit" tenant=a-over op=metrics requested_seconds=14400 estimated_scanned_bytes=500 max_scanned_bytes=350 derived_seconds=10799 allowed_seconds=10799 min_seconds=0 max_seconds=0 result=rejected passes_scanned_bytes=false
level=info msg="scanned bytes limit" tenant=b-fits op=metrics requested_seconds=14400 estimated_scanned_bytes=0 max_scanned_bytes=350 fits=true min_seconds=0 max_seconds=0 result=within passes_scanned_bytes=true
`,
			wantSpanResult: scannedBytesResultRejected,
		},
		{
			name:    "every tenant over its max_duration is logged",
			orgID:   "b-fits|a-over",
			cfg:     QueryRangeSharderConfig{MaxScannedBytes: 350, MaxDuration: time.Hour},
			wantErr: "metrics query time range exceeds the maximum allowed duration of 1h0m0s",
			wantLog: `level=info msg="scanned bytes limit" tenant=a-over op=metrics requested_seconds=14400 estimated_scanned_bytes=500 max_scanned_bytes=350 derived_seconds=10799 allowed_seconds=3600 min_seconds=0 max_seconds=3600 result=rejected_max_duration passes_scanned_bytes=false
level=info msg="scanned bytes limit" tenant=b-fits op=metrics requested_seconds=14400 estimated_scanned_bytes=0 max_scanned_bytes=350 fits=true min_seconds=0 max_seconds=3600 result=rejected_max_duration passes_scanned_bytes=true
`,
			wantSpanResult: scannedBytesResultRejectedMaxDuration,
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
			wantErr: "metrics query time range exceeds the maximum allowed duration of 1h0m0s",
		},
		{
			name:           "of 2 equal rejections the first wins",
			orgID:          "c-over|a-over",
			cfg:            QueryRangeSharderConfig{MaxScannedBytes: 350},
			wantErr:        "metrics query time range of 4h0m0s exceeds the maximum allowed range of 2h59m59s for tenant a-over (limited by scanned bytes), reduce the time range to 2h59m59s or less",
			wantSpanResult: scannedBytesResultRejected,
		},
		{
			name:           "a rejection after a fit decides the span",
			orgID:          "c-over|b-fits",
			cfg:            QueryRangeSharderConfig{MaxScannedBytes: 350},
			wantErr:        "metrics query time range of 4h0m0s exceeds the maximum allowed range of 2h59m59s for tenant c-over (limited by scanned bytes), reduce the time range to 2h59m59s or less",
			wantSpanResult: scannedBytesResultRejected,
		},
		{
			// a-over is checked first and allows 2h59m59s, which e-dense would reject on retry.
			name:           "the rejection with the shortest retry range wins",
			orgID:          "e-dense|a-over",
			cfg:            QueryRangeSharderConfig{MaxScannedBytes: 350},
			wantErr:        "metrics query time range of 4h0m0s exceeds the maximum allowed range of 59m59s for tenant e-dense (limited by scanned bytes), reduce the time range to 59m59s or less",
			wantSpanResult: scannedBytesResultRejected,
		},
		{
			name:  "a retry with the reported range passes every tenant",
			orgID: "e-dense|a-over",
			cfg:   QueryRangeSharderConfig{MaxScannedBytes: 350},
			start: uint64(scannedBytesTestEnd.Add(-(time.Hour - time.Second)).UnixNano()),
			// Both fit, so the span shows the first tenant's check.
			wantSpanResult: scannedBytesResultWithin,
		},
		{
			name:  "a max_duration below the other tenant's allowed range wins",
			orgID: "b-fits|a-over",
			cfg:   QueryRangeSharderConfig{MaxScannedBytes: 350},
			perTenant: `overrides:
  b-fits:
    read:
      max_metrics_duration: 1h
`,
			wantErr:        "metrics query time range exceeds the maximum allowed duration of 1h0m0s",
			wantSpanResult: scannedBytesResultRejectedMaxDuration,
		},
		{
			name:  "an allowed range below the other tenant's max_duration wins",
			orgID: "b-fits|e-dense",
			cfg:   QueryRangeSharderConfig{MaxScannedBytes: 350},
			perTenant: `overrides:
  b-fits:
    read:
      max_metrics_duration: 2h
`,
			wantErr:        "metrics query time range of 4h0m0s exceeds the maximum allowed range of 59m59s for tenant e-dense (limited by scanned bytes), reduce the time range to 59m59s or less",
			wantSpanResult: scannedBytesResultRejected,
		},
		{
			name:           "tenants that all fit put the first check on the span",
			orgID:          "b-fits|d-fits",
			cfg:            QueryRangeSharderConfig{MaxScannedBytes: 350},
			wantSpanResult: scannedBytesResultWithin,
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
		},
		{
			name: "no limits allow any range",
		},
		{
			// Each override must replace its config value.
			name:  "overrides beat the config",
			orgID: "mixed",
			cfg:   QueryRangeSharderConfig{MaxDuration: 24 * time.Hour, MinDuration: time.Hour, MaxScannedBytes: 10_000},
			overrides: overrides.ReadOverrides{
				MaxMetricsDuration: model.Duration(6 * time.Hour), MinMetricsDuration: model.Duration(5 * time.Hour), MaxMetricsScannedBytes: 150,
			},
			start:          uint64(hoursAgo(6).UnixNano()),
			wantErr:        "metrics query time range of 6h0m0s exceeds the maximum allowed range of 5h0m0s for tenant mixed (limited by scanned bytes), reduce the time range to 5h0m0s or less",
			wantSpanResult: scannedBytesResultRejected,
			wantLog: `level=info msg="scanned bytes limit" tenant=mixed op=metrics requested_seconds=21600 estimated_scanned_bytes=700 max_scanned_bytes=150 derived_seconds=3599 allowed_seconds=18000 min_seconds=18000 max_seconds=21600 result=rejected passes_scanned_bytes=false
`,
		},
		{
			// A range past math.MaxInt64 nanoseconds must not wrap negative and slip under the allowed range.
			name:           "huge range",
			cfg:            QueryRangeSharderConfig{MaxScannedBytes: 1},
			start:          1,
			end:            math.MaxUint64,
			wantErr:        "metrics query time range of 2562047h47m16s exceeds the maximum allowed range of 2561047h47m16s for tenant foo (limited by scanned bytes), reduce the time range to 2561047h47m16s or less",
			wantSpanResult: scannedBytesResultRejected,
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
			l := newMetricsRangeLimiter(frontendCfg, o, &perTenantMockReader{metas: metas}, log.NewLogfmtLogger(&buf))
			start, end := unixNanoTime(cmp.Or(tc.start, uint64(hoursAgo(4).UnixNano()))), unixNanoTime(cmp.Or(tc.end, uint64(scannedBytesTestEnd.UnixNano())))
			rec := tracetest.NewSpanRecorder()
			ctx, span := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec)).Tracer("test").Start(context.Background(), "test")

			err := l.Limit(ctx, cmp.Or(tc.orgID, "foo"), start, end)
			span.End()
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
			var spanResult string
			for _, a := range rec.Ended()[0].Attributes() {
				if a.Key == "result" {
					spanResult = a.Value.AsString()
				}
			}
			require.Equal(t, tc.wantSpanResult, spanResult)
		})
	}
}

func TestMetricsRangeLimiterCutoff(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// One large block straddles the cutoff at now-30m.
		now := time.Now()
		rdr := &mockReader{metas: []*backend.BlockMeta{{
			StartTime:         now.Add(-40 * time.Minute),
			EndTime:           now.Add(-time.Minute),
			Size_:             1000,
			ReplicationFactor: backend.MetricsGeneratorReplicationFactor,
		}}}
		cfg := Config{}
		cfg.Metrics.Sharder = QueryRangeSharderConfig{MaxScannedBytes: 500, QueryBackendAfter: 30 * time.Minute}
		l := newMetricsRangeLimiter(cfg, newTestOverrides(t, overrides.ReadOverrides{}), rdr, log.NewNopLogger())

		// The 28m between the cutoff and the end reach no backend block, so they are allowed.
		err := l.Limit(context.Background(), "foo", now.Add(-time.Hour), now.Add(-2*time.Minute))
		require.EqualError(t, err, "metrics query time range of 58m0s exceeds the maximum allowed range of 28m0s for tenant foo (limited by scanned bytes), reduce the time range to 28m0s or less")
	})
}
