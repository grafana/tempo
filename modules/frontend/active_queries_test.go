package frontend

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/user"
	"github.com/stretchr/testify/require"
)

func TestActiveQueryTracker(t *testing.T) {
	tracker := newActiveQueryTracker(log.NewNopLogger())
	started := make(chan struct{})
	finished := make(chan struct{})
	handler := tracker.wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(started)
		<-finished
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/search?q=%7Bstatus%3Derror%7D", nil)
	req = req.WithContext(user.InjectOrgID(req.Context(), "tenant-a"))
	go handler.ServeHTTP(httptest.NewRecorder(), req)
	<-started
	stopGRPCQuery := tracker.track(
		user.InjectOrgID(context.Background(), "tenant-b"),
		"GRPC",
		"/tempopb.StreamingQuerier/Search",
		"{ status = error }",
	)
	defer stopGRPCQuery()

	rec := httptest.NewRecorder()
	statusReq := httptest.NewRequest(http.MethodGet, "/api/status/active_queries", nil)
	statusReq = statusReq.WithContext(user.InjectOrgID(statusReq.Context(), "tenant-a"))
	tracker.handler().ServeHTTP(rec, statusReq)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var response struct {
		Queries []activeQuery `json:"queries"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&response))
	require.Len(t, response.Queries, 1)
	require.Equal(t, "tenant-a", response.Queries[0].Tenant)
	require.Equal(t, http.MethodGet, response.Queries[0].Method)
	require.Equal(t, "/api/search", response.Queries[0].Path)
	require.Equal(t, "q=%7Bstatus%3Derror%7D", response.Queries[0].Query)
	require.WithinDuration(t, time.Now(), response.Queries[0].StartedAt, time.Second)

	tenantBQueries := tracker.activeQueries("tenant-b")
	require.Len(t, tenantBQueries, 1)
	require.Equal(t, "GRPC", tenantBQueries[0].Method)
	require.Equal(t, "/tempopb.StreamingQuerier/Search", tenantBQueries[0].Path)
	require.Equal(t, "{ status = error }", tenantBQueries[0].Query)

	unauthorized := httptest.NewRecorder()
	tracker.handler().ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/status/active_queries", nil))
	require.Equal(t, http.StatusUnauthorized, unauthorized.Code)

	close(finished)
	require.Eventually(t, func() bool {
		return len(tracker.activeQueries("tenant-a")) == 0
	}, time.Second, 10*time.Millisecond)
}
