package frontend

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/go-kit/log" //nolint:all deprecated
	"github.com/grafana/dskit/user"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/grafana/tempo/v3/modules/overrides"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/blocklist"
)

var scannedBytesTestEnd = time.Unix(1000*3600, 0)

func hoursAgo(h int) time.Time {
	return scannedBytesTestEnd.Add(-time.Duration(h) * time.Hour)
}

// testBlock returns an RF1 block spanning the given hours before scannedBytesTestEnd.
func testBlock(startHoursAgo, endHoursAgo int, size uint64) *backend.BlockMeta {
	return &backend.BlockMeta{
		StartTime:         hoursAgo(startHoursAgo),
		EndTime:           hoursAgo(endHoursAgo),
		Size_:             size,
		ReplicationFactor: backend.MetricsGeneratorReplicationFactor,
	}
}

// hourlyMetricsBlocks returns 10 RF1 blocks of 100 bytes, one per hour, ending at end.
func hourlyMetricsBlocks(end time.Time) []*backend.BlockMeta {
	metas := make([]*backend.BlockMeta, 0, 10)
	for i := range 10 {
		metas = append(metas, &backend.BlockMeta{
			StartTime:         end.Add(-time.Duration(i+1) * time.Hour),
			EndTime:           end.Add(-time.Duration(i) * time.Hour),
			Size_:             100,
			TotalRecords:      1,
			BlockID:           backend.MustParse(fmt.Sprintf("00000000-0000-0000-0000-%012d", i)),
			ReplicationFactor: backend.MetricsGeneratorReplicationFactor,
		})
	}
	return metas
}

type countingMockReader struct {
	mockReader
	mtx   sync.Mutex
	calls int
}

func (m *countingMockReader) BlockMetas(tenantID string) []*backend.BlockMeta {
	m.mtx.Lock()
	m.calls++
	m.mtx.Unlock()
	return m.mockReader.BlockMetas(tenantID)
}

func (m *countingMockReader) callCount() int {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	return m.calls
}

type perTenantMockReader struct {
	mockReader
	metas map[string][]*backend.BlockMeta
}

func (m *perTenantMockReader) BlockMetas(tenantID string) []*backend.BlockMeta {
	return m.metas[tenantID]
}

// mockOverridesScannedBytes returns the same limit overrides for every tenant.
type mockOverridesScannedBytes struct {
	overrides.Interface
	metrics, search scannedBytesConfig
}

func (m *mockOverridesScannedBytes) MaxMetricsDuration(string) time.Duration {
	return m.metrics.maxDuration
}
func (m *mockOverridesScannedBytes) MinMetricsDuration(string) time.Duration {
	return m.metrics.minDuration
}
func (m *mockOverridesScannedBytes) MaxMetricsScannedBytes(string) uint64 {
	return m.metrics.maxScannedBytes
}
func (m *mockOverridesScannedBytes) MaxSearchDuration(string) time.Duration {
	return m.search.maxDuration
}
func (m *mockOverridesScannedBytes) MinSearchDuration(string) time.Duration {
	return m.search.minDuration
}
func (m *mockOverridesScannedBytes) MaxSearchScannedBytes(string) uint64 {
	return m.search.maxScannedBytes
}

func histogramSample(t *testing.T, tenant, op string) (count uint64, sum float64) {
	m := &dto.Metric{}
	require.NoError(t, scannedBytesLimitDerivedRange.WithLabelValues(tenant, op).(prometheus.Histogram).Write(m))
	return m.GetHistogram().GetSampleCount(), m.GetHistogram().GetSampleSum()
}

func outcomeCount(tenant, op, outcome string) float64 {
	return testutil.ToFloat64(scannedBytesLimitTotal.WithLabelValues(tenant, op, outcome))
}

func TestEvaluateScannedBytesLimit(t *testing.T) {
	hourly := make([]*backend.BlockMeta, 0, 10)
	for i := range 10 {
		hourly = append(hourly, testBlock(i+1, i, 100))
	}
	nonRF1 := &backend.BlockMeta{StartTime: hoursAgo(1), EndTime: scannedBytesTestEnd, Size_: 10_000}

	tcs := []struct {
		name      string
		op        string
		metas     []*backend.BlockMeta
		cfg       scannedBytesConfig
		overrides scannedBytesConfig
		// start defaults to scannedBytesTestEnd minus requested. cutoff defaults to scannedBytesTestEnd.
		start     time.Time
		cutoff    time.Time
		requested time.Duration
		want      scannedBytesDecision
		wantErr   string
		wantCalls int
	}{
		{
			name:      "limit off allows any range",
			metas:     hourly,
			cfg:       scannedBytesConfig{maxDuration: 24 * time.Hour},
			requested: 8 * time.Hour,
			want:      scannedBytesDecision{outcome: scannedBytesOutcomeWithin, maxDuration: 24 * time.Hour},
		},
		{
			name:      "limit off still rejects over max_duration",
			op:        searchOp,
			metas:     hourly,
			cfg:       scannedBytesConfig{maxDuration: 5 * time.Hour},
			requested: 6 * time.Hour,
			want:      scannedBytesDecision{outcome: scannedBytesOutcomeRejectedMaxDuration, maxDuration: 5 * time.Hour},
		},
		{
			name:      "full fit",
			metas:     hourly,
			cfg:       scannedBytesConfig{maxDuration: 24 * time.Hour, maxScannedBytes: 1000},
			requested: 10 * time.Hour,
			want:      scannedBytesDecision{outcome: scannedBytesOutcomeWithin, evaluated: true, allowed: 24 * time.Hour, scannedBytes: 1000, maxDuration: 24 * time.Hour, maxScannedBytes: 1000},
			wantCalls: 1,
		},
		{
			// The design doc sketch returned the 3h of data here and rejected a query that fits.
			name:      "data starts mid-range",
			metas:     hourly[:3],
			cfg:       scannedBytesConfig{maxDuration: 7 * 24 * time.Hour, maxScannedBytes: 1000},
			requested: 7 * 24 * time.Hour,
			want:      scannedBytesDecision{outcome: scannedBytesOutcomeWithin, evaluated: true, allowed: 7 * 24 * time.Hour, scannedBytes: 300, maxDuration: 7 * 24 * time.Hour, maxScannedBytes: 1000},
			wantCalls: 1,
		},
		{
			name:      "no blocks overlap",
			cfg:       scannedBytesConfig{maxScannedBytes: 1},
			requested: 10 * time.Hour,
			want:      scannedBytesDecision{outcome: scannedBytesOutcomeWithin, evaluated: true, maxScannedBytes: 1},
			wantCalls: 1,
		},
		{
			name:      "bound by scanned bytes",
			metas:     hourly,
			cfg:       scannedBytesConfig{maxDuration: 24 * time.Hour, maxScannedBytes: 450},
			requested: 6 * time.Hour,
			want: scannedBytesDecision{
				outcome: scannedBytesOutcomeRejected, evaluated: true, limited: true,
				allowed: 4*time.Hour - time.Nanosecond, derived: 4*time.Hour - time.Nanosecond,
				scannedBytes: 700, maxDuration: 24 * time.Hour, maxScannedBytes: 450,
			},
			wantErr:   "metrics query time range of 6h0m0s exceeds the maximum allowed range of 3h59m59s for tenant foo (limited by scanned bytes)",
			wantCalls: 1,
		},
		{
			name:      "requested rounds up and allowed rounds down in the error",
			metas:     hourly,
			cfg:       scannedBytesConfig{maxScannedBytes: 450},
			requested: 4*time.Hour + 500*time.Millisecond,
			want: scannedBytesDecision{
				outcome: scannedBytesOutcomeRejected, evaluated: true, limited: true,
				allowed: 4*time.Hour - time.Nanosecond, derived: 4*time.Hour - time.Nanosecond,
				scannedBytes: 500, maxScannedBytes: 450,
			},
			wantErr:   "metrics query time range of 4h0m1s exceeds the maximum allowed range of 3h59m59s for tenant foo (limited by scanned bytes)",
			wantCalls: 1,
		},
		{
			name:      "bound by the floor",
			metas:     hourly,
			cfg:       scannedBytesConfig{maxDuration: 24 * time.Hour, minDuration: 6 * time.Hour, maxScannedBytes: 150},
			requested: 6 * time.Hour,
			want: scannedBytesDecision{
				outcome: scannedBytesOutcomeFloored, evaluated: true, limited: true,
				allowed: 6 * time.Hour, derived: time.Hour - time.Nanosecond,
				scannedBytes: 700, minDuration: 6 * time.Hour, maxDuration: 24 * time.Hour, maxScannedBytes: 150,
			},
			wantCalls: 1,
		},
		{
			name:      "bound by the floor and over it",
			metas:     hourly,
			cfg:       scannedBytesConfig{maxDuration: 24 * time.Hour, minDuration: 6 * time.Hour, maxScannedBytes: 150},
			requested: 7 * time.Hour,
			want: scannedBytesDecision{
				outcome: scannedBytesOutcomeRejected, evaluated: true, limited: true,
				allowed: 6 * time.Hour, derived: time.Hour - time.Nanosecond,
				scannedBytes: 800, minDuration: 6 * time.Hour, maxDuration: 24 * time.Hour, maxScannedBytes: 150,
			},
			wantErr:   "metrics query time range of 7h0m0s exceeds the maximum allowed range of 6h0m0s for tenant foo (limited by scanned bytes)",
			wantCalls: 1,
		},
		{
			name:      "bound by the ceiling over a larger floor",
			metas:     hourly,
			cfg:       scannedBytesConfig{maxDuration: 5 * time.Hour, minDuration: 8 * time.Hour, maxScannedBytes: 150},
			requested: 5 * time.Hour,
			want: scannedBytesDecision{
				outcome: scannedBytesOutcomeFloored, evaluated: true, limited: true,
				allowed: 5 * time.Hour, derived: time.Hour - time.Nanosecond,
				scannedBytes: 600, minDuration: 8 * time.Hour, maxDuration: 5 * time.Hour, maxScannedBytes: 150,
			},
			wantCalls: 1,
		},
		{
			name:      "over max_duration is still evaluated for the log",
			metas:     hourly,
			cfg:       scannedBytesConfig{maxDuration: 5 * time.Hour, maxScannedBytes: 150},
			requested: 6 * time.Hour,
			want: scannedBytesDecision{
				outcome: scannedBytesOutcomeRejectedMaxDuration, evaluated: true, limited: true,
				allowed: time.Hour - time.Nanosecond, derived: time.Hour - time.Nanosecond,
				scannedBytes: 700, maxDuration: 5 * time.Hour, maxScannedBytes: 150,
			},
			wantCalls: 1,
		},
		{
			name:      "negative max_duration fails closed and is not evaluated",
			metas:     hourly,
			cfg:       scannedBytesConfig{maxDuration: -time.Hour, maxScannedBytes: 150},
			requested: time.Hour,
			want:      scannedBytesDecision{outcome: scannedBytesOutcomeRejectedMaxDuration, maxDuration: -time.Hour, maxScannedBytes: 150},
		},
		{
			name:      "zero start is not evaluated",
			metas:     hourly,
			cfg:       scannedBytesConfig{maxScannedBytes: 1},
			start:     time.Unix(0, 0),
			requested: time.Hour,
			want:      scannedBytesDecision{outcome: scannedBytesOutcomeWithin, maxScannedBytes: 1},
		},
		{
			name:      "end before start is not evaluated",
			metas:     hourly,
			cfg:       scannedBytesConfig{maxScannedBytes: 1},
			start:     scannedBytesTestEnd.Add(time.Hour),
			requested: time.Hour,
			want:      scannedBytesDecision{outcome: scannedBytesOutcomeWithin, maxScannedBytes: 1},
		},
		{
			name:      "a block starting at the cutoff counts",
			metas:     []*backend.BlockMeta{testBlock(1, 0, 300)},
			cfg:       scannedBytesConfig{maxScannedBytes: 250},
			cutoff:    hoursAgo(1),
			requested: 2 * time.Hour,
			want: scannedBytesDecision{
				outcome: scannedBytesOutcomeRejected, evaluated: true, limited: true,
				allowed: time.Hour, derived: time.Hour, scannedBytes: 300, maxScannedBytes: 250,
			},
			wantErr:   "metrics query time range of 2h0m0s exceeds the maximum allowed range of 1h0m0s for tenant foo (limited by scanned bytes)",
			wantCalls: 1,
		},
		{
			name:      "a prefix of blocks equal to the budget fits",
			metas:     []*backend.BlockMeta{testBlock(1, 0, 100), testBlock(2, 1, 150)},
			cfg:       scannedBytesConfig{maxScannedBytes: 100},
			requested: 2 * time.Hour,
			want: scannedBytesDecision{
				outcome: scannedBytesOutcomeRejected, evaluated: true, limited: true,
				allowed: time.Hour - time.Nanosecond, derived: time.Hour - time.Nanosecond, scannedBytes: 250, maxScannedBytes: 100,
			},
			wantErr:   "metrics query time range of 2h0m0s exceeds the maximum allowed range of 59m59s for tenant foo (limited by scanned bytes)",
			wantCalls: 1,
		},
		{
			name:      "one block larger than the budget",
			metas:     []*backend.BlockMeta{testBlock(1, 0, 1000)},
			cfg:       scannedBytesConfig{maxDuration: 24 * time.Hour, maxScannedBytes: 500},
			requested: time.Hour,
			want: scannedBytesDecision{
				outcome: scannedBytesOutcomeRejected, evaluated: true, limited: true,
				scannedBytes: 1000, maxDuration: 24 * time.Hour, maxScannedBytes: 500,
			},
			wantErr:   "metrics query time range of 1h0m0s exceeds the maximum allowed range of 0s for tenant foo (limited by scanned bytes)",
			wantCalls: 1,
		},
		{
			name:      "range after the cutoff is free",
			metas:     []*backend.BlockMeta{testBlock(3, 0, 300)},
			cfg:       scannedBytesConfig{maxScannedBytes: 250},
			cutoff:    hoursAgo(1),
			requested: 3 * time.Hour,
			want: scannedBytesDecision{
				outcome: scannedBytesOutcomeRejected, evaluated: true, limited: true,
				allowed: time.Hour, derived: time.Hour, scannedBytes: 300, maxScannedBytes: 250,
			},
			wantErr:   "metrics query time range of 3h0m0s exceeds the maximum allowed range of 1h0m0s for tenant foo (limited by scanned bytes)",
			wantCalls: 1,
		},
		{
			name:      "range wholly after the cutoff skips the blocklist",
			metas:     []*backend.BlockMeta{testBlock(3, 0, 300)},
			cfg:       scannedBytesConfig{maxScannedBytes: 250},
			cutoff:    hoursAgo(2),
			requested: time.Hour,
			want:      scannedBytesDecision{outcome: scannedBytesOutcomeWithin, evaluated: true, maxScannedBytes: 250},
		},
		{
			// A long compacted block starts before the fitting blocks but ends inside the range. Input order is shuffled.
			name:      "blocks enter by end time, newest first",
			metas:     []*backend.BlockMeta{testBlock(3, 2, 100), testBlock(1, 0, 100), testBlock(10, 1, 100)},
			cfg:       scannedBytesConfig{maxScannedBytes: 250},
			requested: 10 * time.Hour,
			want: scannedBytesDecision{
				outcome: scannedBytesOutcomeRejected, evaluated: true, limited: true,
				allowed: 2*time.Hour - time.Nanosecond, derived: 2*time.Hour - time.Nanosecond, scannedBytes: 300, maxScannedBytes: 250,
			},
			wantErr:   "metrics query time range of 10h0m0s exceeds the maximum allowed range of 1h59m59s for tenant foo (limited by scanned bytes)",
			wantCalls: 1,
		},
		{
			name:      "blocks with the same end time enter together",
			metas:     []*backend.BlockMeta{testBlock(1, 0, 100), testBlock(2, 1, 100), testBlock(2, 1, 100)},
			cfg:       scannedBytesConfig{maxScannedBytes: 250},
			requested: 2 * time.Hour,
			want: scannedBytesDecision{
				outcome: scannedBytesOutcomeRejected, evaluated: true, limited: true,
				allowed: time.Hour - time.Nanosecond, derived: time.Hour - time.Nanosecond, scannedBytes: 300, maxScannedBytes: 250,
			},
			wantErr:   "metrics query time range of 2h0m0s exceeds the maximum allowed range of 59m59s for tenant foo (limited by scanned bytes)",
			wantCalls: 1,
		},
		{
			name:      "metrics counts only RF1 blocks",
			metas:     append([]*backend.BlockMeta{nonRF1}, hourly...),
			cfg:       scannedBytesConfig{maxScannedBytes: 350},
			requested: 2 * time.Hour,
			want:      scannedBytesDecision{outcome: scannedBytesOutcomeWithin, evaluated: true, scannedBytes: 300, maxScannedBytes: 350},
			wantCalls: 1,
		},
		{
			name:      "search counts every block",
			op:        searchOp,
			metas:     append([]*backend.BlockMeta{nonRF1}, hourly...),
			cfg:       scannedBytesConfig{maxScannedBytes: 350},
			requested: 2 * time.Hour,
			want: scannedBytesDecision{
				outcome: scannedBytesOutcomeRejected, evaluated: true, limited: true,
				scannedBytes: 10_300, maxScannedBytes: 350,
			},
			wantErr:   "search time range of 2h0m0s exceeds the maximum allowed range of 0s for tenant foo (limited by scanned bytes)",
			wantCalls: 1,
		},
		{
			name:      "metrics overrides beat the config",
			metas:     hourly,
			cfg:       scannedBytesConfig{maxDuration: 24 * time.Hour, minDuration: time.Hour, maxScannedBytes: 10_000},
			overrides: scannedBytesConfig{maxDuration: 6 * time.Hour, minDuration: 5 * time.Hour, maxScannedBytes: 150},
			requested: 6 * time.Hour,
			want: scannedBytesDecision{
				outcome: scannedBytesOutcomeRejected, evaluated: true, limited: true,
				allowed: 5 * time.Hour, derived: time.Hour - time.Nanosecond,
				scannedBytes: 700, minDuration: 5 * time.Hour, maxDuration: 6 * time.Hour, maxScannedBytes: 150,
			},
			wantErr:   "metrics query time range of 6h0m0s exceeds the maximum allowed range of 5h0m0s for tenant foo (limited by scanned bytes)",
			wantCalls: 1,
		},
		{
			name:      "search overrides beat the config",
			op:        searchOp,
			metas:     hourly,
			cfg:       scannedBytesConfig{maxDuration: 24 * time.Hour, minDuration: time.Hour, maxScannedBytes: 10_000},
			overrides: scannedBytesConfig{maxDuration: 6 * time.Hour, minDuration: 5 * time.Hour, maxScannedBytes: 150},
			requested: 6 * time.Hour,
			want: scannedBytesDecision{
				outcome: scannedBytesOutcomeRejected, evaluated: true, limited: true,
				allowed: 5 * time.Hour, derived: time.Hour - time.Nanosecond,
				scannedBytes: 700, minDuration: 5 * time.Hour, maxDuration: 6 * time.Hour, maxScannedBytes: 150,
			},
			wantErr:   "search time range of 6h0m0s exceeds the maximum allowed range of 5h0m0s for tenant foo (limited by scanned bytes)",
			wantCalls: 1,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			op := tc.op
			if op == "" {
				op = metricsOp
			}
			start := tc.start
			if start.IsZero() {
				start = scannedBytesTestEnd.Add(-tc.requested)
			}
			cutoff := tc.cutoff
			if cutoff.IsZero() {
				cutoff = scannedBytesTestEnd
			}
			calls := 0
			o := &mockOverridesScannedBytes{}
			if op == metricsOp {
				o.metrics = tc.overrides
			} else {
				o.search = tc.overrides
			}

			got := evaluateScannedBytesLimit(o, tc.cfg, scannedBytesQuery{
				op:        op,
				tenantID:  "foo",
				requested: tc.requested,
				start:     start,
				end:       scannedBytesTestEnd,
				cutoff:    cutoff,
				metas:     func() []*backend.BlockMeta { calls++; return tc.metas },
			})

			want := tc.want
			want.op, want.tenantID, want.requested = op, "foo", tc.requested
			require.Equal(t, want, got)
			require.Equal(t, tc.wantCalls, calls)

			kind := "metrics query"
			if op == searchOp {
				kind = "search"
			}
			if tc.wantErr == "" {
				require.NoError(t, got.scannedBytesErr(kind))
			} else {
				require.EqualError(t, got.scannedBytesErr(kind), tc.wantErr)
			}
		})
	}
}

func TestScannedBytesDecisionRecord(t *testing.T) {
	tcs := []struct {
		name      string
		d         scannedBytesDecision
		wantLog   string
		wantAttrs map[attribute.Key]attribute.Value
		wantCount float64
		wantHist  uint64
		// wantHistSum is the observed seconds, the raw derived range rather than allowed.
		wantHistSum float64
	}{
		{
			name: "not evaluated records nothing",
			d:    scannedBytesDecision{outcome: scannedBytesOutcomeRejectedMaxDuration},
		},
		{
			// Requested rounds up and the limits round down, like the rejection error.
			name: "over budget",
			d: scannedBytesDecision{
				outcome: scannedBytesOutcomeRejected, requested: 2*time.Hour + 500*time.Millisecond, evaluated: true, limited: true,
				allowed: 2 * time.Hour, derived: time.Hour + 500*time.Millisecond, scannedBytes: 300, minDuration: 2 * time.Hour, maxDuration: 24 * time.Hour,
			},
			wantLog: `level=info msg="scanned bytes limit" tenant=record op=search requested_seconds=7201 scanned_bytes=300 derived_seconds=3600 min_seconds=7200 max_seconds=86400 outcome=rejected passes_scanned_bytes=false`,
			wantAttrs: map[attribute.Key]attribute.Value{
				"scannedBytesLimitOutcome":             attribute.StringValue(scannedBytesOutcomeRejected),
				"scannedBytesLimitRequestedSeconds":    attribute.Int64Value(7201),
				"scannedBytesLimitScannedBytes":        attribute.Int64Value(300),
				"scannedBytesLimitMinSeconds":          attribute.Int64Value(7200),
				"scannedBytesLimitMaxSeconds":          attribute.Int64Value(86400),
				"scannedBytesLimitPassesScannedBytes":  attribute.BoolValue(false),
				"scannedBytesLimitDerivedRangeSeconds": attribute.Int64Value(3600),
				"scannedBytesLimitMaxDurationSeconds":  attribute.Int64Value(7200),
			},
			wantCount:   1,
			wantHist:    1,
			wantHistSum: 3600.5,
		},
		{
			name: "fits without a ceiling sets no allowed range attribute",
			d: scannedBytesDecision{
				outcome: scannedBytesOutcomeWithin, requested: time.Hour, evaluated: true, scannedBytes: 100,
			},
			wantLog: `level=info msg="scanned bytes limit" tenant=record op=search requested_seconds=3600 scanned_bytes=100 fits=true min_seconds=0 max_seconds=0 outcome=within passes_scanned_bytes=true`,
			wantAttrs: map[attribute.Key]attribute.Value{
				"scannedBytesLimitOutcome":            attribute.StringValue(scannedBytesOutcomeWithin),
				"scannedBytesLimitRequestedSeconds":   attribute.Int64Value(3600),
				"scannedBytesLimitScannedBytes":       attribute.Int64Value(100),
				"scannedBytesLimitMinSeconds":         attribute.Int64Value(0),
				"scannedBytesLimitMaxSeconds":         attribute.Int64Value(0),
				"scannedBytesLimitPassesScannedBytes": attribute.BoolValue(true),
				"scannedBytesLimitFits":               attribute.BoolValue(true),
			},
			wantCount: 1,
		},
		{
			name: "fits",
			d: scannedBytesDecision{
				outcome: scannedBytesOutcomeWithin, requested: 2 * time.Hour, evaluated: true,
				allowed: 24 * time.Hour, scannedBytes: 100, minDuration: time.Hour, maxDuration: 24 * time.Hour,
			},
			wantLog: `level=info msg="scanned bytes limit" tenant=record op=search requested_seconds=7200 scanned_bytes=100 fits=true min_seconds=3600 max_seconds=86400 outcome=within passes_scanned_bytes=true`,
			wantAttrs: map[attribute.Key]attribute.Value{
				"scannedBytesLimitOutcome":            attribute.StringValue(scannedBytesOutcomeWithin),
				"scannedBytesLimitRequestedSeconds":   attribute.Int64Value(7200),
				"scannedBytesLimitScannedBytes":       attribute.Int64Value(100),
				"scannedBytesLimitMinSeconds":         attribute.Int64Value(3600),
				"scannedBytesLimitMaxSeconds":         attribute.Int64Value(86400),
				"scannedBytesLimitPassesScannedBytes": attribute.BoolValue(true),
				"scannedBytesLimitFits":               attribute.BoolValue(true),
				"scannedBytesLimitMaxDurationSeconds": attribute.Int64Value(86400),
			},
			wantCount: 1,
		},
		{
			// The floor alone, at exactly the requested range, would allow it, so it passes on scanned bytes.
			name: "over max_duration is counted but stays out of the histogram",
			d: scannedBytesDecision{
				outcome: scannedBytesOutcomeRejectedMaxDuration, requested: 6 * time.Hour, evaluated: true, limited: true,
				allowed: 5 * time.Hour, derived: time.Hour, scannedBytes: 700, minDuration: 6 * time.Hour, maxDuration: 5 * time.Hour,
			},
			wantLog: `level=info msg="scanned bytes limit" tenant=record op=search requested_seconds=21600 scanned_bytes=700 derived_seconds=3600 min_seconds=21600 max_seconds=18000 outcome=rejected_max_duration passes_scanned_bytes=true`,
			wantAttrs: map[attribute.Key]attribute.Value{
				"scannedBytesLimitOutcome":             attribute.StringValue(scannedBytesOutcomeRejectedMaxDuration),
				"scannedBytesLimitRequestedSeconds":    attribute.Int64Value(21600),
				"scannedBytesLimitScannedBytes":        attribute.Int64Value(700),
				"scannedBytesLimitMinSeconds":          attribute.Int64Value(21600),
				"scannedBytesLimitMaxSeconds":          attribute.Int64Value(18000),
				"scannedBytesLimitPassesScannedBytes":  attribute.BoolValue(true),
				"scannedBytesLimitDerivedRangeSeconds": attribute.Int64Value(3600),
				"scannedBytesLimitMaxDurationSeconds":  attribute.Int64Value(18000),
			},
			wantCount: 1,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			rec := tracetest.NewSpanRecorder()
			tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
			_, span := tp.Tracer("test").Start(context.Background(), "test")
			var buf bytes.Buffer
			d := tc.d
			d.op, d.tenantID = searchOp, "record"

			countBefore := outcomeCount("record", searchOp, d.outcome)
			histBefore, sumBefore := histogramSample(t, "record", searchOp)
			d.record(span, log.NewLogfmtLogger(&buf))
			span.End()

			require.Equal(t, tc.wantCount, outcomeCount("record", searchOp, d.outcome)-countBefore)
			histAfter, sumAfter := histogramSample(t, "record", searchOp)
			require.Equal(t, tc.wantHist, histAfter-histBefore)
			require.InDelta(t, tc.wantHistSum, sumAfter-sumBefore, 1e-9)
			if tc.wantLog == "" {
				require.Empty(t, buf.String())
			} else {
				require.Equal(t, tc.wantLog+"\n", buf.String())
			}
			got := map[attribute.Key]attribute.Value{}
			for _, kv := range rec.Ended()[0].Attributes() {
				got[kv.Key] = kv.Value
			}
			if tc.wantAttrs == nil {
				require.Empty(t, got)
			} else {
				require.Equal(t, tc.wantAttrs, got)
			}
		})
	}
}

func TestScannedBytesLimiterCheckMultiTenant(t *testing.T) {
	rdr := &perTenantMockReader{metas: map[string][]*backend.BlockMeta{
		"mt-a-over": hourlyMetricsBlocks(scannedBytesTestEnd),
		"mt-b-fits": nil,
	}}
	var buf bytes.Buffer
	l := newScannedBytesLimiter(rdr, &mockOverridesScannedBytes{}, QueryRangeSharderConfig{MaxScannedBytes: 350}, log.NewLogfmtLogger(&buf))

	overBefore := outcomeCount("mt-a-over", metricsOp, scannedBytesOutcomeRejected)
	fitsBefore := outcomeCount("mt-b-fits", metricsOp, scannedBytesOutcomeWithin)
	// tenant.TenantIDs sorts, so the over-budget tenant comes first and the fitting one is evaluated after the rejection.
	err := l.check(user.InjectOrgID(context.Background(), "mt-b-fits|mt-a-over"), &tempopb.QueryRangeRequest{
		Start: uint64(hoursAgo(4).UnixNano()),
		End:   uint64(scannedBytesTestEnd.UnixNano()),
	})
	require.EqualError(t, err, "metrics query time range of 4h0m0s exceeds the maximum allowed range of 2h59m59s for tenant mt-a-over (limited by scanned bytes)")

	require.Equal(t, 1.0, outcomeCount("mt-a-over", metricsOp, scannedBytesOutcomeRejected)-overBefore)
	require.Equal(t, 1.0, outcomeCount("mt-b-fits", metricsOp, scannedBytesOutcomeWithin)-fitsBefore)
	require.Equal(t, `level=info msg="scanned bytes limit" tenant=mt-a-over op=metrics requested_seconds=14400 scanned_bytes=500 derived_seconds=10799 min_seconds=0 max_seconds=0 outcome=rejected passes_scanned_bytes=false
level=info msg="scanned bytes limit" tenant=mt-b-fits op=metrics requested_seconds=14400 scanned_bytes=0 fits=true min_seconds=0 max_seconds=0 outcome=within passes_scanned_bytes=true
`, buf.String())
}

func TestScannedBytesLimiterRecordRejectedMaxDuration(t *testing.T) {
	rdr := &perTenantMockReader{metas: map[string][]*backend.BlockMeta{
		"rmd-a-over": hourlyMetricsBlocks(scannedBytesTestEnd),
		"rmd-b-fits": nil,
	}}
	var buf bytes.Buffer
	l := newScannedBytesLimiter(rdr, &mockOverridesScannedBytes{}, QueryRangeSharderConfig{MaxScannedBytes: 350}, log.NewLogfmtLogger(&buf))

	histBefore, _ := histogramSample(t, "rmd-a-over", metricsOp)
	// The raw org ID's ceiling rejected it, so every tenant logs rejected_max_duration even though its own limits allow more.
	l.recordRejectedMaxDuration(user.InjectOrgID(context.Background(), "rmd-b-fits|rmd-a-over"), &tempopb.QueryRangeRequest{
		Start: uint64(hoursAgo(4).UnixNano()),
		End:   uint64(scannedBytesTestEnd.UnixNano()),
	})

	histAfter, _ := histogramSample(t, "rmd-a-over", metricsOp)
	require.Equal(t, histBefore, histAfter)
	require.Equal(t, `level=info msg="scanned bytes limit" tenant=rmd-a-over op=metrics requested_seconds=14400 scanned_bytes=500 derived_seconds=10799 min_seconds=0 max_seconds=0 outcome=rejected_max_duration passes_scanned_bytes=false
level=info msg="scanned bytes limit" tenant=rmd-b-fits op=metrics requested_seconds=14400 scanned_bytes=0 fits=true min_seconds=0 max_seconds=0 outcome=rejected_max_duration passes_scanned_bytes=true
`, buf.String())
}

func TestScannedBytesLimiterCheckTenantMaxDuration(t *testing.T) {
	// validateMetricsQueryMaxDuration reads the raw org ID, so a tenant's own ceiling is enforced by the limiter.
	for _, maxDuration := range []time.Duration{time.Hour, -time.Hour} {
		l := newScannedBytesLimiter(&mockReader{}, &mockOverridesScannedBytes{metrics: scannedBytesConfig{maxDuration: maxDuration}},
			QueryRangeSharderConfig{MaxScannedBytes: 1, MaxDuration: 24 * time.Hour}, log.NewNopLogger())

		err := l.check(user.InjectOrgID(context.Background(), "foo"), &tempopb.QueryRangeRequest{
			Start: uint64(hoursAgo(2).UnixNano()),
			End:   uint64(scannedBytesTestEnd.UnixNano()),
		})
		require.EqualError(t, err, fmt.Sprintf("metrics query time range exceeds the maximum allowed duration of %s", maxDuration))
	}
}

func TestScannedBytesLimiterCheckLimitOff(t *testing.T) {
	// With the limit off the raw-org-ID max_duration check stays the only ceiling, as before this feature.
	l := newScannedBytesLimiter(&mockReader{}, &mockOverridesScannedBytes{metrics: scannedBytesConfig{maxDuration: time.Hour}},
		QueryRangeSharderConfig{MaxDuration: 24 * time.Hour}, log.NewNopLogger())

	require.NoError(t, l.check(user.InjectOrgID(context.Background(), "foo"), &tempopb.QueryRangeRequest{
		Start: uint64(hoursAgo(2).UnixNano()),
		End:   uint64(scannedBytesTestEnd.UnixNano()),
	}))
}

func TestScannedBytesLimiterCheckHugeRange(t *testing.T) {
	// A range past math.MaxInt64 nanoseconds must not wrap negative and slip under the allowed range.
	l := newScannedBytesLimiter(&mockReader{metas: hourlyMetricsBlocks(scannedBytesTestEnd)}, &mockOverridesScannedBytes{},
		QueryRangeSharderConfig{MaxScannedBytes: 1}, log.NewNopLogger())

	err := l.check(user.InjectOrgID(context.Background(), "foo"), &tempopb.QueryRangeRequest{Start: 1, End: math.MaxUint64})
	require.ErrorContains(t, err, "(limited by scanned bytes)")
}

func TestScannedBytesLimiterCheckCutoff(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// One large block straddles the cutoff at now-30m.
		now := time.Now()
		rdr := &mockReader{metas: []*backend.BlockMeta{{
			StartTime:         now.Add(-40 * time.Minute),
			EndTime:           now.Add(-time.Minute),
			Size_:             1000,
			ReplicationFactor: backend.MetricsGeneratorReplicationFactor,
		}}}
		l := newScannedBytesLimiter(rdr, &mockOverridesScannedBytes{}, QueryRangeSharderConfig{MaxScannedBytes: 500, QueryBackendAfter: 30 * time.Minute}, log.NewNopLogger())

		// The 28m between the cutoff and the end reach no backend block, so they are allowed.
		err := l.check(user.InjectOrgID(context.Background(), "foo"), &tempopb.QueryRangeRequest{
			Start: uint64(now.Add(-time.Hour).UnixNano()),
			End:   uint64(now.Add(-2 * time.Minute).UnixNano()),
		})
		require.EqualError(t, err, "metrics query time range of 58m0s exceeds the maximum allowed range of 28m0s for tenant foo (limited by scanned bytes)")
	})
}

func BenchmarkEvaluateScannedBytesLimit(b *testing.B) {
	const (
		retention = 14 * 24 * time.Hour
		blockSize = 100 * 1024 * 1024
	)
	for _, n := range []int{85_145, 2 * 85_145, 10 * 85_145} {
		r := rand.New(rand.NewPCG(1, 2))
		metas := make([]*backend.BlockMeta, n)
		for i := range metas {
			end := scannedBytesTestEnd.Add(-time.Duration(r.Int64N(int64(retention))))
			metas[i] = &backend.BlockMeta{StartTime: end.Add(-time.Hour), EndTime: end, Size_: blockSize, ReplicationFactor: backend.MetricsGeneratorReplicationFactor}
		}
		// The real blocklist, so the benchmark pays the lock and copy that BlockMetas does per query.
		list := blocklist.New()
		list.ApplyPollResults(blocklist.PerTenant{"foo": metas}, nil, nil)
		for _, bc := range []struct {
			name            string
			maxScannedBytes uint64
		}{
			{name: "fits", maxScannedBytes: math.MaxUint64},
			{name: "over_budget", maxScannedBytes: uint64(n) * blockSize / 4},
		} {
			cfg := scannedBytesConfig{maxDuration: 14 * 24 * time.Hour, maxScannedBytes: bc.maxScannedBytes}
			q := scannedBytesQuery{
				op:        metricsOp,
				tenantID:  "foo",
				requested: 7 * 24 * time.Hour,
				start:     scannedBytesTestEnd.Add(-7 * 24 * time.Hour),
				end:       scannedBytesTestEnd,
				cutoff:    scannedBytesTestEnd,
				metas:     func() []*backend.BlockMeta { return list.Metas("foo") },
			}
			b.Run(fmt.Sprintf("blocks=%d/%s", n, bc.name), func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					evaluateScannedBytesLimit(&mockOverridesScannedBytes{}, cfg, q)
				}
			})
		}
	}
}
