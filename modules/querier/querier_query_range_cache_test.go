package querier

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/grafana/dskit/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/cache"
	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/vparquet5"
)

func TestMetricsRowGroupCacheKey_Format(t *testing.T) {
	blockID := backend.MustParse("00000000-0000-0000-0000-000000000123")
	key := metricsRowGroupCacheKey("tenant-a", blockID, 4, 42)
	require.Equal(t, "qrgv1:tenant-a:00000000-0000-0000-0000-000000000123:4:42", key)
}

func TestMetricsRowGroupCacheKey_EmptyOnZeroHash(t *testing.T) {
	blockID := backend.MustParse("00000000-0000-0000-0000-000000000123")
	require.Equal(t, "", metricsRowGroupCacheKey("tenant-a", blockID, 4, 0))
}

func TestMetricsRowGroupCacheKey_Uniqueness(t *testing.T) {
	blockA := backend.MustParse("00000000-0000-0000-0000-000000000001")
	blockB := backend.MustParse("00000000-0000-0000-0000-000000000002")

	base := metricsRowGroupCacheKey("tenant-a", blockA, 1, 100)
	require.NotEmpty(t, base)
	require.Equal(t, base, metricsRowGroupCacheKey("tenant-a", blockA, 1, 100))

	require.NotEqual(t, base, metricsRowGroupCacheKey("tenant-b", blockA, 1, 100))
	require.NotEqual(t, base, metricsRowGroupCacheKey("tenant-a", blockB, 1, 100))
	require.NotEqual(t, base, metricsRowGroupCacheKey("tenant-a", blockA, 2, 100))
	require.NotEqual(t, base, metricsRowGroupCacheKey("tenant-a", blockA, 1, 101))
}

func TestMetricsRowGroupCacheKey_TenantIsolation(t *testing.T) {
	blockID := backend.NewUUID()
	require.NotEqual(t,
		metricsRowGroupCacheKey("a", blockID, 0, 1),
		metricsRowGroupCacheKey("b", blockID, 0, 1),
	)
}

func TestMetricsRowGroupCacheKey_StableForSameUUID(t *testing.T) {
	id := uuid.MustParse("00000000-0000-0000-0000-deadbeef0001")
	blockID := backend.UUID(id)
	a := metricsRowGroupCacheKey("t", blockID, 7, 9)
	b := metricsRowGroupCacheKey("t", blockID, 7, 9)
	require.Equal(t, a, b)
}

func TestRowGroupQueryRangeHash_Deterministic(t *testing.T) {
	req := &tempopb.QueryRangeRequest{
		Query:     "{ } | rate()",
		Step:      15_000_000_000,
		MaxSeries: 100,
		Exemplars: 50,
	}
	h1 := rowGroupQueryRangeHash(req, nil, 0.2, false)
	h2 := rowGroupQueryRangeHash(req, nil, 0.2, false)
	require.Equal(t, h1, h2)
	require.NotZero(t, h1)
}

func TestRowGroupQueryRangeHash_ZeroOnParseFail(t *testing.T) {
	require.Zero(t, rowGroupQueryRangeHash(&tempopb.QueryRangeRequest{Query: "not a query"}, nil, 0.2, false))
	require.Zero(t, rowGroupQueryRangeHash(&tempopb.QueryRangeRequest{Query: ""}, nil, 0.2, false))
}

func TestRowGroupQueryRangeHash_FieldsAffectHash(t *testing.T) {
	baseReq := &tempopb.QueryRangeRequest{
		Query:                  "{ } | rate()",
		Step:                   15_000_000_000,
		MaxSeries:              100,
		Exemplars:              50,
		SkipASTTransformations: []string{"x"},
	}
	base := rowGroupQueryRangeHash(baseReq, nil, 0.2, false)

	r := *baseReq
	r.Step = 30_000_000_000
	require.NotEqual(t, base, rowGroupQueryRangeHash(&r, nil, 0.2, false))

	r = *baseReq
	r.Exemplars = 60
	require.NotEqual(t, base, rowGroupQueryRangeHash(&r, nil, 0.2, false))

	r = *baseReq
	r.SkipASTTransformations = []string{"y"}
	require.NotEqual(t, base, rowGroupQueryRangeHash(&r, nil, 0.2, false))

	require.NotEqual(t, base, rowGroupQueryRangeHash(baseReq, nil, 0.3, false))
	require.NotEqual(t, base, rowGroupQueryRangeHash(baseReq, nil, 0.2, true))

	dc2 := backend.DedicatedColumns{
		{Scope: backend.DedicatedColumnScopeSpan, Name: "http.url", Type: backend.DedicatedColumnTypeString},
	}
	require.NotEqual(t, base, rowGroupQueryRangeHash(baseReq, dc2, 0.2, false))
}

func TestRowGroupQueryRangeHash_MaxSeriesIgnored(t *testing.T) {
	// MaxSeries deliberately excluded from the hash — per-row-group eval is
	// uncapped and truncation runs only on the merged result. Two requests
	// differing only in MaxSeries must share cache entries.
	r1 := &tempopb.QueryRangeRequest{Query: "{ } | rate()", Step: 15_000_000_000, MaxSeries: 100}
	r2 := *r1
	r2.MaxSeries = 5
	require.Equal(t,
		rowGroupQueryRangeHash(r1, nil, 0.2, false),
		rowGroupQueryRangeHash(&r2, nil, 0.2, false),
	)
}

func TestRowGroupCacheValueRoundTrip(t *testing.T) {
	orig := &tempopb.QueryRangeResponse{
		Series: []*tempopb.TimeSeries{
			{
				Labels: []commonv1.KeyValue{
					{Key: "__name__", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "rate"}}},
				},
				Samples: []tempopb.Sample{
					{TimestampMs: 1000, Value: 1.5},
					{TimestampMs: 2000, Value: 2.5},
				},
			},
		},
		Metrics: &tempopb.SearchMetrics{
			InspectedBytes: 12345,
			InspectedSpans: 678,
		},
	}
	buf, err := marshalRowGroupCacheValue(orig)
	require.NoError(t, err)
	require.NotEmpty(t, buf)

	got, err := unmarshalRowGroupCacheValue(buf)
	require.NoError(t, err)
	require.Equal(t, orig.Metrics.InspectedBytes, got.Metrics.InspectedBytes)
	require.Equal(t, orig.Metrics.InspectedSpans, got.Metrics.InspectedSpans)
	require.Len(t, got.Series, 1)
	require.Equal(t, orig.Series[0].Samples, got.Series[0].Samples)
	require.Equal(t, orig.Series[0].Labels[0].Key, got.Series[0].Labels[0].Key)
}

func TestEvaluateRowGroupCacheEligibility_NoProvider(t *testing.T) {
	req := &tempopb.QueryRangeRequest{Version: vparquet5.VersionString}
	elig := evaluateRowGroupCacheEligibility(req, nil, nil, false, 1)
	require.False(t, elig.use)
	require.Equal(t, cacheSkipReasonNoProvider, elig.reason)
}

func TestEvaluateRowGroupCacheEligibility_WrongVersion(t *testing.T) {
	req := &tempopb.QueryRangeRequest{Version: "vParquet4"}
	provider := newFakeProvider(newInMemCache())
	elig := evaluateRowGroupCacheEligibility(req, nil, provider, false, 1)
	require.False(t, elig.use)
	require.Equal(t, cacheSkipReasonWrongVersion, elig.reason)
}

func TestEvaluateRowGroupCacheEligibility_ZeroHash(t *testing.T) {
	req := &tempopb.QueryRangeRequest{Version: vparquet5.VersionString}
	provider := newFakeProvider(newInMemCache())
	elig := evaluateRowGroupCacheEligibility(req, nil, provider, false, 0)
	require.False(t, elig.use)
	require.Equal(t, cacheSkipReasonZeroQueryHash, elig.reason)
}

func TestEvaluateRowGroupCacheEligibility_SamplerHint(t *testing.T) {
	req := &tempopb.QueryRangeRequest{Version: vparquet5.VersionString, Query: "{ } | rate() with(trace_sample=0.5)"}
	expr, err := traceql.ParseNoOptimizations(req.Query)
	require.NoError(t, err)
	provider := newFakeProvider(newInMemCache())
	// allowUnsafeHints=true so the trace_sample hint is honored.
	elig := evaluateRowGroupCacheEligibility(req, expr, provider, true, 1)
	require.False(t, elig.use)
	require.Equal(t, cacheSkipReasonSamplerHint, elig.reason)
}

func TestEvaluateRowGroupCacheEligibility_NoCacheForRole(t *testing.T) {
	req := &tempopb.QueryRangeRequest{Version: vparquet5.VersionString}
	provider := &fakeProvider{caches: map[cache.Role]cache.Cache{}}
	elig := evaluateRowGroupCacheEligibility(req, nil, provider, false, 1)
	require.False(t, elig.use)
	require.Equal(t, cacheSkipReasonNoCache, elig.reason)
}

func TestEvaluateRowGroupCacheEligibility_Use(t *testing.T) {
	req := &tempopb.QueryRangeRequest{Version: vparquet5.VersionString, Query: "{ } | rate()"}
	expr, err := traceql.ParseNoOptimizations(req.Query)
	require.NoError(t, err)
	want := newInMemCache()
	provider := newFakeProvider(want)
	elig := evaluateRowGroupCacheEligibility(req, expr, provider, true, 100)
	require.True(t, elig.use)
	require.Equal(t, want, elig.cache)
}

func TestRowGroupCacheStore_RespectsMaxItemSize(t *testing.T) {
	c := newInMemCache()
	c.maxItem = 4
	rowGroupCacheStore(context.Background(), c, "key", []byte("too-large-payload"))
	assert.Equal(t, 0, c.Len())

	rowGroupCacheStore(context.Background(), c, "key", []byte("ok"))
	assert.Equal(t, 1, c.Len())
}

// --- test helpers ---

type inMemCache struct {
	mu      sync.Mutex
	store   map[string][]byte
	maxItem int
}

func newInMemCache() *inMemCache {
	return &inMemCache{store: map[string][]byte{}}
}

func (m *inMemCache) Store(_ context.Context, keys []string, bufs [][]byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, k := range keys {
		buf := make([]byte, len(bufs[i]))
		copy(buf, bufs[i])
		m.store[k] = buf
	}
}

func (m *inMemCache) Fetch(_ context.Context, keys []string) (found []string, bufs [][]byte, missing []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		if v, ok := m.store[k]; ok {
			found = append(found, k)
			bufs = append(bufs, v)
		} else {
			missing = append(missing, k)
		}
	}
	return
}

func (m *inMemCache) FetchKey(_ context.Context, key string) ([]byte, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.store[key]
	return v, ok
}

func (m *inMemCache) MaxItemSize() int { return m.maxItem }
func (m *inMemCache) Release([]byte)   {}
func (m *inMemCache) Stop()            {}

func (m *inMemCache) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.store)
}

type fakeProvider struct {
	caches map[cache.Role]cache.Cache
}

func newFakeProvider(c cache.Cache) *fakeProvider {
	return &fakeProvider{caches: map[cache.Role]cache.Cache{cache.RoleMetricsQueryRowGroup: c}}
}

func (f *fakeProvider) CacheFor(role cache.Role) cache.Cache { return f.caches[role] }
func (f *fakeProvider) AddCache(role cache.Role, c cache.Cache) error {
	if f.caches == nil {
		f.caches = map[cache.Role]cache.Cache{}
	}
	f.caches[role] = c
	return nil
}

func (f *fakeProvider) StartAsync(context.Context) error      { return nil }
func (f *fakeProvider) AwaitRunning(context.Context) error    { return nil }
func (f *fakeProvider) StopAsync()                            {}
func (f *fakeProvider) AwaitTerminated(context.Context) error { return nil }
func (f *fakeProvider) FailureCase() error                    { return nil }
func (f *fakeProvider) State() services.State                 { return services.New }
func (f *fakeProvider) AddListener(services.Listener) func()  { return func() {} }

var _ cache.Provider = (*fakeProvider)(nil)
