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
	metrics, search rangeLimits
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

// testScannedBytesLimit returns the real op's limit with limits set in its config.
func testScannedBytesLimit(op string, l rangeLimits) scannedBytesLimit {
	if op == searchOp {
		return searchScannedBytesLimit{
			cfg:       SearchSharderConfig{MinDuration: l.minDuration, MaxDuration: l.maxDuration, MaxScannedBytes: l.maxScannedBytes},
			overrides: &mockOverridesScannedBytes{},
		}
	}
	return newMetricsScannedBytesLimit(nil, &mockOverridesScannedBytes{},
		QueryRangeSharderConfig{MinDuration: l.minDuration, MaxDuration: l.maxDuration, MaxScannedBytes: l.maxScannedBytes}, true, nil)
}

func TestScannedBytesLimitOverrides(t *testing.T) {
	cfg := rangeLimits{minDuration: time.Hour, maxDuration: 24 * time.Hour, maxScannedBytes: 1000}
	o := rangeLimits{minDuration: 2 * time.Hour, maxDuration: 12 * time.Hour, maxScannedBytes: 500}
	for _, op := range []string{searchOp, metricsOp} {
		t.Run(op, func(t *testing.T) {
			require.Equal(t, cfg, testScannedBytesLimit(op, cfg).limits("foo"))

			var l scannedBytesLimit = searchScannedBytesLimit{
				cfg:       SearchSharderConfig{MinDuration: cfg.minDuration, MaxDuration: cfg.maxDuration, MaxScannedBytes: cfg.maxScannedBytes},
				overrides: &mockOverridesScannedBytes{search: o},
			}
			if op == metricsOp {
				l = newMetricsScannedBytesLimit(nil, &mockOverridesScannedBytes{metrics: o},
					QueryRangeSharderConfig{MinDuration: cfg.minDuration, MaxDuration: cfg.maxDuration, MaxScannedBytes: cfg.maxScannedBytes}, true, nil)
			}
			require.Equal(t, o, l.limits("foo"))
		})
	}
}

func TestCheckScannedBytes(t *testing.T) {
	hourly := make([]*backend.BlockMeta, 0, 10)
	for i := range 10 {
		hourly = append(hourly, testBlock(i+1, i, 100))
	}
	nonRF1 := &backend.BlockMeta{StartTime: hoursAgo(1), EndTime: scannedBytesTestEnd, Size_: 10_000}

	tcs := []struct {
		name   string
		op     string
		metas  []*backend.BlockMeta
		limits rangeLimits
		// start defaults to scannedBytesTestEnd minus requested. cutoff defaults to scannedBytesTestEnd.
		start     time.Time
		cutoff    time.Time
		requested time.Duration
		want      rangeCheck
		wantErr   string
		wantCalls int
	}{
		{
			name:      "limit off allows any range",
			metas:     hourly,
			limits:    rangeLimits{maxDuration: 24 * time.Hour},
			requested: 8 * time.Hour,
			want:      rangeCheck{fits: true, outcome: scannedBytesOutcomeWithin},
		},
		{
			name:      "limit off still rejects over max_duration",
			op:        searchOp,
			metas:     hourly,
			limits:    rangeLimits{maxDuration: 5 * time.Hour},
			requested: 6 * time.Hour,
			want:      rangeCheck{fits: true, outcome: scannedBytesOutcomeRejectedMaxDuration},
		},
		{
			name:      "full fit",
			metas:     hourly,
			limits:    rangeLimits{maxDuration: 24 * time.Hour, maxScannedBytes: 1000},
			requested: 10 * time.Hour,
			want:      rangeCheck{fits: true, outcome: scannedBytesOutcomeWithin, evaluated: true, allowed: 24 * time.Hour, scannedBytes: 1000},
			wantCalls: 1,
		},
		{
			// The design doc sketch returned the 3h of data here and rejected a query that fits.
			name:      "data starts mid-range",
			metas:     hourly[:3],
			limits:    rangeLimits{maxDuration: 7 * 24 * time.Hour, maxScannedBytes: 1000},
			requested: 7 * 24 * time.Hour,
			want:      rangeCheck{fits: true, outcome: scannedBytesOutcomeWithin, evaluated: true, allowed: 7 * 24 * time.Hour, scannedBytes: 300},
			wantCalls: 1,
		},
		{
			name:      "no blocks overlap",
			limits:    rangeLimits{maxScannedBytes: 1},
			requested: 10 * time.Hour,
			want:      rangeCheck{fits: true, outcome: scannedBytesOutcomeWithin, evaluated: true},
			wantCalls: 1,
		},
		{
			name:      "bound by scanned bytes",
			metas:     hourly,
			limits:    rangeLimits{maxDuration: 24 * time.Hour, maxScannedBytes: 450},
			requested: 6 * time.Hour,
			want: rangeCheck{
				outcome: scannedBytesOutcomeRejected, evaluated: true, allowed: 4*time.Hour - time.Nanosecond, derived: 4*time.Hour - time.Nanosecond,
				scannedBytes: 700,
			},
			wantErr:   "metrics query time range of 6h0m0s exceeds the maximum allowed range of 3h59m59s for tenant foo (limited by scanned bytes)",
			wantCalls: 1,
		},
		{
			name:      "requested rounds up and allowed rounds down in the error",
			metas:     hourly,
			limits:    rangeLimits{maxScannedBytes: 450},
			requested: 4*time.Hour + 500*time.Millisecond,
			want: rangeCheck{
				outcome: scannedBytesOutcomeRejected, evaluated: true, allowed: 4*time.Hour - time.Nanosecond, derived: 4*time.Hour - time.Nanosecond,
				scannedBytes: 500,
			},
			wantErr:   "metrics query time range of 4h0m1s exceeds the maximum allowed range of 3h59m59s for tenant foo (limited by scanned bytes)",
			wantCalls: 1,
		},
		{
			name:      "raised to min_duration",
			metas:     hourly,
			limits:    rangeLimits{maxDuration: 24 * time.Hour, minDuration: 6 * time.Hour, maxScannedBytes: 150},
			requested: 6 * time.Hour,
			want: rangeCheck{
				outcome: scannedBytesOutcomeRaisedToMin, evaluated: true, allowed: 6 * time.Hour, derived: time.Hour - time.Nanosecond,
				scannedBytes: 700,
			},
			wantCalls: 1,
		},
		{
			name:      "over min_duration",
			metas:     hourly,
			limits:    rangeLimits{maxDuration: 24 * time.Hour, minDuration: 6 * time.Hour, maxScannedBytes: 150},
			requested: 7 * time.Hour,
			want: rangeCheck{
				outcome: scannedBytesOutcomeRejected, evaluated: true, allowed: 6 * time.Hour, derived: time.Hour - time.Nanosecond,
				scannedBytes: 800,
			},
			wantErr:   "metrics query time range of 7h0m0s exceeds the maximum allowed range of 6h0m0s for tenant foo (limited by scanned bytes)",
			wantCalls: 1,
		},
		{
			name:      "max_duration below min_duration wins",
			metas:     hourly,
			limits:    rangeLimits{maxDuration: 5 * time.Hour, minDuration: 8 * time.Hour, maxScannedBytes: 150},
			requested: 5 * time.Hour,
			want: rangeCheck{
				outcome: scannedBytesOutcomeRaisedToMin, evaluated: true, allowed: 5 * time.Hour, derived: time.Hour - time.Nanosecond,
				scannedBytes: 600,
			},
			wantCalls: 1,
		},
		{
			name:      "over max_duration is still evaluated for the log",
			metas:     hourly,
			limits:    rangeLimits{maxDuration: 5 * time.Hour, maxScannedBytes: 150},
			requested: 6 * time.Hour,
			want: rangeCheck{
				outcome: scannedBytesOutcomeRejectedMaxDuration, evaluated: true, allowed: time.Hour - time.Nanosecond, derived: time.Hour - time.Nanosecond,
				scannedBytes: 700,
			},
			wantCalls: 1,
		},
		{
			name:      "negative max_duration fails closed and is not evaluated",
			metas:     hourly,
			limits:    rangeLimits{maxDuration: -time.Hour, maxScannedBytes: 150},
			requested: time.Hour,
			want:      rangeCheck{fits: true, outcome: scannedBytesOutcomeRejectedMaxDuration},
		},
		{
			name:      "zero start is not evaluated",
			metas:     hourly,
			limits:    rangeLimits{maxScannedBytes: 1},
			start:     time.Unix(0, 0),
			requested: time.Hour,
			want:      rangeCheck{fits: true, outcome: scannedBytesOutcomeWithin},
		},
		{
			name:      "end before start is not evaluated",
			metas:     hourly,
			limits:    rangeLimits{maxScannedBytes: 1},
			start:     scannedBytesTestEnd.Add(time.Hour),
			requested: time.Hour,
			want:      rangeCheck{fits: true, outcome: scannedBytesOutcomeWithin},
		},
		{
			name:      "a block starting at the cutoff counts",
			metas:     []*backend.BlockMeta{testBlock(1, 0, 300)},
			limits:    rangeLimits{maxScannedBytes: 250},
			cutoff:    hoursAgo(1),
			requested: 2 * time.Hour,
			want: rangeCheck{
				outcome: scannedBytesOutcomeRejected, evaluated: true, allowed: time.Hour, derived: time.Hour, scannedBytes: 300,
			},
			wantErr:   "metrics query time range of 2h0m0s exceeds the maximum allowed range of 1h0m0s for tenant foo (limited by scanned bytes)",
			wantCalls: 1,
		},
		{
			name:      "a prefix of blocks equal to the budget fits",
			metas:     []*backend.BlockMeta{testBlock(1, 0, 100), testBlock(2, 1, 150)},
			limits:    rangeLimits{maxScannedBytes: 100},
			requested: 2 * time.Hour,
			want: rangeCheck{
				outcome: scannedBytesOutcomeRejected, evaluated: true, allowed: time.Hour - time.Nanosecond, derived: time.Hour - time.Nanosecond, scannedBytes: 250,
			},
			wantErr:   "metrics query time range of 2h0m0s exceeds the maximum allowed range of 59m59s for tenant foo (limited by scanned bytes)",
			wantCalls: 1,
		},
		{
			name:      "one byte over the budget",
			metas:     []*backend.BlockMeta{testBlock(1, 0, 100), testBlock(2, 1, 151)},
			limits:    rangeLimits{maxScannedBytes: 250},
			requested: 2 * time.Hour,
			want: rangeCheck{
				outcome: scannedBytesOutcomeRejected, evaluated: true,
				allowed: time.Hour - time.Nanosecond, derived: time.Hour - time.Nanosecond, scannedBytes: 251,
			},
			wantErr:   "metrics query time range of 2h0m0s exceeds the maximum allowed range of 59m59s for tenant foo (limited by scanned bytes)",
			wantCalls: 1,
		},
		{
			name:      "one block larger than the budget",
			metas:     []*backend.BlockMeta{testBlock(1, 0, 1000)},
			limits:    rangeLimits{maxDuration: 24 * time.Hour, maxScannedBytes: 500},
			requested: time.Hour,
			want: rangeCheck{
				outcome: scannedBytesOutcomeRejected, evaluated: true, scannedBytes: 1000,
			},
			wantErr:   "metrics query time range of 1h0m0s exceeds the maximum allowed range of 0s for tenant foo (limited by scanned bytes)",
			wantCalls: 1,
		},
		{
			name:      "range after the cutoff is free",
			metas:     []*backend.BlockMeta{testBlock(3, 0, 300)},
			limits:    rangeLimits{maxScannedBytes: 250},
			cutoff:    hoursAgo(1),
			requested: 3 * time.Hour,
			want: rangeCheck{
				outcome: scannedBytesOutcomeRejected, evaluated: true, allowed: time.Hour, derived: time.Hour, scannedBytes: 300,
			},
			wantErr:   "metrics query time range of 3h0m0s exceeds the maximum allowed range of 1h0m0s for tenant foo (limited by scanned bytes)",
			wantCalls: 1,
		},
		{
			name:      "range wholly after the cutoff skips the blocklist",
			metas:     []*backend.BlockMeta{testBlock(3, 0, 300)},
			limits:    rangeLimits{maxScannedBytes: 250},
			cutoff:    hoursAgo(2),
			requested: time.Hour,
			want:      rangeCheck{fits: true, outcome: scannedBytesOutcomeWithin, evaluated: true},
		},
		{
			// A long compacted block starts before the fitting blocks but ends inside the range. Input order is shuffled.
			name:      "blocks enter by end time, newest first",
			metas:     []*backend.BlockMeta{testBlock(3, 2, 100), testBlock(1, 0, 100), testBlock(10, 1, 100)},
			limits:    rangeLimits{maxScannedBytes: 250},
			requested: 10 * time.Hour,
			want: rangeCheck{
				outcome: scannedBytesOutcomeRejected, evaluated: true, allowed: 2*time.Hour - time.Nanosecond, derived: 2*time.Hour - time.Nanosecond, scannedBytes: 300,
			},
			wantErr:   "metrics query time range of 10h0m0s exceeds the maximum allowed range of 1h59m59s for tenant foo (limited by scanned bytes)",
			wantCalls: 1,
		},
		{
			name:      "blocks with the same end time enter together",
			metas:     []*backend.BlockMeta{testBlock(1, 0, 100), testBlock(2, 1, 100), testBlock(2, 1, 100)},
			limits:    rangeLimits{maxScannedBytes: 250},
			requested: 2 * time.Hour,
			want: rangeCheck{
				outcome: scannedBytesOutcomeRejected, evaluated: true, allowed: time.Hour - time.Nanosecond, derived: time.Hour - time.Nanosecond, scannedBytes: 300,
			},
			wantErr:   "metrics query time range of 2h0m0s exceeds the maximum allowed range of 59m59s for tenant foo (limited by scanned bytes)",
			wantCalls: 1,
		},
		{
			name:      "metrics counts only RF1 blocks",
			metas:     append([]*backend.BlockMeta{nonRF1}, hourly...),
			limits:    rangeLimits{maxScannedBytes: 350},
			requested: 2 * time.Hour,
			want:      rangeCheck{fits: true, outcome: scannedBytesOutcomeWithin, evaluated: true, scannedBytes: 300},
			wantCalls: 1,
		},
		{
			name:      "search counts every block",
			op:        searchOp,
			metas:     append([]*backend.BlockMeta{nonRF1}, hourly...),
			limits:    rangeLimits{maxScannedBytes: 350},
			requested: 2 * time.Hour,
			want: rangeCheck{
				outcome: scannedBytesOutcomeRejected, evaluated: true, scannedBytes: 10_300,
			},
			wantErr:   "search time range of 2h0m0s exceeds the maximum allowed range of 0s for tenant foo (limited by scanned bytes)",
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
			got := checkScannedBytes(testScannedBytesLimit(op, tc.limits), "foo", start, scannedBytesTestEnd, cutoff,
				func() []*backend.BlockMeta { calls++; return tc.metas })

			want := tc.want
			want.rangeLimits, want.op, want.tenantID, want.requested = tc.limits, op, "foo", scannedBytesTestEnd.Sub(start)
			require.Equal(t, want, got)
			require.Equal(t, tc.wantCalls, calls)

			kind := "metrics query"
			if op == searchOp {
				kind = "search"
			}
			if tc.wantErr == "" {
				require.NoError(t, got.err(kind))
			} else {
				require.EqualError(t, got.err(kind), tc.wantErr)
			}
		})
	}
}

func TestRangeCheckRecord(t *testing.T) {
	tcs := []struct {
		name      string
		c         rangeCheck
		wantLog   string
		wantAttrs map[attribute.Key]attribute.Value
		wantCount float64
		wantHist  uint64
		// wantHistSum is the observed seconds, the raw derived range rather than allowed.
		wantHistSum float64
	}{
		{
			name: "not evaluated records nothing",
			c:    rangeCheck{fits: true, outcome: scannedBytesOutcomeRejectedMaxDuration},
		},
		{
			// Requested rounds up and the limits round down, like the rejection error.
			name: "over budget",
			c: rangeCheck{
				outcome: scannedBytesOutcomeRejected, requested: 2*time.Hour + 500*time.Millisecond, evaluated: true, allowed: 2 * time.Hour, derived: time.Hour + 500*time.Millisecond, scannedBytes: 300, rangeLimits: rangeLimits{minDuration: 2 * time.Hour, maxDuration: 24 * time.Hour},
			},
			wantLog: `level=info msg="scanned bytes limit" tenant=record op=search requested_seconds=7201 scanned_bytes=300 derived_seconds=3600 min_seconds=7200 max_seconds=86400 outcome=rejected passes_scanned_bytes=false`,
			wantAttrs: map[attribute.Key]attribute.Value{
				"scannedBytesLimitOutcome":        attribute.StringValue(scannedBytesOutcomeRejected),
				"scannedBytesLimitAllowedSeconds": attribute.Int64Value(7200),
			},
			wantCount:   1,
			wantHist:    1,
			wantHistSum: 3600.5,
		},
		{
			name: "fits without max_duration",
			c: rangeCheck{
				fits:    true,
				outcome: scannedBytesOutcomeWithin, requested: time.Hour, evaluated: true, scannedBytes: 100,
			},
			wantLog: `level=info msg="scanned bytes limit" tenant=record op=search requested_seconds=3600 scanned_bytes=100 fits=true min_seconds=0 max_seconds=0 outcome=within passes_scanned_bytes=true`,
			wantAttrs: map[attribute.Key]attribute.Value{
				"scannedBytesLimitOutcome":        attribute.StringValue(scannedBytesOutcomeWithin),
				"scannedBytesLimitAllowedSeconds": attribute.Int64Value(0),
			},
			wantCount: 1,
		},
		{
			name: "fits",
			c: rangeCheck{
				fits:    true,
				outcome: scannedBytesOutcomeWithin, requested: 2 * time.Hour, evaluated: true,
				allowed: 24 * time.Hour, scannedBytes: 100, rangeLimits: rangeLimits{minDuration: time.Hour, maxDuration: 24 * time.Hour},
			},
			wantLog: `level=info msg="scanned bytes limit" tenant=record op=search requested_seconds=7200 scanned_bytes=100 fits=true min_seconds=3600 max_seconds=86400 outcome=within passes_scanned_bytes=true`,
			wantAttrs: map[attribute.Key]attribute.Value{
				"scannedBytesLimitOutcome":        attribute.StringValue(scannedBytesOutcomeWithin),
				"scannedBytesLimitAllowedSeconds": attribute.Int64Value(86400),
			},
			wantCount: 1,
		},
		{
			// min_duration alone, at exactly the requested range, would allow it, so it passes on scanned bytes.
			name: "over max_duration is counted but stays out of the histogram",
			c: rangeCheck{
				outcome: scannedBytesOutcomeRejectedMaxDuration, requested: 6 * time.Hour, evaluated: true, allowed: 5 * time.Hour, derived: time.Hour, scannedBytes: 700, rangeLimits: rangeLimits{minDuration: 6 * time.Hour, maxDuration: 5 * time.Hour},
			},
			wantLog: `level=info msg="scanned bytes limit" tenant=record op=search requested_seconds=21600 scanned_bytes=700 derived_seconds=3600 min_seconds=21600 max_seconds=18000 outcome=rejected_max_duration passes_scanned_bytes=true`,
			wantAttrs: map[attribute.Key]attribute.Value{
				"scannedBytesLimitOutcome":        attribute.StringValue(scannedBytesOutcomeRejectedMaxDuration),
				"scannedBytesLimitAllowedSeconds": attribute.Int64Value(18000),
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
			c := tc.c
			c.op, c.tenantID = searchOp, "record"

			countBefore := outcomeCount("record", searchOp, c.outcome)
			histBefore, sumBefore := histogramSample(t, "record", searchOp)
			c.record(span, log.NewLogfmtLogger(&buf))
			span.End()

			require.Equal(t, tc.wantCount, outcomeCount("record", searchOp, c.outcome)-countBefore)
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

func TestMetricsScannedBytesLimitCheckMultiTenant(t *testing.T) {
	rdr := &perTenantMockReader{metas: map[string][]*backend.BlockMeta{
		"mt-a-over": hourlyMetricsBlocks(scannedBytesTestEnd),
		"mt-b-fits": nil,
	}}
	var buf bytes.Buffer
	l := newMetricsScannedBytesLimit(rdr, &mockOverridesScannedBytes{}, QueryRangeSharderConfig{MaxScannedBytes: 350}, true, log.NewLogfmtLogger(&buf))

	overBefore := outcomeCount("mt-a-over", metricsOp, scannedBytesOutcomeRejected)
	fitsBefore := outcomeCount("mt-b-fits", metricsOp, scannedBytesOutcomeWithin)
	// tenant.TenantIDs sorts, so the over-budget tenant comes first and the fitting one is evaluated after the rejection.
	err := l.check(user.InjectOrgID(context.Background(), "mt-b-fits|mt-a-over"), &tempopb.QueryRangeRequest{
		Start: uint64(hoursAgo(4).UnixNano()),
		End:   uint64(scannedBytesTestEnd.UnixNano()),
	}, nil)
	require.EqualError(t, err, "metrics query time range of 4h0m0s exceeds the maximum allowed range of 2h59m59s for tenant mt-a-over (limited by scanned bytes)")

	require.Equal(t, 1.0, outcomeCount("mt-a-over", metricsOp, scannedBytesOutcomeRejected)-overBefore)
	require.Equal(t, 1.0, outcomeCount("mt-b-fits", metricsOp, scannedBytesOutcomeWithin)-fitsBefore)
	require.Equal(t, `level=info msg="scanned bytes limit" tenant=mt-a-over op=metrics requested_seconds=14400 scanned_bytes=500 derived_seconds=10799 min_seconds=0 max_seconds=0 outcome=rejected passes_scanned_bytes=false
level=info msg="scanned bytes limit" tenant=mt-b-fits op=metrics requested_seconds=14400 scanned_bytes=0 fits=true min_seconds=0 max_seconds=0 outcome=within passes_scanned_bytes=true
`, buf.String())
}

func TestMetricsScannedBytesLimitCheckMaxErr(t *testing.T) {
	rdr := &perTenantMockReader{metas: map[string][]*backend.BlockMeta{
		"rmd-a-over": hourlyMetricsBlocks(scannedBytesTestEnd),
		"rmd-b-fits": nil,
	}}
	var buf bytes.Buffer
	l := newMetricsScannedBytesLimit(rdr, &mockOverridesScannedBytes{}, QueryRangeSharderConfig{MaxScannedBytes: 350}, true, log.NewLogfmtLogger(&buf))
	maxErr := fmt.Errorf("%w of 1h0m0s", errMetricsMaxDuration)

	histBefore, _ := histogramSample(t, "rmd-a-over", metricsOp)
	// The raw org ID's max_duration rejected it, so every tenant logs rejected_max_duration even though its own limits allow more.
	err := l.check(user.InjectOrgID(context.Background(), "rmd-b-fits|rmd-a-over"), &tempopb.QueryRangeRequest{
		Start: uint64(hoursAgo(4).UnixNano()),
		End:   uint64(scannedBytesTestEnd.UnixNano()),
	}, maxErr)
	require.Equal(t, maxErr, err)

	histAfter, _ := histogramSample(t, "rmd-a-over", metricsOp)
	require.Equal(t, histBefore, histAfter)
	require.Equal(t, `level=info msg="scanned bytes limit" tenant=rmd-a-over op=metrics requested_seconds=14400 scanned_bytes=500 derived_seconds=10799 min_seconds=0 max_seconds=0 outcome=rejected_max_duration passes_scanned_bytes=false
level=info msg="scanned bytes limit" tenant=rmd-b-fits op=metrics requested_seconds=14400 scanned_bytes=0 fits=true min_seconds=0 max_seconds=0 outcome=rejected_max_duration passes_scanned_bytes=true
`, buf.String())
}

func TestMetricsScannedBytesLimitCheckOrgID(t *testing.T) {
	req := &tempopb.QueryRangeRequest{Start: uint64(hoursAgo(4).UnixNano()), End: uint64(scannedBytesTestEnd.UnixNano())}

	t.Run("invalid org ID", func(t *testing.T) {
		l := newMetricsScannedBytesLimit(&mockReader{}, &mockOverridesScannedBytes{}, QueryRangeSharderConfig{MaxScannedBytes: 350}, true, log.NewNopLogger())
		require.Error(t, l.check(user.InjectOrgID(context.Background(), "foo bar"), req, nil))
	})

	t.Run("multi-tenant queries disabled", func(t *testing.T) {
		// The multi-tenant unsupported middleware rejects a|b later, so no tenant is checked or logged here.
		var buf bytes.Buffer
		rdr := &countingMockReader{mockReader: mockReader{metas: hourlyMetricsBlocks(scannedBytesTestEnd)}}
		l := newMetricsScannedBytesLimit(rdr, &mockOverridesScannedBytes{}, QueryRangeSharderConfig{MaxScannedBytes: 350}, false, log.NewLogfmtLogger(&buf))
		require.NoError(t, l.check(user.InjectOrgID(context.Background(), "a|b"), req, nil))
		require.Empty(t, buf.String())
		require.Zero(t, rdr.callCount())
	})
}

func TestMetricsScannedBytesLimitCheckTenantMaxDuration(t *testing.T) {
	// validateMetricsQueryMaxDuration reads the raw org ID, so a tenant's own max_duration is enforced by the limiter.
	for _, maxDuration := range []time.Duration{time.Hour, -time.Hour} {
		l := newMetricsScannedBytesLimit(&mockReader{}, &mockOverridesScannedBytes{metrics: rangeLimits{maxDuration: maxDuration}},
			QueryRangeSharderConfig{MaxScannedBytes: 1, MaxDuration: 24 * time.Hour}, true, log.NewNopLogger())

		err := l.check(user.InjectOrgID(context.Background(), "foo"), &tempopb.QueryRangeRequest{
			Start: uint64(hoursAgo(2).UnixNano()),
			End:   uint64(scannedBytesTestEnd.UnixNano()),
		}, nil)
		require.ErrorIs(t, err, errMetricsMaxDuration)
		require.EqualError(t, err, fmt.Sprintf("metrics query time range exceeds the maximum allowed duration of %s", maxDuration))
	}
}

func TestMetricsScannedBytesLimitCheckOverrides(t *testing.T) {
	// Each override must replace its config value, and the non RF1 block must not count.
	metas := append(hourlyMetricsBlocks(scannedBytesTestEnd), &backend.BlockMeta{StartTime: hoursAgo(1), EndTime: scannedBytesTestEnd, Size_: 10_000})
	var buf bytes.Buffer
	l := newMetricsScannedBytesLimit(&mockReader{metas: metas},
		&mockOverridesScannedBytes{metrics: rangeLimits{maxDuration: 6 * time.Hour, minDuration: 5 * time.Hour, maxScannedBytes: 150}},
		QueryRangeSharderConfig{MaxDuration: 24 * time.Hour, MinDuration: time.Hour, MaxScannedBytes: 10_000}, true, log.NewLogfmtLogger(&buf))

	err := l.check(user.InjectOrgID(context.Background(), "foo"), &tempopb.QueryRangeRequest{
		Start: uint64(hoursAgo(6).UnixNano()),
		End:   uint64(scannedBytesTestEnd.UnixNano()),
	}, nil)
	require.EqualError(t, err, "metrics query time range of 6h0m0s exceeds the maximum allowed range of 5h0m0s for tenant foo (limited by scanned bytes)")
	require.Contains(t, buf.String(), " scanned_bytes=700 ")
	require.Contains(t, buf.String(), " max_seconds=21600 ")
}

func TestMetricsScannedBytesLimitCheckLimitOff(t *testing.T) {
	// With the limit off the raw-org-ID max_duration check stays the only max_duration, as before this feature.
	l := newMetricsScannedBytesLimit(&mockReader{}, &mockOverridesScannedBytes{metrics: rangeLimits{maxDuration: time.Hour}},
		QueryRangeSharderConfig{MaxDuration: 24 * time.Hour}, true, log.NewNopLogger())

	require.NoError(t, l.check(user.InjectOrgID(context.Background(), "foo"), &tempopb.QueryRangeRequest{
		Start: uint64(hoursAgo(2).UnixNano()),
		End:   uint64(scannedBytesTestEnd.UnixNano()),
	}, nil))
}

func TestMetricsScannedBytesLimitCheckHugeRange(t *testing.T) {
	// A range past math.MaxInt64 nanoseconds must not wrap negative and slip under the allowed range.
	l := newMetricsScannedBytesLimit(&mockReader{metas: hourlyMetricsBlocks(scannedBytesTestEnd)}, &mockOverridesScannedBytes{},
		QueryRangeSharderConfig{MaxScannedBytes: 1}, true, log.NewNopLogger())

	err := l.check(user.InjectOrgID(context.Background(), "foo"), &tempopb.QueryRangeRequest{Start: 1, End: math.MaxUint64}, nil)
	require.ErrorContains(t, err, "(limited by scanned bytes)")
}

func TestMetricsScannedBytesLimitCheckCutoff(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// One large block straddles the cutoff at now-30m.
		now := time.Now()
		rdr := &mockReader{metas: []*backend.BlockMeta{{
			StartTime:         now.Add(-40 * time.Minute),
			EndTime:           now.Add(-time.Minute),
			Size_:             1000,
			ReplicationFactor: backend.MetricsGeneratorReplicationFactor,
		}}}
		l := newMetricsScannedBytesLimit(rdr, &mockOverridesScannedBytes{}, QueryRangeSharderConfig{MaxScannedBytes: 500, QueryBackendAfter: 30 * time.Minute}, true, log.NewNopLogger())

		// The 28m between the cutoff and the end reach no backend block, so they are allowed.
		err := l.check(user.InjectOrgID(context.Background(), "foo"), &tempopb.QueryRangeRequest{
			Start: uint64(now.Add(-time.Hour).UnixNano()),
			End:   uint64(now.Add(-2 * time.Minute).UnixNano()),
		}, nil)
		require.EqualError(t, err, "metrics query time range of 58m0s exceeds the maximum allowed range of 28m0s for tenant foo (limited by scanned bytes)")
	})
}

func BenchmarkCheckScannedBytes(b *testing.B) {
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
			l := testScannedBytesLimit(metricsOp, rangeLimits{maxDuration: 14 * 24 * time.Hour, maxScannedBytes: bc.maxScannedBytes})
			metasFn := func() []*backend.BlockMeta { return list.Metas("foo") }
			b.Run(fmt.Sprintf("blocks=%d/%s", n, bc.name), func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					checkScannedBytes(l, "foo", scannedBytesTestEnd.Add(-7*24*time.Hour), scannedBytesTestEnd, scannedBytesTestEnd, metasFn)
				}
			})
		}
	}
}
