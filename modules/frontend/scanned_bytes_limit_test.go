package frontend

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/log" //nolint:all deprecated
	"github.com/grafana/dskit/services"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/modules/overrides"
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

// nonRF1Block is large enough to reject any metrics query it counts against.
var nonRF1Block = &backend.BlockMeta{StartTime: hoursAgo(1), EndTime: scannedBytesTestEnd, Size_: 10_000}

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

// newTestPerTenantOverrides loads perTenant, a per-tenant overrides file, like the overrides module tests.
func newTestPerTenantOverrides(t testing.TB, perTenant string) overrides.Interface {
	path := filepath.Join(t.TempDir(), "overrides.yaml")
	require.NoError(t, os.WriteFile(path, []byte(perTenant), 0o600))
	o, err := overrides.NewOverrides(overrides.Config{PerTenantOverrideConfig: path, PerTenantOverridePeriod: model.Duration(time.Hour)}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	require.NoError(t, services.StartAndAwaitRunning(context.Background(), o))
	t.Cleanup(func() { require.NoError(t, services.StopAndAwaitTerminated(context.Background(), o)) })
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

func TestCheckScannedBytes(t *testing.T) {
	hourly := hourlyMetricsBlocks(scannedBytesTestEnd)

	tcs := []struct {
		name      string
		metas     []*backend.BlockMeta
		limits    rangeLimits
		requested time.Duration
		// start defaults to scannedBytesTestEnd minus requested. cutoff defaults to scannedBytesTestEnd.
		start, cutoff time.Time
		// want leaves out the fields every case sets: the limits, op, tenant and requested range.
		want    scannedBytesCheck
		wantErr string
	}{
		{
			name:      "fits",
			metas:     hourly,
			limits:    rangeLimits{maxDuration: 24 * time.Hour, maxScannedBytes: 1000},
			requested: 10 * time.Hour,
			want:      scannedBytesCheck{fits: true, result: scannedBytesResultWithin, estimatedBytes: 1000},
		},
		{
			name:      "data starting mid-range fits",
			metas:     hourly[:3],
			limits:    rangeLimits{maxDuration: 7 * 24 * time.Hour, maxScannedBytes: 1000},
			requested: 7 * 24 * time.Hour,
			want:      scannedBytesCheck{fits: true, result: scannedBytesResultWithin, estimatedBytes: 300},
		},
		{
			name:      "no blocks overlap",
			limits:    rangeLimits{maxScannedBytes: 1},
			requested: 10 * time.Hour,
			want:      scannedBytesCheck{fits: true, result: scannedBytesResultWithin},
		},
		{
			name:      "over the budget",
			metas:     hourly,
			limits:    rangeLimits{maxDuration: 24 * time.Hour, maxScannedBytes: 450},
			requested: 6 * time.Hour,
			want:      scannedBytesCheck{result: scannedBytesResultRejected, allowed: 4*time.Hour - time.Nanosecond, derived: 4*time.Hour - time.Nanosecond, estimatedBytes: 700},
			wantErr:   "metrics query time range of 6h0m0s exceeds the maximum allowed range of 3h59m59s for tenant foo (limited by scanned bytes), reduce the time range to 3h59m59s or less",
		},
		{
			name:      "requested rounds up and allowed rounds down in the error",
			metas:     hourly,
			limits:    rangeLimits{maxScannedBytes: 450},
			requested: 4*time.Hour + 500*time.Millisecond,
			want:      scannedBytesCheck{result: scannedBytesResultRejected, allowed: 4*time.Hour - time.Nanosecond, derived: 4*time.Hour - time.Nanosecond, estimatedBytes: 500},
			wantErr:   "metrics query time range of 4h0m1s exceeds the maximum allowed range of 3h59m59s for tenant foo (limited by scanned bytes), reduce the time range to 3h59m59s or less",
		},
		{
			name:      "over min_duration",
			metas:     hourly,
			limits:    rangeLimits{maxDuration: 24 * time.Hour, minDuration: 6 * time.Hour, maxScannedBytes: 150},
			requested: 7 * time.Hour,
			want:      scannedBytesCheck{result: scannedBytesResultRejected, allowed: 6 * time.Hour, derived: time.Hour - time.Nanosecond, estimatedBytes: 800},
			wantErr:   "metrics query time range of 7h0m0s exceeds the maximum allowed range of 6h0m0s for tenant foo (limited by scanned bytes), reduce the time range to 6h0m0s or less",
		},
		{
			name:      "max_duration caps min_duration",
			metas:     hourly,
			limits:    rangeLimits{maxDuration: 5 * time.Hour, minDuration: 8 * time.Hour, maxScannedBytes: 150},
			requested: 6 * time.Hour,
			want:      scannedBytesCheck{result: scannedBytesResultRejected, allowed: 5 * time.Hour, derived: time.Hour - time.Nanosecond, estimatedBytes: 700},
			wantErr:   "metrics query time range of 6h0m0s exceeds the maximum allowed range of 5h0m0s for tenant foo (limited by scanned bytes), reduce the time range to 5h0m0s or less",
		},
		{
			name:      "end before start fits",
			metas:     hourly,
			limits:    rangeLimits{maxScannedBytes: 1},
			start:     scannedBytesTestEnd.Add(time.Hour),
			requested: time.Hour,
			want:      scannedBytesCheck{fits: true, result: scannedBytesResultWithin},
		},
		{
			name:      "a block starting at the cutoff counts",
			metas:     []*backend.BlockMeta{testBlock(1, 0, 300)},
			limits:    rangeLimits{maxScannedBytes: 250},
			cutoff:    hoursAgo(1),
			requested: 2 * time.Hour,
			want:      scannedBytesCheck{result: scannedBytesResultRejected, allowed: time.Hour, derived: time.Hour, estimatedBytes: 300},
			wantErr:   "metrics query time range of 2h0m0s exceeds the maximum allowed range of 1h0m0s for tenant foo (limited by scanned bytes), reduce the time range to 1h0m0s or less",
		},
		{
			name:      "a prefix of blocks equal to the budget fits",
			metas:     []*backend.BlockMeta{testBlock(1, 0, 100), testBlock(2, 1, 150)},
			limits:    rangeLimits{maxScannedBytes: 100},
			requested: 2 * time.Hour,
			want:      scannedBytesCheck{result: scannedBytesResultRejected, allowed: time.Hour - time.Nanosecond, derived: time.Hour - time.Nanosecond, estimatedBytes: 250},
			wantErr:   "metrics query time range of 2h0m0s exceeds the maximum allowed range of 59m59s for tenant foo (limited by scanned bytes), reduce the time range to 59m59s or less",
		},
		{
			name:      "one byte over the budget",
			metas:     []*backend.BlockMeta{testBlock(1, 0, 100), testBlock(2, 1, 151)},
			limits:    rangeLimits{maxScannedBytes: 250},
			requested: 2 * time.Hour,
			want:      scannedBytesCheck{result: scannedBytesResultRejected, allowed: time.Hour - time.Nanosecond, derived: time.Hour - time.Nanosecond, estimatedBytes: 251},
			wantErr:   "metrics query time range of 2h0m0s exceeds the maximum allowed range of 59m59s for tenant foo (limited by scanned bytes), reduce the time range to 59m59s or less",
		},
		{
			name:      "one block larger than the budget",
			metas:     []*backend.BlockMeta{testBlock(1, 0, 1000)},
			limits:    rangeLimits{maxDuration: 24 * time.Hour, maxScannedBytes: 500},
			requested: time.Hour,
			want:      scannedBytesCheck{result: scannedBytesResultRejected, estimatedBytes: 1000},
			wantErr:   "metrics query time range of 1h0m0s exceeds the maximum allowed range for tenant foo (limited by scanned bytes), the newest data in the range is already over the limit",
		},
		{
			name:      "range after the cutoff is free",
			metas:     []*backend.BlockMeta{testBlock(3, 0, 300)},
			limits:    rangeLimits{maxScannedBytes: 250},
			cutoff:    hoursAgo(1),
			requested: 3 * time.Hour,
			want:      scannedBytesCheck{result: scannedBytesResultRejected, allowed: time.Hour, derived: time.Hour, estimatedBytes: 300},
			wantErr:   "metrics query time range of 3h0m0s exceeds the maximum allowed range of 1h0m0s for tenant foo (limited by scanned bytes), reduce the time range to 1h0m0s or less",
		},
		{
			// Search start and cutoff are both whole seconds, so this edge is reachable.
			name:      "range starting at the cutoff fits",
			metas:     []*backend.BlockMeta{testBlock(3, 0, 300)},
			limits:    rangeLimits{maxScannedBytes: 250},
			cutoff:    hoursAgo(1),
			requested: time.Hour,
			want:      scannedBytesCheck{fits: true, result: scannedBytesResultWithin},
		},
		{
			name:      "blocks starting after the cutoff are free",
			metas:     []*backend.BlockMeta{testBlock(3, 2, 100), {StartTime: scannedBytesTestEnd.Add(-30 * time.Minute), EndTime: scannedBytesTestEnd.Add(-10 * time.Minute), Size_: 1000, ReplicationFactor: backend.MetricsGeneratorReplicationFactor}},
			limits:    rangeLimits{maxScannedBytes: 250},
			cutoff:    hoursAgo(1),
			requested: 3 * time.Hour,
			want:      scannedBytesCheck{fits: true, result: scannedBytesResultWithin, estimatedBytes: 100},
		},
		{
			name:      "range wholly after the cutoff fits",
			metas:     []*backend.BlockMeta{testBlock(3, 0, 300)},
			limits:    rangeLimits{maxScannedBytes: 250},
			cutoff:    hoursAgo(2),
			requested: time.Hour,
			want:      scannedBytesCheck{fits: true, result: scannedBytesResultWithin},
		},
		{
			// A long compacted block starts before the fitting blocks but ends inside the range. Input order is shuffled.
			name:      "blocks enter by end time, newest first",
			metas:     []*backend.BlockMeta{testBlock(3, 2, 100), testBlock(1, 0, 100), testBlock(10, 1, 100)},
			limits:    rangeLimits{maxScannedBytes: 250},
			requested: 10 * time.Hour,
			want:      scannedBytesCheck{result: scannedBytesResultRejected, allowed: 2*time.Hour - time.Nanosecond, derived: 2*time.Hour - time.Nanosecond, estimatedBytes: 300},
			wantErr:   "metrics query time range of 10h0m0s exceeds the maximum allowed range of 1h59m59s for tenant foo (limited by scanned bytes), reduce the time range to 1h59m59s or less",
		},
		{
			name:      "blocks with the same end time enter together",
			metas:     []*backend.BlockMeta{testBlock(1, 0, 100), testBlock(2, 1, 100), testBlock(2, 1, 100)},
			limits:    rangeLimits{maxScannedBytes: 250},
			requested: 2 * time.Hour,
			want:      scannedBytesCheck{result: scannedBytesResultRejected, allowed: time.Hour - time.Nanosecond, derived: time.Hour - time.Nanosecond, estimatedBytes: 300},
			wantErr:   "metrics query time range of 2h0m0s exceeds the maximum allowed range of 59m59s for tenant foo (limited by scanned bytes), reduce the time range to 59m59s or less",
		},
		{
			name:      "blocks after the cutoff stay out of the derived range",
			metas:     []*backend.BlockMeta{{StartTime: scannedBytesTestEnd.Add(-30 * time.Minute), EndTime: scannedBytesTestEnd.Add(-10 * time.Minute), Size_: 1000, ReplicationFactor: backend.MetricsGeneratorReplicationFactor}, testBlock(3, 2, 100), testBlock(5, 4, 200)},
			limits:    rangeLimits{maxScannedBytes: 250},
			cutoff:    hoursAgo(1),
			requested: 5 * time.Hour,
			want:      scannedBytesCheck{result: scannedBytesResultRejected, allowed: 4*time.Hour - time.Nanosecond, derived: 4*time.Hour - time.Nanosecond, estimatedBytes: 300},
			wantErr:   "metrics query time range of 5h0m0s exceeds the maximum allowed range of 3h59m59s for tenant foo (limited by scanned bytes), reduce the time range to 3h59m59s or less",
		},
		{
			// A metrics cutoff is in nanoseconds, so it can land under 1s before the end.
			name:      "allowed range under 1s",
			metas:     []*backend.BlockMeta{testBlock(1, 0, 300)},
			limits:    rangeLimits{maxScannedBytes: 250},
			cutoff:    scannedBytesTestEnd.Add(-500 * time.Millisecond),
			requested: time.Hour,
			want:      scannedBytesCheck{result: scannedBytesResultRejected, allowed: 500 * time.Millisecond, derived: 500 * time.Millisecond, estimatedBytes: 300},
			wantErr:   "metrics query time range of 1h0m0s exceeds the maximum allowed range of 500ms for tenant foo (limited by scanned bytes), reduce the time range to 500ms or less",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			start := tc.start
			if start.IsZero() {
				start = scannedBytesTestEnd.Add(-tc.requested)
			}
			cutoff := tc.cutoff
			if cutoff.IsZero() {
				cutoff = scannedBytesTestEnd
			}

			s := scannedBytesLimit{op: metricsOp, reader: &mockReader{metas: tc.metas}, filterFn: isRF1Block}
			got := s.check("foo", tc.limits, start, scannedBytesTestEnd, cutoff)

			want := tc.want
			want.rangeLimits, want.op, want.tenantID, want.requested = tc.limits, metricsOp, "foo", scannedBytesTestEnd.Sub(start)
			require.Equal(t, want, got)
			if tc.wantErr == "" {
				require.NoError(t, got.err("metrics query"))
			} else {
				require.EqualError(t, got.err("metrics query"), tc.wantErr)
			}
		})
	}
}

func TestScannedBytesLimitOrder(t *testing.T) {
	errMax := errors.New("over max_duration")
	tcs := []struct {
		name      string
		limits    rangeLimits
		requested time.Duration
		wantErr   string
		// wantLog is the logged line, or empty when the blocklist isn't read.
		wantLog string
	}{
		{
			name:      "over max_duration without max_scanned_bytes reads nothing",
			limits:    rangeLimits{maxDuration: 5 * time.Hour},
			requested: 6 * time.Hour,
			wantErr:   errMax.Error(),
		},
		{
			name:      "over max_duration is still logged",
			limits:    rangeLimits{maxDuration: 5 * time.Hour, maxScannedBytes: 150},
			requested: 6 * time.Hour,
			wantErr:   errMax.Error(),
			wantLog: `level=info msg="scanned bytes limit" tenant=foo op=metrics requested_seconds=21600 estimated_scanned_bytes=700 max_scanned_bytes=150 derived_seconds=3599 allowed_seconds=3599 min_seconds=0 max_seconds=18000 result=rejected_max_duration passes_scanned_bytes=false
`,
		},
		{
			name:      "max_duration wins over min_duration",
			limits:    rangeLimits{maxDuration: 5 * time.Hour, minDuration: 8 * time.Hour, maxScannedBytes: 150},
			requested: 6 * time.Hour,
			wantErr:   errMax.Error(),
			wantLog: `level=info msg="scanned bytes limit" tenant=foo op=metrics requested_seconds=21600 estimated_scanned_bytes=700 max_scanned_bytes=150 derived_seconds=3599 allowed_seconds=18000 min_seconds=28800 max_seconds=18000 result=rejected_max_duration passes_scanned_bytes=true
`,
		},
		{
			name:      "within min_duration reads nothing",
			limits:    rangeLimits{maxDuration: 24 * time.Hour, minDuration: 6 * time.Hour, maxScannedBytes: 150},
			requested: 6 * time.Hour,
		},
		{
			name:      "max_scanned_bytes off reads nothing",
			limits:    rangeLimits{maxDuration: 24 * time.Hour},
			requested: 6 * time.Hour,
		},
		{
			name:      "fits",
			limits:    rangeLimits{maxScannedBytes: 1000},
			requested: 10 * time.Hour,
			wantLog: `level=info msg="scanned bytes limit" tenant=foo op=metrics requested_seconds=36000 estimated_scanned_bytes=1000 max_scanned_bytes=1000 fits=true min_seconds=0 max_seconds=0 result=within passes_scanned_bytes=true
`,
		},
		{
			name:      "over the budget",
			limits:    rangeLimits{maxScannedBytes: 150},
			requested: 6 * time.Hour,
			wantErr:   "metrics query time range of 6h0m0s exceeds the maximum allowed range of 59m59s for tenant foo (limited by scanned bytes), reduce the time range to 59m59s or less",
			wantLog: `level=info msg="scanned bytes limit" tenant=foo op=metrics requested_seconds=21600 estimated_scanned_bytes=700 max_scanned_bytes=150 derived_seconds=3599 allowed_seconds=3599 min_seconds=0 max_seconds=0 result=rejected passes_scanned_bytes=false
`,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			rdr := &countingMockReader{mockReader: mockReader{metas: hourlyMetricsBlocks(scannedBytesTestEnd)}}
			s := scannedBytesLimit{
				op:       metricsOp,
				kind:     "metrics query",
				reader:   rdr,
				filterFn: isRF1Block,
				maxErr:   func(rangeLimits, time.Time, time.Time) error { return errMax },
				logger:   log.NewLogfmtLogger(&buf),
			}

			_, err := s.limit(context.Background(), "foo", tc.limits, scannedBytesTestEnd.Add(-tc.requested), scannedBytesTestEnd, scannedBytesTestEnd)
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tc.wantErr)
			}
			if tc.wantLog == "" {
				require.Zero(t, rdr.callCount())
				require.Empty(t, buf.String())
			} else {
				require.Equal(t, 1, rdr.callCount())
				require.Equal(t, tc.wantLog, buf.String())
			}
		})
	}
}

func TestRecordScannedBytesLimit(t *testing.T) {
	tcs := []struct {
		name     string
		c        scannedBytesCheck
		wantLog  string
		wantHist uint64
	}{
		{
			// Requested rounds up and the limits round down, like the rejection error.
			name: "over the budget",
			c: scannedBytesCheck{
				result: scannedBytesResultRejected, requested: 2*time.Hour + 500*time.Millisecond, allowed: 2 * time.Hour,
				derived: time.Hour + 500*time.Millisecond, estimatedBytes: 300, rangeLimits: rangeLimits{minDuration: 2 * time.Hour, maxDuration: 24 * time.Hour, maxScannedBytes: 250},
			},
			wantLog:  `level=info msg="scanned bytes limit" tenant=record op=search requested_seconds=7201 estimated_scanned_bytes=300 max_scanned_bytes=250 derived_seconds=3600 allowed_seconds=7200 min_seconds=7200 max_seconds=86400 result=rejected passes_scanned_bytes=false`,
			wantHist: 1,
		},
		{
			name:    "fits",
			c:       scannedBytesCheck{fits: true, result: scannedBytesResultWithin, requested: time.Hour, estimatedBytes: 100, rangeLimits: rangeLimits{maxScannedBytes: 250}},
			wantLog: `level=info msg="scanned bytes limit" tenant=record op=search requested_seconds=3600 estimated_scanned_bytes=100 max_scanned_bytes=250 fits=true min_seconds=0 max_seconds=0 result=within passes_scanned_bytes=true`,
		},
		{
			// min_duration alone, at exactly the requested range, would allow it, so it passes on scanned bytes.
			name: "over max_duration stays out of the histogram",
			c: scannedBytesCheck{
				result: scannedBytesResultRejectedMaxDuration, requested: 6 * time.Hour, allowed: 5 * time.Hour,
				derived: time.Hour, estimatedBytes: 700, rangeLimits: rangeLimits{minDuration: 6 * time.Hour, maxDuration: 5 * time.Hour, maxScannedBytes: 250},
			},
			wantLog: `level=info msg="scanned bytes limit" tenant=record op=search requested_seconds=21600 estimated_scanned_bytes=700 max_scanned_bytes=250 derived_seconds=3600 allowed_seconds=18000 min_seconds=21600 max_seconds=18000 result=rejected_max_duration passes_scanned_bytes=true`,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			c := tc.c
			c.op, c.tenantID = searchOp, "record"

			countBefore, histBefore := resultCount("record", searchOp, c.result), histogramCount(t, "record", searchOp)
			recordScannedBytesLimit(context.Background(), log.NewLogfmtLogger(&buf), c)

			require.Equal(t, tc.wantHist, histogramCount(t, "record", searchOp)-histBefore)
			require.Equal(t, tc.wantLog+"\n", buf.String())
			require.Equal(t, countBefore+1, resultCount("record", searchOp, c.result))
		})
	}
}

// blocklistReader reads a real blocklist, so the benchmark pays the lock and copy that BlockMetas does per query.
type blocklistReader struct {
	mockReader
	list *blocklist.List
}

func (r *blocklistReader) BlockMetas(tenantID string) []*backend.BlockMeta {
	return r.list.Metas(tenantID)
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
		list := blocklist.New()
		list.ApplyPollResults(blocklist.PerTenant{"foo": metas}, nil, nil)
		for _, bc := range []struct {
			name            string
			maxScannedBytes uint64
		}{
			{name: "fits", maxScannedBytes: math.MaxUint64},
			{name: "over_budget", maxScannedBytes: uint64(n) * blockSize / 4},
		} {
			l := rangeLimits{maxDuration: 14 * 24 * time.Hour, maxScannedBytes: bc.maxScannedBytes}
			s := scannedBytesLimit{op: metricsOp, reader: &blocklistReader{list: list}, filterFn: isRF1Block}
			b.Run(fmt.Sprintf("blocks=%d/%s", n, bc.name), func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					s.check("foo", l, scannedBytesTestEnd.Add(-7*24*time.Hour), scannedBytesTestEnd, scannedBytesTestEnd)
				}
			})
		}
	}
}
