package frontend

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/go-kit/log" //nolint:all deprecated
	"github.com/grafana/dskit/user"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/model"
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

func newTestOverrides(t testing.TB, read overrides.ReadOverrides) overrides.Interface {
	o, err := overrides.NewOverrides(overrides.Config{Defaults: overrides.Overrides{Read: read}}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	return o
}

func histogramCount(t *testing.T, tenant, op string) uint64 {
	m := &dto.Metric{}
	require.NoError(t, scannedBytesLimitDerivedRange.WithLabelValues(tenant, op).(prometheus.Histogram).Write(m))
	return m.GetHistogram().GetSampleCount()
}

func resultCount(tenant, op, result string) float64 {
	return testutil.ToFloat64(scannedBytesLimitTotal.WithLabelValues(tenant, op, result))
}

// testScannedBytesLimit returns op's limit with l set in its config.
func testScannedBytesLimit(t testing.TB, op string, l rangeLimits) scannedBytesLimit {
	o := newTestOverrides(t, overrides.ReadOverrides{})
	if op == searchOp {
		return searchScannedBytesLimit{cfg: SearchSharderConfig{MinDuration: l.minDuration, MaxDuration: l.maxDuration, MaxScannedBytes: l.maxScannedBytes}, overrides: o}
	}
	return newMetricsScannedBytesLimit(nil, o, QueryRangeSharderConfig{MinDuration: l.minDuration, MaxDuration: l.maxDuration, MaxScannedBytes: l.maxScannedBytes}, true, nil)
}

func TestScannedBytesLimitOverrides(t *testing.T) {
	cfg := rangeLimits{minDuration: time.Hour, maxDuration: 24 * time.Hour, maxScannedBytes: 1000}
	want := rangeLimits{minDuration: 2 * time.Hour, maxDuration: 12 * time.Hour, maxScannedBytes: 500}

	search := searchScannedBytesLimit{
		cfg: SearchSharderConfig{MinDuration: cfg.minDuration, MaxDuration: cfg.maxDuration, MaxScannedBytes: cfg.maxScannedBytes},
		overrides: newTestOverrides(t, overrides.ReadOverrides{
			MinSearchDuration: model.Duration(want.minDuration), MaxSearchDuration: model.Duration(want.maxDuration), MaxSearchScannedBytes: want.maxScannedBytes,
		}),
	}
	metrics := newMetricsScannedBytesLimit(nil, newTestOverrides(t, overrides.ReadOverrides{
		MinMetricsDuration: model.Duration(want.minDuration), MaxMetricsDuration: model.Duration(want.maxDuration), MaxMetricsScannedBytes: want.maxScannedBytes,
	}), QueryRangeSharderConfig{MinDuration: cfg.minDuration, MaxDuration: cfg.maxDuration, MaxScannedBytes: cfg.maxScannedBytes}, true, nil)

	require.Equal(t, cfg, testScannedBytesLimit(t, searchOp, cfg).limits("foo"))
	require.Equal(t, cfg, testScannedBytesLimit(t, metricsOp, cfg).limits("foo"))
	require.Equal(t, want, search.limits("foo"))
	require.Equal(t, want, metrics.limits("foo"))
}

func TestCheckScannedBytes(t *testing.T) {
	hourly := make([]*backend.BlockMeta, 0, 10)
	for i := range 10 {
		hourly = append(hourly, testBlock(i+1, i, 100))
	}
	nonRF1 := &backend.BlockMeta{StartTime: hoursAgo(1), EndTime: scannedBytesTestEnd, Size_: 10_000}

	tcs := []struct {
		name      string
		op        string
		metas     []*backend.BlockMeta
		limits    rangeLimits
		requested time.Duration
		// start defaults to scannedBytesTestEnd minus requested. cutoff defaults to scannedBytesTestEnd.
		start, cutoff time.Time
		// want leaves out the fields every case sets: the limits, op, tenant and requested range.
		want    rangeCheck
		wantErr string
	}{
		{
			name:      "limit off allows any range",
			metas:     hourly,
			limits:    rangeLimits{maxDuration: 24 * time.Hour},
			requested: 8 * time.Hour,
			want:      rangeCheck{fits: true, result: scannedBytesResultWithin},
		},
		{
			name:      "limit off still rejects over max_duration",
			op:        searchOp,
			metas:     hourly,
			limits:    rangeLimits{maxDuration: 5 * time.Hour},
			requested: 6 * time.Hour,
			want:      rangeCheck{fits: true, result: scannedBytesResultRejectedMaxDuration},
		},
		{
			name:      "fits",
			metas:     hourly,
			limits:    rangeLimits{maxDuration: 24 * time.Hour, maxScannedBytes: 1000},
			requested: 10 * time.Hour,
			want:      rangeCheck{evaluated: true, fits: true, result: scannedBytesResultWithin, allowed: 24 * time.Hour, estimatedBytes: 1000},
		},
		{
			name:      "data starting mid-range fits",
			metas:     hourly[:3],
			limits:    rangeLimits{maxDuration: 7 * 24 * time.Hour, maxScannedBytes: 1000},
			requested: 7 * 24 * time.Hour,
			want:      rangeCheck{evaluated: true, fits: true, result: scannedBytesResultWithin, allowed: 7 * 24 * time.Hour, estimatedBytes: 300},
		},
		{
			name:      "no blocks overlap",
			limits:    rangeLimits{maxScannedBytes: 1},
			requested: 10 * time.Hour,
			want:      rangeCheck{evaluated: true, fits: true, result: scannedBytesResultWithin},
		},
		{
			name:      "over the budget",
			metas:     hourly,
			limits:    rangeLimits{maxDuration: 24 * time.Hour, maxScannedBytes: 450},
			requested: 6 * time.Hour,
			want:      rangeCheck{evaluated: true, result: scannedBytesResultRejected, allowed: 4*time.Hour - time.Nanosecond, derived: 4*time.Hour - time.Nanosecond, estimatedBytes: 700},
			wantErr:   "metrics query time range of 6h0m0s exceeds the maximum allowed range of 3h59m59s for tenant foo (limited by scanned bytes), reduce the time range to 3h59m59s or less",
		},
		{
			name:      "requested rounds up and allowed rounds down in the error",
			metas:     hourly,
			limits:    rangeLimits{maxScannedBytes: 450},
			requested: 4*time.Hour + 500*time.Millisecond,
			want:      rangeCheck{evaluated: true, result: scannedBytesResultRejected, allowed: 4*time.Hour - time.Nanosecond, derived: 4*time.Hour - time.Nanosecond, estimatedBytes: 500},
			wantErr:   "metrics query time range of 4h0m1s exceeds the maximum allowed range of 3h59m59s for tenant foo (limited by scanned bytes), reduce the time range to 3h59m59s or less",
		},
		{
			name:      "raised to min_duration",
			metas:     hourly,
			limits:    rangeLimits{maxDuration: 24 * time.Hour, minDuration: 6 * time.Hour, maxScannedBytes: 150},
			requested: 6 * time.Hour,
			want:      rangeCheck{evaluated: true, result: scannedBytesResultRaisedToMin, allowed: 6 * time.Hour, derived: time.Hour - time.Nanosecond, estimatedBytes: 700},
		},
		{
			name:      "over min_duration",
			metas:     hourly,
			limits:    rangeLimits{maxDuration: 24 * time.Hour, minDuration: 6 * time.Hour, maxScannedBytes: 150},
			requested: 7 * time.Hour,
			want:      rangeCheck{evaluated: true, result: scannedBytesResultRejected, allowed: 6 * time.Hour, derived: time.Hour - time.Nanosecond, estimatedBytes: 800},
			wantErr:   "metrics query time range of 7h0m0s exceeds the maximum allowed range of 6h0m0s for tenant foo (limited by scanned bytes), reduce the time range to 6h0m0s or less",
		},
		{
			name:      "max_duration below min_duration wins",
			metas:     hourly,
			limits:    rangeLimits{maxDuration: 5 * time.Hour, minDuration: 8 * time.Hour, maxScannedBytes: 150},
			requested: 5 * time.Hour,
			want:      rangeCheck{evaluated: true, result: scannedBytesResultRaisedToMin, allowed: 5 * time.Hour, derived: time.Hour - time.Nanosecond, estimatedBytes: 600},
		},
		{
			name:      "over max_duration is still evaluated for the log",
			metas:     hourly,
			limits:    rangeLimits{maxDuration: 5 * time.Hour, maxScannedBytes: 150},
			requested: 6 * time.Hour,
			want:      rangeCheck{evaluated: true, result: scannedBytesResultRejectedMaxDuration, allowed: time.Hour - time.Nanosecond, derived: time.Hour - time.Nanosecond, estimatedBytes: 700},
		},
		{
			name:      "negative max_duration fails closed and is not evaluated",
			metas:     hourly,
			limits:    rangeLimits{maxDuration: -time.Hour, maxScannedBytes: 150},
			requested: time.Hour,
			want:      rangeCheck{fits: true, result: scannedBytesResultRejectedMaxDuration},
		},
		{
			name:      "zero start is not evaluated",
			metas:     hourly,
			limits:    rangeLimits{maxScannedBytes: 1},
			start:     time.Unix(0, 0),
			requested: time.Hour,
			want:      rangeCheck{fits: true, result: scannedBytesResultWithin},
		},
		{
			name:      "end before start is not evaluated",
			metas:     hourly,
			limits:    rangeLimits{maxScannedBytes: 1},
			start:     scannedBytesTestEnd.Add(time.Hour),
			requested: time.Hour,
			want:      rangeCheck{fits: true, result: scannedBytesResultWithin},
		},
		{
			name:      "a block starting at the cutoff counts",
			metas:     []*backend.BlockMeta{testBlock(1, 0, 300)},
			limits:    rangeLimits{maxScannedBytes: 250},
			cutoff:    hoursAgo(1),
			requested: 2 * time.Hour,
			want:      rangeCheck{evaluated: true, result: scannedBytesResultRejected, allowed: time.Hour, derived: time.Hour, estimatedBytes: 300},
			wantErr:   "metrics query time range of 2h0m0s exceeds the maximum allowed range of 1h0m0s for tenant foo (limited by scanned bytes), reduce the time range to 1h0m0s or less",
		},
		{
			name:      "a prefix of blocks equal to the budget fits",
			metas:     []*backend.BlockMeta{testBlock(1, 0, 100), testBlock(2, 1, 150)},
			limits:    rangeLimits{maxScannedBytes: 100},
			requested: 2 * time.Hour,
			want:      rangeCheck{evaluated: true, result: scannedBytesResultRejected, allowed: time.Hour - time.Nanosecond, derived: time.Hour - time.Nanosecond, estimatedBytes: 250},
			wantErr:   "metrics query time range of 2h0m0s exceeds the maximum allowed range of 59m59s for tenant foo (limited by scanned bytes), reduce the time range to 59m59s or less",
		},
		{
			name:      "one byte over the budget",
			metas:     []*backend.BlockMeta{testBlock(1, 0, 100), testBlock(2, 1, 151)},
			limits:    rangeLimits{maxScannedBytes: 250},
			requested: 2 * time.Hour,
			want:      rangeCheck{evaluated: true, result: scannedBytesResultRejected, allowed: time.Hour - time.Nanosecond, derived: time.Hour - time.Nanosecond, estimatedBytes: 251},
			wantErr:   "metrics query time range of 2h0m0s exceeds the maximum allowed range of 59m59s for tenant foo (limited by scanned bytes), reduce the time range to 59m59s or less",
		},
		{
			name:      "one block larger than the budget",
			metas:     []*backend.BlockMeta{testBlock(1, 0, 1000)},
			limits:    rangeLimits{maxDuration: 24 * time.Hour, maxScannedBytes: 500},
			requested: time.Hour,
			want:      rangeCheck{evaluated: true, result: scannedBytesResultRejected, estimatedBytes: 1000},
			wantErr:   "metrics query time range of 1h0m0s exceeds the maximum allowed range of 0s for tenant foo (limited by scanned bytes), reduce the time range to 0s or less",
		},
		{
			name:      "range after the cutoff is free",
			metas:     []*backend.BlockMeta{testBlock(3, 0, 300)},
			limits:    rangeLimits{maxScannedBytes: 250},
			cutoff:    hoursAgo(1),
			requested: 3 * time.Hour,
			want:      rangeCheck{evaluated: true, result: scannedBytesResultRejected, allowed: time.Hour, derived: time.Hour, estimatedBytes: 300},
			wantErr:   "metrics query time range of 3h0m0s exceeds the maximum allowed range of 1h0m0s for tenant foo (limited by scanned bytes), reduce the time range to 1h0m0s or less",
		},
		{
			// Search start and cutoff are both whole seconds, so this edge is reachable.
			name:      "range starting at the cutoff fits",
			metas:     []*backend.BlockMeta{testBlock(3, 0, 300)},
			limits:    rangeLimits{maxScannedBytes: 250},
			cutoff:    hoursAgo(1),
			requested: time.Hour,
			want:      rangeCheck{evaluated: true, fits: true, result: scannedBytesResultWithin},
		},
		{
			name:      "range wholly after the cutoff fits",
			metas:     []*backend.BlockMeta{testBlock(3, 0, 300)},
			limits:    rangeLimits{maxScannedBytes: 250},
			cutoff:    hoursAgo(2),
			requested: time.Hour,
			want:      rangeCheck{evaluated: true, fits: true, result: scannedBytesResultWithin},
		},
		{
			// A long compacted block starts before the fitting blocks but ends inside the range. Input order is shuffled.
			name:      "blocks enter by end time, newest first",
			metas:     []*backend.BlockMeta{testBlock(3, 2, 100), testBlock(1, 0, 100), testBlock(10, 1, 100)},
			limits:    rangeLimits{maxScannedBytes: 250},
			requested: 10 * time.Hour,
			want:      rangeCheck{evaluated: true, result: scannedBytesResultRejected, allowed: 2*time.Hour - time.Nanosecond, derived: 2*time.Hour - time.Nanosecond, estimatedBytes: 300},
			wantErr:   "metrics query time range of 10h0m0s exceeds the maximum allowed range of 1h59m59s for tenant foo (limited by scanned bytes), reduce the time range to 1h59m59s or less",
		},
		{
			name:      "blocks with the same end time enter together",
			metas:     []*backend.BlockMeta{testBlock(1, 0, 100), testBlock(2, 1, 100), testBlock(2, 1, 100)},
			limits:    rangeLimits{maxScannedBytes: 250},
			requested: 2 * time.Hour,
			want:      rangeCheck{evaluated: true, result: scannedBytesResultRejected, allowed: time.Hour - time.Nanosecond, derived: time.Hour - time.Nanosecond, estimatedBytes: 300},
			wantErr:   "metrics query time range of 2h0m0s exceeds the maximum allowed range of 59m59s for tenant foo (limited by scanned bytes), reduce the time range to 59m59s or less",
		},
		{
			name:      "metrics counts only RF1 blocks",
			metas:     append([]*backend.BlockMeta{nonRF1}, hourly...),
			limits:    rangeLimits{maxScannedBytes: 350},
			requested: 2 * time.Hour,
			want:      rangeCheck{evaluated: true, fits: true, result: scannedBytesResultWithin, estimatedBytes: 300},
		},
		{
			name:      "search counts every block",
			op:        searchOp,
			metas:     append([]*backend.BlockMeta{nonRF1}, hourly...),
			limits:    rangeLimits{maxScannedBytes: 350},
			requested: 2 * time.Hour,
			want:      rangeCheck{evaluated: true, result: scannedBytesResultRejected, estimatedBytes: 10_300},
			wantErr:   "search time range of 2h0m0s exceeds the maximum allowed range of 0s for tenant foo (limited by scanned bytes), reduce the time range to 0s or less",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			op, kind := cmp.Or(tc.op, metricsOp), "metrics query"
			if op == searchOp {
				kind = "search"
			}
			start := tc.start
			if start.IsZero() {
				start = scannedBytesTestEnd.Add(-tc.requested)
			}
			cutoff := tc.cutoff
			if cutoff.IsZero() {
				cutoff = scannedBytesTestEnd
			}

			got := checkScannedBytes(testScannedBytesLimit(t, op, tc.limits), "foo", start, scannedBytesTestEnd, cutoff,
				func() []*backend.BlockMeta { return tc.metas })

			want := tc.want
			want.rangeLimits, want.op, want.tenantID, want.requested = tc.limits, op, "foo", scannedBytesTestEnd.Sub(start)
			require.Equal(t, want, got)
			if tc.wantErr == "" {
				require.NoError(t, got.err(kind))
			} else {
				require.EqualError(t, got.err(kind), tc.wantErr)
			}
		})
	}
}

func TestRecordScannedBytesLimit(t *testing.T) {
	tcs := []struct {
		name     string
		c        rangeCheck
		wantLog  string
		wantHist uint64
	}{
		{
			name: "not evaluated records nothing",
			c:    rangeCheck{fits: true, result: scannedBytesResultRejectedMaxDuration},
		},
		{
			// Requested rounds up and the limits round down, like the rejection error.
			name: "over the budget",
			c: rangeCheck{
				evaluated: true, result: scannedBytesResultRejected, requested: 2*time.Hour + 500*time.Millisecond, allowed: 2 * time.Hour,
				derived: time.Hour + 500*time.Millisecond, estimatedBytes: 300, rangeLimits: rangeLimits{minDuration: 2 * time.Hour, maxDuration: 24 * time.Hour, maxScannedBytes: 250},
			},
			wantLog:  `level=info msg="scanned bytes limit" tenant=record op=search requested_seconds=7201 estimated_scanned_bytes=300 max_scanned_bytes=250 derived_seconds=3600 min_seconds=7200 max_seconds=86400 result=rejected passes_scanned_bytes=false`,
			wantHist: 1,
		},
		{
			name:    "fits",
			c:       rangeCheck{evaluated: true, fits: true, result: scannedBytesResultWithin, requested: time.Hour, estimatedBytes: 100, rangeLimits: rangeLimits{maxScannedBytes: 250}},
			wantLog: `level=info msg="scanned bytes limit" tenant=record op=search requested_seconds=3600 estimated_scanned_bytes=100 max_scanned_bytes=250 fits=true min_seconds=0 max_seconds=0 result=within passes_scanned_bytes=true`,
		},
		{
			// min_duration alone, at exactly the requested range, would allow it, so it passes on scanned bytes.
			name: "over max_duration stays out of the histogram",
			c: rangeCheck{
				evaluated: true, result: scannedBytesResultRejectedMaxDuration, requested: 6 * time.Hour, allowed: 5 * time.Hour,
				derived: time.Hour, estimatedBytes: 700, rangeLimits: rangeLimits{minDuration: 6 * time.Hour, maxDuration: 5 * time.Hour, maxScannedBytes: 250},
			},
			wantLog: `level=info msg="scanned bytes limit" tenant=record op=search requested_seconds=21600 estimated_scanned_bytes=700 max_scanned_bytes=250 derived_seconds=3600 min_seconds=21600 max_seconds=18000 result=rejected_max_duration passes_scanned_bytes=true`,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			rec := tracetest.NewSpanRecorder()
			ctx, span := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec)).Tracer("test").Start(context.Background(), "test")
			var buf bytes.Buffer
			c := tc.c
			c.op, c.tenantID = searchOp, "record"

			countBefore, histBefore := resultCount("record", searchOp, c.result), histogramCount(t, "record", searchOp)
			recordScannedBytesLimit(ctx, log.NewLogfmtLogger(&buf), c)
			span.End()

			require.Equal(t, tc.wantHist, histogramCount(t, "record", searchOp)-histBefore)
			if tc.wantLog == "" {
				require.Empty(t, buf.String())
				require.Equal(t, countBefore, resultCount("record", searchOp, c.result))
				require.Empty(t, rec.Ended()[0].Attributes())
				return
			}
			require.Equal(t, tc.wantLog+"\n", buf.String())
			require.Equal(t, countBefore+1, resultCount("record", searchOp, c.result))
			require.Contains(t, rec.Ended()[0].Attributes(), attribute.String("result", c.result))
		})
	}
}

func TestMetricsScannedBytesLimitCheck(t *testing.T) {
	maxErr := fmt.Errorf("%w of 1h0m0s", errMetricsMaxDuration)
	metas := map[string][]*backend.BlockMeta{
		"foo":    hourlyMetricsBlocks(scannedBytesTestEnd),
		"a-over": hourlyMetricsBlocks(scannedBytesTestEnd),
		"b-fits": nil,
		// The non RF1 block must not count.
		"mixed": append(hourlyMetricsBlocks(scannedBytesTestEnd), &backend.BlockMeta{StartTime: hoursAgo(1), EndTime: scannedBytesTestEnd, Size_: 10_000}),
	}

	tcs := []struct {
		name                       string
		orgID                      string
		cfg                        QueryRangeSharderConfig
		overrides                  overrides.ReadOverrides
		multiTenantQueriesDisabled bool
		maxErr                     error
		// start defaults to 4h before scannedBytesTestEnd.
		start, end uint64
		wantErr    string
		wantLog    string
	}{
		{
			// tenant.TenantIDs sorts, so the over-budget tenant comes first and the fitting one is checked after the rejection.
			name:    "every tenant is logged and the first rejection wins",
			orgID:   "b-fits|a-over",
			cfg:     QueryRangeSharderConfig{MaxScannedBytes: 350},
			wantErr: "metrics query time range of 4h0m0s exceeds the maximum allowed range of 2h59m59s for tenant a-over (limited by scanned bytes), reduce the time range to 2h59m59s or less",
			wantLog: `level=info msg="scanned bytes limit" tenant=a-over op=metrics requested_seconds=14400 estimated_scanned_bytes=500 max_scanned_bytes=350 derived_seconds=10799 min_seconds=0 max_seconds=0 result=rejected passes_scanned_bytes=false
level=info msg="scanned bytes limit" tenant=b-fits op=metrics requested_seconds=14400 estimated_scanned_bytes=0 max_scanned_bytes=350 fits=true min_seconds=0 max_seconds=0 result=within passes_scanned_bytes=true
`,
		},
		{
			// The raw org ID's max_duration rejected it, so every tenant logs rejected_max_duration even though its own limits allow more.
			name:    "maxErr wins and marks every tenant",
			orgID:   "b-fits|a-over",
			cfg:     QueryRangeSharderConfig{MaxScannedBytes: 350},
			maxErr:  maxErr,
			wantErr: maxErr.Error(),
			wantLog: `level=info msg="scanned bytes limit" tenant=a-over op=metrics requested_seconds=14400 estimated_scanned_bytes=500 max_scanned_bytes=350 derived_seconds=10799 min_seconds=0 max_seconds=0 result=rejected_max_duration passes_scanned_bytes=false
level=info msg="scanned bytes limit" tenant=b-fits op=metrics requested_seconds=14400 estimated_scanned_bytes=0 max_scanned_bytes=350 fits=true min_seconds=0 max_seconds=0 result=rejected_max_duration passes_scanned_bytes=true
`,
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
			// validateMetricsQueryMaxDuration reads the raw org ID, so the limit enforces a tenant's own max_duration.
			name:      "tenant max_duration",
			cfg:       QueryRangeSharderConfig{MaxScannedBytes: 1, MaxDuration: 24 * time.Hour},
			overrides: overrides.ReadOverrides{MaxMetricsDuration: model.Duration(time.Hour)},
			wantErr:   "metrics query time range exceeds the maximum allowed duration of 1h0m0s",
		},
		{
			name:      "negative tenant max_duration",
			cfg:       QueryRangeSharderConfig{MaxScannedBytes: 1, MaxDuration: 24 * time.Hour},
			overrides: overrides.ReadOverrides{MaxMetricsDuration: model.Duration(-time.Hour)},
			wantErr:   "metrics query time range exceeds the maximum allowed duration of -1h0m0s",
		},
		{
			// With the limit off the raw org ID's max_duration check stays the only one, as before this limit.
			name:      "limit off ignores the tenant max_duration",
			cfg:       QueryRangeSharderConfig{MaxDuration: 24 * time.Hour},
			overrides: overrides.ReadOverrides{MaxMetricsDuration: model.Duration(time.Hour)},
		},
		{
			// Each override must replace its config value.
			name:  "overrides beat the config",
			orgID: "mixed",
			cfg:   QueryRangeSharderConfig{MaxDuration: 24 * time.Hour, MinDuration: time.Hour, MaxScannedBytes: 10_000},
			overrides: overrides.ReadOverrides{
				MaxMetricsDuration: model.Duration(6 * time.Hour), MinMetricsDuration: model.Duration(5 * time.Hour), MaxMetricsScannedBytes: 150,
			},
			start:   uint64(hoursAgo(6).UnixNano()),
			wantErr: "metrics query time range of 6h0m0s exceeds the maximum allowed range of 5h0m0s for tenant mixed (limited by scanned bytes), reduce the time range to 5h0m0s or less",
			wantLog: `level=info msg="scanned bytes limit" tenant=mixed op=metrics requested_seconds=21600 estimated_scanned_bytes=700 max_scanned_bytes=150 derived_seconds=3599 min_seconds=18000 max_seconds=21600 result=rejected passes_scanned_bytes=false
`,
		},
		{
			// A range past math.MaxInt64 nanoseconds must not wrap negative and slip under the allowed range.
			name:    "huge range",
			cfg:     QueryRangeSharderConfig{MaxScannedBytes: 1},
			start:   1,
			end:     math.MaxUint64,
			wantErr: "metrics query time range of 2562047h47m16s exceeds the maximum allowed range of 2561047h47m16s for tenant foo (limited by scanned bytes), reduce the time range to 2561047h47m16s or less",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := newMetricsScannedBytesLimit(&perTenantMockReader{metas: metas}, newTestOverrides(t, tc.overrides), tc.cfg,
				!tc.multiTenantQueriesDisabled, log.NewLogfmtLogger(&buf))
			req := &tempopb.QueryRangeRequest{Start: cmp.Or(tc.start, uint64(hoursAgo(4).UnixNano())), End: cmp.Or(tc.end, uint64(scannedBytesTestEnd.UnixNano()))}

			err := l.check(user.InjectOrgID(context.Background(), cmp.Or(tc.orgID, "foo")), req, tc.maxErr)
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
		})
	}
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
		l := newMetricsScannedBytesLimit(rdr, newTestOverrides(t, overrides.ReadOverrides{}),
			QueryRangeSharderConfig{MaxScannedBytes: 500, QueryBackendAfter: 30 * time.Minute}, true, log.NewNopLogger())

		// The 28m between the cutoff and the end reach no backend block, so they are allowed.
		err := l.check(user.InjectOrgID(context.Background(), "foo"), &tempopb.QueryRangeRequest{
			Start: uint64(now.Add(-time.Hour).UnixNano()),
			End:   uint64(now.Add(-2 * time.Minute).UnixNano()),
		}, nil)
		require.EqualError(t, err, "metrics query time range of 58m0s exceeds the maximum allowed range of 28m0s for tenant foo (limited by scanned bytes), reduce the time range to 28m0s or less")
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
			l := testScannedBytesLimit(b, metricsOp, rangeLimits{maxDuration: 14 * 24 * time.Hour, maxScannedBytes: bc.maxScannedBytes})
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
