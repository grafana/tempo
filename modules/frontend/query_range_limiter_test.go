package frontend

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/log" //nolint:all deprecated
	"github.com/grafana/dskit/services"
	"github.com/grafana/dskit/user"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/grafana/tempo/v3/modules/frontend/pipeline"
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
	require.NoError(t, scannedBytesLimitBudgetRange.WithLabelValues(tenant, op).(prometheus.Histogram).Write(m))
	return m.GetHistogram().GetSampleCount()
}

func resultCount(tenant, op, result string) float64 {
	return testutil.ToFloat64(scannedBytesLimitTotal.WithLabelValues(tenant, op, result))
}

func TestEstimateBackendBytes(t *testing.T) {
	hourly := hourlyMetricsBlocks(scannedBytesTestEnd)

	tcs := []struct {
		name      string
		metas     []*backend.BlockMeta
		limits    rangeLimits
		requested time.Duration
		// start defaults to scannedBytesTestEnd minus requested. backendAfter defaults to scannedBytesTestEnd.
		start, backendAfter time.Time
		want                scannedBytesCheck
		wantErr             string
	}{
		{
			name:      "fits",
			metas:     hourly,
			limits:    rangeLimits{maxDuration: 24 * time.Hour, maxScannedBytes: 1000},
			requested: 10 * time.Hour,
			want:      scannedBytesCheck{withinBudget: true, estimatedBytes: 1000},
		},
		{
			name:      "data starting mid-range fits",
			metas:     hourly[:3],
			limits:    rangeLimits{maxDuration: 7 * 24 * time.Hour, maxScannedBytes: 1000},
			requested: 7 * 24 * time.Hour,
			want:      scannedBytesCheck{withinBudget: true, estimatedBytes: 300},
		},
		{
			name:      "no blocks overlap",
			limits:    rangeLimits{maxScannedBytes: 1},
			requested: 10 * time.Hour,
			want:      scannedBytesCheck{withinBudget: true},
		},
		{
			name:      "over the budget",
			metas:     hourly,
			limits:    rangeLimits{maxDuration: 24 * time.Hour, maxScannedBytes: 450},
			requested: 6 * time.Hour,
			want:      scannedBytesCheck{budgetRange: 4*time.Hour - time.Nanosecond, estimatedBytes: 700, allowedRange: 4*time.Hour - time.Nanosecond},
			wantErr:   "metrics query time range of 6h0m0s reads too much data, the allowed range for tenant foo is 3h59m59s",
		},
		{
			name:      "requested rounds up and allowed rounds down in the error",
			metas:     hourly,
			limits:    rangeLimits{maxScannedBytes: 450},
			requested: 4*time.Hour + 500*time.Millisecond,
			want:      scannedBytesCheck{budgetRange: 4*time.Hour - time.Nanosecond, estimatedBytes: 500, allowedRange: 4*time.Hour - time.Nanosecond},
			wantErr:   "metrics query time range of 4h0m1s reads too much data, the allowed range for tenant foo is 3h59m59s",
		},
		{
			name:      "over min_duration",
			metas:     hourly,
			limits:    rangeLimits{maxDuration: 24 * time.Hour, minDuration: 6 * time.Hour, maxScannedBytes: 150},
			requested: 7 * time.Hour,
			want:      scannedBytesCheck{budgetRange: time.Hour - time.Nanosecond, estimatedBytes: 800, allowedRange: 6 * time.Hour},
			wantErr:   "metrics query time range of 7h0m0s reads too much data, the allowed range for tenant foo is 6h0m0s",
		},
		{
			name:      "max_duration caps min_duration",
			metas:     hourly,
			limits:    rangeLimits{maxDuration: 5 * time.Hour, minDuration: 8 * time.Hour, maxScannedBytes: 150},
			requested: 6 * time.Hour,
			want:      scannedBytesCheck{budgetRange: time.Hour - time.Nanosecond, estimatedBytes: 700, allowedRange: 5 * time.Hour},
			wantErr:   "metrics query time range of 6h0m0s reads too much data, the allowed range for tenant foo is 5h0m0s",
		},
		{
			name:      "end before start fits",
			metas:     hourly,
			limits:    rangeLimits{maxScannedBytes: 1},
			start:     scannedBytesTestEnd.Add(time.Hour),
			requested: time.Hour,
			want:      scannedBytesCheck{withinBudget: true},
		},
		{
			name:         "a block starting at backendAfter counts",
			metas:        []*backend.BlockMeta{testBlock(1, 0, 300)},
			limits:       rangeLimits{maxScannedBytes: 250},
			backendAfter: hoursAgo(1),
			requested:    2 * time.Hour,
			want:         scannedBytesCheck{budgetRange: time.Hour, estimatedBytes: 300, allowedRange: time.Hour},
			wantErr:      "metrics query time range of 2h0m0s reads too much data, the allowed range for tenant foo is 1h0m0s",
		},
		{
			name:      "a prefix of blocks equal to the budget fits",
			metas:     []*backend.BlockMeta{testBlock(1, 0, 100), testBlock(2, 1, 150)},
			limits:    rangeLimits{maxScannedBytes: 100},
			requested: 2 * time.Hour,
			want:      scannedBytesCheck{budgetRange: time.Hour - time.Nanosecond, estimatedBytes: 250, allowedRange: time.Hour - time.Nanosecond},
			wantErr:   "metrics query time range of 2h0m0s reads too much data, the allowed range for tenant foo is 59m59s",
		},
		{
			name:      "one byte over the budget",
			metas:     []*backend.BlockMeta{testBlock(1, 0, 100), testBlock(2, 1, 151)},
			limits:    rangeLimits{maxScannedBytes: 250},
			requested: 2 * time.Hour,
			want:      scannedBytesCheck{budgetRange: time.Hour - time.Nanosecond, estimatedBytes: 251, allowedRange: time.Hour - time.Nanosecond},
			wantErr:   "metrics query time range of 2h0m0s reads too much data, the allowed range for tenant foo is 59m59s",
		},
		{
			name:      "one block larger than the budget",
			metas:     []*backend.BlockMeta{testBlock(1, 0, 1000)},
			limits:    rangeLimits{maxDuration: 24 * time.Hour, maxScannedBytes: 500},
			requested: time.Hour,
			want:      scannedBytesCheck{estimatedBytes: 1000},
			wantErr:   "metrics query time range of 1h0m0s reads too much data, the allowed range for tenant foo is 0s",
		},
		{
			name:         "range after backendAfter is free",
			metas:        []*backend.BlockMeta{testBlock(3, 0, 300)},
			limits:       rangeLimits{maxScannedBytes: 250},
			backendAfter: hoursAgo(1),
			requested:    3 * time.Hour,
			want:         scannedBytesCheck{budgetRange: time.Hour, estimatedBytes: 300, allowedRange: time.Hour},
			wantErr:      "metrics query time range of 3h0m0s reads too much data, the allowed range for tenant foo is 1h0m0s",
		},
		{
			// Search start and backendAfter are both whole seconds, so this edge is reachable.
			name:         "range starting at backendAfter fits",
			metas:        []*backend.BlockMeta{testBlock(3, 0, 300)},
			limits:       rangeLimits{maxScannedBytes: 250},
			backendAfter: hoursAgo(1),
			requested:    time.Hour,
			want:         scannedBytesCheck{withinBudget: true},
		},
		{
			name:         "blocks starting after backendAfter are free",
			metas:        []*backend.BlockMeta{testBlock(3, 2, 100), {StartTime: scannedBytesTestEnd.Add(-30 * time.Minute), EndTime: scannedBytesTestEnd.Add(-10 * time.Minute), Size_: 1000, ReplicationFactor: backend.MetricsGeneratorReplicationFactor}},
			limits:       rangeLimits{maxScannedBytes: 250},
			backendAfter: hoursAgo(1),
			requested:    3 * time.Hour,
			want:         scannedBytesCheck{withinBudget: true, estimatedBytes: 100},
		},
		{
			name:         "range wholly after backendAfter fits",
			metas:        []*backend.BlockMeta{testBlock(3, 0, 300)},
			limits:       rangeLimits{maxScannedBytes: 250},
			backendAfter: hoursAgo(2),
			requested:    time.Hour,
			want:         scannedBytesCheck{withinBudget: true},
		},
		{
			// A long compacted block starts before the fitting blocks but ends inside the range. Input order is shuffled.
			name:      "blocks enter by end time, newest first",
			metas:     []*backend.BlockMeta{testBlock(3, 2, 100), testBlock(1, 0, 100), testBlock(10, 1, 100)},
			limits:    rangeLimits{maxScannedBytes: 250},
			requested: 10 * time.Hour,
			want:      scannedBytesCheck{budgetRange: 2*time.Hour - time.Nanosecond, estimatedBytes: 300, allowedRange: 2*time.Hour - time.Nanosecond},
			wantErr:   "metrics query time range of 10h0m0s reads too much data, the allowed range for tenant foo is 1h59m59s",
		},
		{
			name:      "blocks with the same end time enter together",
			metas:     []*backend.BlockMeta{testBlock(1, 0, 100), testBlock(2, 1, 100), testBlock(2, 1, 100)},
			limits:    rangeLimits{maxScannedBytes: 250},
			requested: 2 * time.Hour,
			want:      scannedBytesCheck{budgetRange: time.Hour - time.Nanosecond, estimatedBytes: 300, allowedRange: time.Hour - time.Nanosecond},
			wantErr:   "metrics query time range of 2h0m0s reads too much data, the allowed range for tenant foo is 59m59s",
		},
		{
			name:         "blocks after backendAfter stay out of the budget range",
			metas:        []*backend.BlockMeta{{StartTime: scannedBytesTestEnd.Add(-30 * time.Minute), EndTime: scannedBytesTestEnd.Add(-10 * time.Minute), Size_: 1000, ReplicationFactor: backend.MetricsGeneratorReplicationFactor}, testBlock(3, 2, 100), testBlock(5, 4, 200)},
			limits:       rangeLimits{maxScannedBytes: 250},
			backendAfter: hoursAgo(1),
			requested:    5 * time.Hour,
			want:         scannedBytesCheck{budgetRange: 4*time.Hour - time.Nanosecond, estimatedBytes: 300, allowedRange: 4*time.Hour - time.Nanosecond},
			wantErr:      "metrics query time range of 5h0m0s reads too much data, the allowed range for tenant foo is 3h59m59s",
		},
		{
			// A metrics backendAfter is in nanoseconds, so it can land under 1s before the end.
			name:         "allowed range under 1s",
			metas:        []*backend.BlockMeta{testBlock(1, 0, 300)},
			limits:       rangeLimits{maxScannedBytes: 250},
			backendAfter: scannedBytesTestEnd.Add(-500 * time.Millisecond),
			requested:    time.Hour,
			want:         scannedBytesCheck{budgetRange: 500 * time.Millisecond, estimatedBytes: 300, allowedRange: 500 * time.Millisecond},
			wantErr:      "metrics query time range of 1h0m0s reads too much data, the allowed range for tenant foo is 500ms",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			start := tc.start
			if start.IsZero() {
				start = scannedBytesTestEnd.Add(-tc.requested)
			}
			backendAfter := tc.backendAfter
			if backendAfter.IsZero() {
				backendAfter = scannedBytesTestEnd
			}

			s := scannedBytesLimit{op: metricsOp, reader: &mockReader{metas: tc.metas}, filterFn: isRF1Block}
			got := s.estimateBackendBytes("foo", tc.limits, start, scannedBytesTestEnd, backendAfter)

			require.Equal(t, tc.want, got)
			if tc.wantErr != "" {
				require.EqualError(t, scannedBytesErr(metricsOp, "foo", scannedBytesTestEnd.Sub(start), got.allowedRange), tc.wantErr)
			}
		})
	}
}

func TestScannedBytesLimitOrder(t *testing.T) {
	tcs := []struct {
		name      string
		limits    rangeLimits
		requested time.Duration
		// wantCheck is the check's result, or empty when the blocklist isn't read.
		wantCheck string
		// wantRange is the query range histogram result.
		wantRange string
		// wantBudgetRange is how many budget ranges the histogram records.
		wantBudgetRange uint64
	}{
		{
			name:      "over max_duration without max_scanned_bytes reads nothing",
			wantRange: queryRangeResultRejectedMaxDuration,
			limits:    rangeLimits{maxDuration: 5 * time.Hour},
			requested: 6 * time.Hour,
		},
		{
			name:      "over max_duration is still logged",
			wantRange: queryRangeResultRejectedMaxDuration,
			limits:    rangeLimits{maxDuration: 5 * time.Hour, maxScannedBytes: 150},
			requested: 6 * time.Hour,
			wantCheck: queryRangeResultRejectedMaxDuration,
		},
		{
			name:      "max_duration wins over min_duration",
			wantRange: queryRangeResultRejectedMaxDuration,
			limits:    rangeLimits{maxDuration: 5 * time.Hour, minDuration: 8 * time.Hour, maxScannedBytes: 150},
			requested: 6 * time.Hour,
			wantCheck: queryRangeResultRejectedMaxDuration,
		},
		{
			name:      "within min_duration reads nothing",
			wantRange: queryRangeResultAllowed,
			limits:    rangeLimits{maxDuration: 24 * time.Hour, minDuration: 6 * time.Hour, maxScannedBytes: 150},
			requested: 6 * time.Hour,
		},
		{
			name:      "max_scanned_bytes off reads nothing",
			wantRange: queryRangeResultAllowed,
			limits:    rangeLimits{maxDuration: 24 * time.Hour},
			requested: 6 * time.Hour,
		},
		{
			name:      "fits",
			wantRange: queryRangeResultAllowed,
			limits:    rangeLimits{maxScannedBytes: 1000},
			requested: 10 * time.Hour,
			wantCheck: queryRangeResultAllowed,
		},
		{
			name:            "over the budget",
			wantRange:       queryRangeResultRejectedScannedBytes,
			wantBudgetRange: 1,
			limits:          rangeLimits{maxScannedBytes: 150},
			requested:       6 * time.Hour,
			wantCheck:       queryRangeResultRejectedScannedBytes,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			rdr := &countingMockReader{mockReader: mockReader{metas: hourlyMetricsBlocks(scannedBytesTestEnd)}}
			s := scannedBytesLimit{
				op:       metricsOp,
				reader:   rdr,
				filterFn: isRF1Block,
			}

			rangeBefore := queryRangeCount(t, metricsOp, tc.wantRange)
			countBefore, derivedBefore := resultCount("foo", metricsOp, tc.wantRange), histogramCount(t, "foo", metricsOp)
			result, est := s.checkTenantRange("foo", tc.limits, scannedBytesTestEnd.Add(-tc.requested), scannedBytesTestEnd, scannedBytesTestEnd)
			recordRangeCheck(metricsRangeLimitMetrics, "foo", tc.requested, result, est)
			require.Equal(t, tc.wantRange, result)
			require.Equal(t, rangeBefore+1, queryRangeCount(t, metricsOp, tc.wantRange))
			require.Equal(t, tc.wantBudgetRange, histogramCount(t, "foo", metricsOp)-derivedBefore)
			if tc.wantCheck == "" {
				require.Nil(t, est)
				require.Zero(t, rdr.callCount())
			} else {
				require.NotNil(t, est)
				require.Equal(t, tc.wantCheck, result)
				require.Equal(t, 1, rdr.callCount())
				require.Equal(t, countBefore+1, resultCount("foo", metricsOp, tc.wantRange))
			}
		})
	}
}

var (
	checkRangeSpansOnce sync.Once
	checkRangeSpans     *tracetest.SpanRecorder
)

// recordCheckRangeSpans returns a traced context, and a func that ends it and maps tenant to result for its frontend.CheckRange spans.
// The global tracer delegate is set only once, so every test shares one recorder.
func recordCheckRangeSpans() (context.Context, func() map[string]string) {
	checkRangeSpansOnce.Do(func() {
		checkRangeSpans = tracetest.NewSpanRecorder()
		otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(checkRangeSpans)))
	})
	ctx, parent := tracer.Start(context.Background(), "test")
	return ctx, func() map[string]string {
		parent.End()
		got := map[string]string{}
		for _, s := range checkRangeSpans.Ended() {
			if s.Name() != "frontend.CheckRange" || s.SpanContext().TraceID() != parent.SpanContext().TraceID() {
				continue
			}
			var tenantID, result string
			for _, a := range s.Attributes() {
				switch a.Key {
				case "tenant":
					tenantID = a.Value.AsString()
				case "result":
					result = a.Value.AsString()
				}
			}
			got[tenantID] = result
		}
		return got
	}
}

func queryRangeCount(t *testing.T, op, result string) uint64 {
	m := &dto.Metric{}
	require.NoError(t, requestQueryRange.WithLabelValues(op, result).(prometheus.Histogram).Write(m))
	return m.GetHistogram().GetSampleCount()
}

// Both callers record every query, with max_scanned_bytes off.
func TestRangeLimitersRecordQueryRange(t *testing.T) {
	o := newTestOverrides(t, overrides.ReadOverrides{})

	t.Run(metricsOp, func(t *testing.T) {
		cfg := Config{}
		cfg.Metrics.Sharder.MaxDuration = time.Hour
		l := newMetricsRangeLimiter(cfg, o, &mockReader{})
		allowed, overMax := queryRangeCount(t, metricsOp, queryRangeResultAllowed), queryRangeCount(t, metricsOp, queryRangeResultRejectedMaxDuration)
		require.NoError(t, checkMetricsRange(context.Background(), cfg, l, log.NewNopLogger(), "foo", hoursAgo(1), scannedBytesTestEnd))
		require.Error(t, checkMetricsRange(context.Background(), cfg, l, log.NewNopLogger(), "foo", hoursAgo(2), scannedBytesTestEnd))
		require.Equal(t, allowed+1, queryRangeCount(t, metricsOp, queryRangeResultAllowed))
		require.Equal(t, overMax+1, queryRangeCount(t, metricsOp, queryRangeResultRejectedMaxDuration))
	})

	t.Run(searchOp, func(t *testing.T) {
		sharder := newAsyncSearchSharder(&mockReader{}, o, SearchSharderConfig{
			ConcurrentRequests:    defaultConcurrentRequests,
			TargetBytesPerRequest: defaultTargetBytesPerRequest,
			MostRecentShards:      defaultMostRecentShards,
			MaxDuration:           time.Hour,
		}, nil, newJobsPerQueryHistogram(), log.NewNopLogger())
		roundTrip := func(query string) {
			req := httptest.NewRequest("GET", query, nil)
			// Unblocks the sharder's sender goroutine, since the allowed search's response is never read.
			ctx, cancel := context.WithCancel(user.InjectOrgID(req.Context(), "foo"))
			t.Cleanup(cancel)
			_, err := sharder.Wrap(nopNext).RoundTrip(pipeline.NewHTTPRequest(req.WithContext(ctx)))
			require.NoError(t, err)
		}
		allowed, overMax := queryRangeCount(t, searchOp, queryRangeResultAllowed), queryRangeCount(t, searchOp, queryRangeResultRejectedMaxDuration)
		roundTrip("/?start=1&end=3600")
		roundTrip("/?start=1&end=8200")
		require.Equal(t, allowed+1, queryRangeCount(t, searchOp, queryRangeResultAllowed))
		require.Equal(t, overMax+1, queryRangeCount(t, searchOp, queryRangeResultRejectedMaxDuration))
	})
}

// blocklistReader reads a real blocklist, so the benchmark pays the lock and copy that BlockMetas does per query.
type blocklistReader struct {
	mockReader
	list *blocklist.List
}

func (r *blocklistReader) BlockMetas(tenantID string) []*backend.BlockMeta {
	return r.list.Metas(tenantID)
}

func BenchmarkEstimateBackendBytes(b *testing.B) {
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
					s.estimateBackendBytes("foo", l, scannedBytesTestEnd.Add(-7*24*time.Hour), scannedBytesTestEnd, scannedBytesTestEnd)
				}
			})
		}
	}
}
