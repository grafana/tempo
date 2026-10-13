package frontend

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestActiveQueryTracker(t *testing.T) {
	tracker := newActiveQueryTracker(QueryTrackerConfig{MaxQueriesPerTenant: 1, MaxQueryTextBytes: 8})
	stop := tracker.track(user.InjectOrgID(context.Background(), "tenant-a"), http.MethodGet, "/api/search", "{ .foo = \"é\" }")
	defer stop()
	tracker.track(user.InjectOrgID(context.Background(), "tenant-a"), "GRPC", "/tempopb.StreamingQuerier/Search", "{ .bar }")()

	queries := tracker.activeQueries("tenant-a")
	require.Len(t, queries, 1)
	require.Equal(t, "{ .foo =", queries[0].Query)
	require.True(t, time.Now().After(queries[0].StartedAt))
	require.Equal(t, uint64(1), tracker.tenants()[0].Dropped)

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
		var page activeQueriesPage
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
		var page activeQueriesPage
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
		_, ok := tracker.(noopQueryTracker)
		require.True(t, ok)
	})

	require.Equal(t, "ab", truncateQuery("abé", 3))
	require.False(t, strings.Contains(truncateQuery("abc", 2), "\x00"))
}

func TestHandlerTracksDecodedQueryText(t *testing.T) {
	tracker := &recordingQueryTracker{}
	handler := newHandlerWithTracker(nil, testRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusInternalServerError, Header: make(http.Header), Body: http.NoBody}, nil
	}), log.NewNopLogger(), tracker)

	req := httptest.NewRequest(http.MethodGet, "/api/search?q=%7B+resource.service.name+%3D+%22api%22+%7D", nil)
	req = req.WithContext(user.InjectOrgID(req.Context(), "tenant-a"))
	handler.ServeHTTP(httptest.NewRecorder(), req)

	require.Equal(t, http.MethodGet, tracker.method)
	require.Equal(t, "/api/search", tracker.path)
	require.Equal(t, `{ resource.service.name = "api" }`, tracker.query)
}
