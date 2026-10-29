package frontend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/go-kit/log"
	"github.com/gorilla/mux"
	"github.com/grafana/dskit/user"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestQueryTrackerDroppedObservability(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var logs bytes.Buffer
		reg := prometheus.NewPedanticRegistry()
		tracker := newQueryTracker(QueryTrackerConfig{
			Enabled: true, MaxQueriesPerTenant: 1,
		}, log.NewLogfmtLogger(&logs), reg).(*queryTracker)
		ctxA := user.InjectOrgID(context.Background(), "tenant-a")
		ctxB := user.InjectOrgID(context.Background(), "tenant-b")
		stopA := tracker.track(ctxA, http.MethodGet, "/api/search", "{}")
		defer stopA()
		stopB := tracker.track(ctxB, http.MethodGet, "/api/search", "{}")
		defer stopB()
		families, err := reg.Gather()
		require.NoError(t, err)
		require.Empty(t, families)

		for range 3 {
			tracker.track(ctxA, http.MethodGet, "/api/search", "not logged")()
		}
		for range 2 {
			tracker.track(ctxB, http.MethodGet, "/api/search", "not logged")()
		}
		require.NoError(t, testutil.GatherAndCompare(reg, strings.NewReader(`
# HELP tempo_query_frontend_query_tracker_dropped_total Total number of query tracking entries dropped because the per-tenant limit was reached. The queries still execute.
# TYPE tempo_query_frontend_query_tracker_dropped_total counter
tempo_query_frontend_query_tracker_dropped_total{tenant="tenant-a"} 3
tempo_query_frontend_query_tracker_dropped_total{tenant="tenant-b"} 2
`), "tempo_query_frontend_query_tracker_dropped_total"))
		require.Equal(t, "level=warn msg=\"query tracker capacity reached; query continues without tracking\" tenant=tenant-a max_queries_per_tenant=1\n", logs.String())
		require.Len(t, tracker.trackedQueries("tenant-a"), 1)
		require.Len(t, tracker.trackedQueries("tenant-b"), 1)

		time.Sleep(time.Second)
		tracker.track(ctxB, http.MethodGet, "/api/search", "{}")()
		require.Equal(t, float64(3), testutil.ToFloat64(tracker.droppedQueries.WithLabelValues("tenant-b")))
		require.Equal(t, 2, strings.Count(logs.String(), "\n"))
		require.Contains(t, logs.String(), "tenant=tenant-b max_queries_per_tenant=1\n")

		stopA()
		tracker.track(ctxA, http.MethodGet, "/api/search", "{}")()
		require.Equal(t, float64(3), testutil.ToFloat64(tracker.droppedQueries.WithLabelValues("tenant-a")))
	})
}

func TestDisabledQueryTrackerDoesNotRegisterDropMetrics(t *testing.T) {
	var logs bytes.Buffer
	reg := prometheus.NewPedanticRegistry()
	tracker := newQueryTracker(QueryTrackerConfig{}, log.NewLogfmtLogger(&logs), reg)
	tracker.track(user.InjectOrgID(context.Background(), "tenant-a"), http.MethodGet, "/api/search", "{}")()
	families, err := reg.Gather()
	require.NoError(t, err)
	require.Empty(t, families)
	require.Empty(t, logs.String())
}

func TestQueryTrackerZeroMaxQueryTextBytesKeepsFullText(t *testing.T) {
	tracker := newQueryTracker(QueryTrackerConfig{
		Enabled: true, MaxQueryTextBytes: 0,
	}, log.NewNopLogger(), nil).(*queryTracker)
	query := "{ .message = \"" + strings.Repeat("é", 4096) + "\" }"
	ctx := user.InjectOrgID(context.Background(), "tenant-a")
	stop := tracker.track(ctx, http.MethodGet, "/api/search", query)
	defer stop()

	queries := tracker.trackedQueries("tenant-a")
	require.Len(t, queries, 1)
	require.Equal(t, query, queries[0].Query)
}

type recordingQueryTracker struct {
	pushes, pops                int
	method, path, query, tenant string
}

type testRoundTripper func(*http.Request) (*http.Response, error)

func (f testRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func (t *recordingQueryTracker) track(ctx context.Context, method, path, query string) func() {
	t.pushes++
	t.method = method
	t.path = path
	t.query = query
	t.tenant, _ = user.ExtractOrgID(ctx)
	return func() { t.pops++ }
}

func TestQueryTracker(t *testing.T) {
	tracker := newQueryTracker(QueryTrackerConfig{Enabled: true, MaxQueriesPerTenant: 1, MaxQueryTextBytes: 8}, log.NewNopLogger(), nil).(*queryTracker)
	stop := tracker.track(user.InjectOrgID(context.Background(), "tenant-a"), http.MethodGet, "/api/search", "{ .foo = \"é\" }")
	defer stop()
	tracker.track(user.InjectOrgID(context.Background(), "tenant-a"), "GRPC", "/tempopb.StreamingQuerier/Search", "{ .bar }")()

	queries := tracker.trackedQueries("tenant-a")
	require.Len(t, queries, 1)
	require.Equal(t, "{ .foo =", queries[0].Query)
	require.True(t, time.Now().After(queries[0].StartedAt))
	require.Equal(t, uint64(1), tracker.trackedTenants()[0].Dropped)

	t.Run("tenants page renders HTML", func(t *testing.T) {
		rec := httptest.NewRecorder()
		tracker.tenantsHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/status/active-queries", nil))
		require.Equal(t, http.StatusOK, rec.Code)
		require.Contains(t, rec.Header().Get("Content-Type"), "text/html")
		require.Contains(t, rec.Body.String(), "tenant-a")
	})

	t.Run("tenants page renders JSON", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/status/active-queries", nil)
		req.Header.Set("Accept", "application/json")
		tracker.tenantsHandler().ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		var page queryTrackerPage
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&page))
		require.Len(t, page.Tenants, 1)
	})

	t.Run("tenant page renders JSON", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/status/active-queries/tenant-a", nil)
		req.Header.Set("Accept", "application/json")
		req = mux.SetURLVars(req, map[string]string{"tenant": "tenant-a"})
		tracker.tenantHandler().ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		var page queryTrackerPage
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&page))
		require.Equal(t, "tenant-a", page.Tenant)
		require.Len(t, page.Queries, 1)
	})

	t.Run("tenant page renders HTML", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/status/active-queries/tenant-a", nil), map[string]string{"tenant": "tenant-a"})
		tracker.tenantHandler().ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Contains(t, rec.Body.String(), "{ .foo =")
	})

	t.Run("disabled tracker is a no-op", func(t *testing.T) {
		tracker := newQueryTracker(QueryTrackerConfig{}, log.NewNopLogger(), nil)
		tracker.track(user.InjectOrgID(context.Background(), "tenant-a"), http.MethodGet, "/api/search", "query")()
		_, ok := tracker.(noOpQueryTracker)
		require.True(t, ok)
	})

	require.Equal(t, "ab", truncateQuery("abé", 3))
	require.False(t, strings.Contains(truncateQuery("abc", 2), "\x00"))
}

func TestHandlerTracksDecodedQueryText(t *testing.T) {
	for _, tc := range []struct {
		name     string
		request  string
		path     string
		expected string
	}{
		{
			name:     "search q parameter",
			request:  "/api/search?q=%7B+resource.service.name+%3D+%22api%22+%7D",
			path:     "/api/search",
			expected: `{ resource.service.name = "api" }`,
		},
		{
			name:     "metrics query parameter",
			request:  "/api/metrics/query_range?query=%7B+.status+%3D+%22error%22+%7D",
			path:     "/api/metrics/query_range",
			expected: `{ .status = "error" }`,
		},
		{
			name:     "q parameter takes precedence",
			request:  "/api/search?q=%7B+.service.name+%3D+%22api%22+%7D&query=%7B+.service.name+%3D+%22worker%22+%7D",
			path:     "/api/search",
			expected: `{ .service.name = "api" }`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tracker := &recordingQueryTracker{}
			handler := newHandlerWithTracker(nil, testRoundTripper(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusInternalServerError, Header: make(http.Header), Body: http.NoBody}, nil
			}), log.NewNopLogger(), tracker)

			req := httptest.NewRequest(http.MethodGet, tc.request, nil)
			req = req.WithContext(user.InjectOrgID(req.Context(), "tenant-a"))
			handler.ServeHTTP(httptest.NewRecorder(), req)

			require.Equal(t, http.MethodGet, tracker.method)
			require.Equal(t, tc.path, tracker.path)
			require.Equal(t, tc.expected, tracker.query)
		})
	}
}

func TestQueryTrackerConcurrentTenants(t *testing.T) {
	tracker := newQueryTracker(QueryTrackerConfig{Enabled: true}, log.NewNopLogger(), nil).(*queryTracker)
	var wg sync.WaitGroup

	for tenant := 0; tenant < 8; tenant++ {
		wg.Add(1)
		go func(tenant string) {
			defer wg.Done()
			ctx := user.InjectOrgID(context.Background(), tenant)
			for query := 0; query < 100; query++ {
				tracker.track(ctx, "GRPC", "/tempopb.StreamingQuerier/Search", "{}")()
			}
		}(fmt.Sprintf("tenant-%d", tenant))
	}

	wg.Wait()
	require.Empty(t, tracker.trackedTenants())
}

func TestQueryTrackerTracksMultiTenantQueryForEachTenant(t *testing.T) {
	tracker := newQueryTracker(QueryTrackerConfig{Enabled: true}, log.NewNopLogger(), nil).(*queryTracker)
	stop := tracker.track(user.InjectOrgID(context.Background(), "tenant-a|tenant-b"), "GRPC", "/tempopb.StreamingQuerier/Search", "{}")
	defer stop()

	for _, tenant := range []string{"tenant-a", "tenant-b"} {
		queries := tracker.trackedQueries(tenant)
		require.Len(t, queries, 1)
		require.Equal(t, tenant, queries[0].Tenant)
	}

	tenantStatus := tracker.trackedTenants()
	require.Len(t, tenantStatus, 2)
	require.Equal(t, "tenant-a", tenantStatus[0].Name)
	require.Equal(t, "tenant-b", tenantStatus[1].Name)
}

func TestQueryTrackerTracksMultiTenantQueryWhenOneTenantIsAtCap(t *testing.T) {
	tracker := newQueryTracker(QueryTrackerConfig{
		Enabled: true, MaxQueriesPerTenant: 1,
	}, log.NewNopLogger(), nil).(*queryTracker)

	stopA := tracker.track(user.InjectOrgID(context.Background(), "a"), http.MethodGet, "/api/search", "{ .existing = true }")
	defer stopA()
	existingQueries := tracker.trackedQueries("a")
	require.Len(t, existingQueries, 1)

	const query = "{ .status = \"error\" }"
	stopMultiTenant := tracker.track(user.InjectOrgID(context.Background(), "a|b"), "GRPC", "/tempopb.StreamingQuerier/Search", query)
	defer stopMultiTenant()

	require.Equal(t, existingQueries, tracker.trackedQueries("a"))
	queriesB := tracker.trackedQueries("b")
	require.Len(t, queriesB, 1)
	require.Equal(t, "b", queriesB[0].Tenant)
	require.Equal(t, query, queriesB[0].Query)
	require.Equal(t, []queryTrackerTenant{
		{Name: "a", Active: 1, Dropped: 1},
		{Name: "b", Active: 1, Dropped: 0},
	}, tracker.trackedTenants())
	require.Equal(t, float64(1), testutil.ToFloat64(tracker.droppedQueries.WithLabelValues("a")))
	require.Equal(t, float64(0), testutil.ToFloat64(tracker.droppedQueries.WithLabelValues("b")))

	stopMultiTenant()
	require.Empty(t, tracker.trackedQueries("b"))
	require.Equal(t, existingQueries, tracker.trackedQueries("a"))
	require.Equal(t, []queryTrackerTenant{
		{Name: "a", Active: 1, Dropped: 1},
	}, tracker.trackedTenants())

	stopA()
	require.Empty(t, tracker.trackedQueries("a"))
	require.Equal(t, []queryTrackerTenant{
		{Name: "a", Active: 0, Dropped: 1},
	}, tracker.trackedTenants())
}
