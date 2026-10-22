package frontend

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/gorilla/mux"
	"github.com/grafana/dskit/user"
	"github.com/stretchr/testify/require"
)

type recordingQueryTracker struct {
	method string
	path   string
	query  string
}

type testRoundTripper func(*http.Request) (*http.Response, error)

func (f testRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func (t *recordingQueryTracker) track(_ context.Context, method, path, query string) func() {
	t.method = method
	t.path = path
	t.query = query
	return func() {}
}

func TestQueryTracker(t *testing.T) {
	tracker := newQueryTracker(QueryTrackerConfig{Enabled: true, MaxQueriesPerTenant: 1, MaxQueryTextBytes: 8}).(*queryTracker)
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
		tracker := newQueryTracker(QueryTrackerConfig{})
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
	tracker := newQueryTracker(QueryTrackerConfig{Enabled: true}).(*queryTracker)
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
	tracker := newQueryTracker(QueryTrackerConfig{Enabled: true}).(*queryTracker)
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
