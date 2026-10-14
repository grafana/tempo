package frontend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/go-kit/log"
	"github.com/gorilla/mux"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/stretchr/testify/require"
)

func TestHandlerLogsConfiguredRequestHeaders(t *testing.T) {
	var logs bytes.Buffer
	h := newHandler([]string{"X-Request-ID"}, testRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: http.NoBody}, nil
	}), log.NewLogfmtLogger(&logs))
	req := httptest.NewRequest(http.MethodGet, "/api/search", nil)
	req.Header.Set("X-Request-ID", "request-123")
	req.Header.Set("X-Unconfigured-Header", "do-not-log")
	h.ServeHTTP(httptest.NewRecorder(), req)
	require.Contains(t, logs.String(), "header_x_request_id=request-123")
	require.NotContains(t, logs.String(), "do-not-log")
}

type lifecycleQueryTracker struct {
	pushes, pops                int
	method, path, query, tenant string
}

func (r *lifecycleQueryTracker) track(ctx context.Context, method, path, query string) func() {
	r.pushes++
	r.method, r.path, r.query = method, path, query
	r.tenant, _ = user.ExtractOrgID(ctx)
	return func() { r.pops++ }
}

func TestQueryFrontendTracksStreamingQueries(t *testing.T) {
	const query = "{ .status = \"error\" }"
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "error"}[fail], func(t *testing.T) {
			tests := []struct {
				name   string
				invoke func(*QueryFrontend, func() error) error
			}{
				{name: "Search", invoke: func(f *QueryFrontend, run func() error) error {
					f.streamingSearch = func(*tempopb.SearchRequest, tempopb.StreamingQuerier_SearchServer) error { return run() }
					return f.Search(&tempopb.SearchRequest{Query: query}, newMockStreamingServer[*tempopb.SearchResponse]("tenant-a", nil))
				}},
				{name: "SearchTags", invoke: func(f *QueryFrontend, run func() error) error {
					f.streamingTags = func(*tempopb.SearchTagsRequest, tempopb.StreamingQuerier_SearchTagsServer) error { return run() }
					return f.SearchTags(&tempopb.SearchTagsRequest{Query: query}, newMockStreamingServer[*tempopb.SearchTagsResponse]("tenant-a", nil))
				}},
				{name: "SearchTagsV2", invoke: func(f *QueryFrontend, run func() error) error {
					f.streamingTagsV2 = func(*tempopb.SearchTagsRequest, tempopb.StreamingQuerier_SearchTagsV2Server) error { return run() }
					return f.SearchTagsV2(&tempopb.SearchTagsRequest{Query: query}, newMockStreamingServer[*tempopb.SearchTagsV2Response]("tenant-a", nil))
				}},
				{name: "SearchTagValues", invoke: func(f *QueryFrontend, run func() error) error {
					f.streamingTagValues = func(*tempopb.SearchTagValuesRequest, tempopb.StreamingQuerier_SearchTagValuesServer) error {
						return run()
					}
					return f.SearchTagValues(&tempopb.SearchTagValuesRequest{Query: query}, newMockStreamingServer[*tempopb.SearchTagValuesResponse]("tenant-a", nil))
				}},
				{name: "SearchTagValuesV2", invoke: func(f *QueryFrontend, run func() error) error {
					f.streamingTagValuesV2 = func(*tempopb.SearchTagValuesRequest, tempopb.StreamingQuerier_SearchTagValuesV2Server) error {
						return run()
					}
					return f.SearchTagValuesV2(&tempopb.SearchTagValuesRequest{Query: query}, newMockStreamingServer[*tempopb.SearchTagValuesV2Response]("tenant-a", nil))
				}},
				{name: "MetricsQueryRange", invoke: func(f *QueryFrontend, run func() error) error {
					f.streamingQueryRange = func(*tempopb.QueryRangeRequest, tempopb.StreamingQuerier_MetricsQueryRangeServer) error { return run() }
					return f.MetricsQueryRange(&tempopb.QueryRangeRequest{Query: query}, newMockStreamingServer[*tempopb.QueryRangeResponse]("tenant-a", nil))
				}},
				{name: "MetricsQueryInstant", invoke: func(f *QueryFrontend, run func() error) error {
					f.streamingQueryInstant = func(*tempopb.QueryInstantRequest, tempopb.StreamingQuerier_MetricsQueryInstantServer) error {
						return run()
					}
					return f.MetricsQueryInstant(&tempopb.QueryInstantRequest{Query: query}, newMockStreamingServer[*tempopb.QueryInstantResponse]("tenant-a", nil))
				}},
			}
			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					recorder := &lifecycleQueryTracker{}
					f := &QueryFrontend{queryTracker: recorder}
					expectedErr := errors.New("handler failed")
					err := tc.invoke(f, func() error {
						require.Equal(t, 1, recorder.pushes)
						require.Zero(t, recorder.pops)
						if fail {
							return expectedErr
						}
						return nil
					})
					if fail {
						require.ErrorIs(t, err, expectedErr)
					} else {
						require.NoError(t, err)
					}
					require.Equal(t, 1, recorder.pushes)
					require.Equal(t, 1, recorder.pops)
					require.Equal(t, "GRPC", recorder.method)
					require.Equal(t, "/tempopb.StreamingQuerier/"+tc.name, recorder.path)
					require.Equal(t, query, recorder.query)
					require.Equal(t, "tenant-a", recorder.tenant)
				})
			}
		})
	}
}

func TestHTTPQueryTrackerLifecycle(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "error"}[fail], func(t *testing.T) {
			recorder := &lifecycleQueryTracker{}
			h := newHandlerWithTracker(nil, testRoundTripper(func(*http.Request) (*http.Response, error) {
				require.Equal(t, 1, recorder.pushes)
				require.Zero(t, recorder.pops)
				if fail {
					return nil, errors.New("upstream failed")
				}
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: http.NoBody}, nil
			}), log.NewNopLogger(), recorder)
			req := httptest.NewRequest(http.MethodGet, "/api/search?q=%7B%7D", nil)
			req = req.WithContext(user.InjectOrgID(req.Context(), "tenant-a"))
			response := httptest.NewRecorder()
			h.ServeHTTP(response, req)
			require.Equal(t, 1, recorder.pushes)
			require.Equal(t, 1, recorder.pops)
			if fail {
				require.Equal(t, http.StatusInternalServerError, response.Code)
			} else {
				require.Equal(t, http.StatusOK, response.Code)
			}
		})
	}
}

func TestQueryFrontendTracksHTTPHandlers(t *testing.T) {
	f := frontendWithSettings(t, nil, nil, nil, nil)
	for name, h := range map[string]http.Handler{
		"trace": f.TraceByIDHandler, "trace v2": f.TraceByIDHandlerV2, "trace diff": f.TraceDiffHandler,
		"search": f.SearchHandler, "tags": f.SearchTagsHandler, "tags v2": f.SearchTagsV2Handler,
		"values": f.SearchTagsValuesHandler, "values v2": f.SearchTagsValuesV2Handler,
		"instant": f.MetricsQueryInstantHandler, "range": f.MetricsQueryRangeHandler,
	} {
		t.Run(name, func(t *testing.T) {
			for _, fail := range []bool{false, true} {
				t.Run(map[bool]string{false: "success", true: "error"}[fail], func(t *testing.T) {
					recorder := &lifecycleQueryTracker{}
					wrapped := h.(*handler)
					wrapped.queryTracker = recorder
					wrapped.roundTripper = testRoundTripper(func(*http.Request) (*http.Response, error) {
						require.Equal(t, 1, recorder.pushes)
						require.Zero(t, recorder.pops)
						if fail {
							return nil, errors.New("upstream failed")
						}
						return &http.Response{StatusCode: 200, Header: make(http.Header), Body: http.NoBody}, nil
					})
					req := httptest.NewRequest("GET", "/api/search?q=%7B%7D", nil)
					wrapped.ServeHTTP(httptest.NewRecorder(), req.WithContext(user.InjectOrgID(req.Context(), "tenant-a")))
					require.Equal(t, 1, recorder.pushes)
					require.Equal(t, 1, recorder.pops)
					require.Equal(t, "{}", recorder.query)
					require.Equal(t, "tenant-a", recorder.tenant)
				})
			}
		})
	}
}

func TestDisabledQueryTrackerDoesNotTrackFrontendRequests(t *testing.T) {
	cfg := &Config{}
	cfg.RegisterFlagsAndApplyDefaults("", flag.NewFlagSet("test", flag.ContinueOnError))
	f := frontendWithSettings(t, nil, nil, cfg, nil)
	require.IsType(t, noOpQueryTracker{}, f.queryTracker)
	require.Nil(t, f.QueryTrackerHandler)
	require.Nil(t, f.QueryTrackerTenantHandler)
	require.Error(t, f.Search(&tempopb.SearchRequest{Query: "invalid"}, newMockStreamingServer[*tempopb.SearchResponse]("tenant-a", nil)))
	for name, h := range map[string]http.Handler{
		"trace": f.TraceByIDHandler, "trace v2": f.TraceByIDHandlerV2, "trace diff": f.TraceDiffHandler,
		"search": f.SearchHandler, "tags": f.SearchTagsHandler, "tags v2": f.SearchTagsV2Handler,
		"values": f.SearchTagsValuesHandler, "values v2": f.SearchTagsValuesV2Handler,
		"instant": f.MetricsQueryInstantHandler, "range": f.MetricsQueryRangeHandler,
	} {
		t.Run(name, func(t *testing.T) {
			require.IsType(t, noOpQueryTracker{}, h.(*handler).queryTracker)
			req := httptest.NewRequest(http.MethodGet, "/api/search?q=invalid", nil)
			req = req.WithContext(user.InjectOrgID(req.Context(), "tenant-a"))
			h.ServeHTTP(httptest.NewRecorder(), req)
		})
	}
}

func TestQueryTrackerShowsActiveStreamingQuery(t *testing.T) {
	tracker := newQueryTracker(QueryTrackerConfig{Enabled: true, MaxQueriesPerTenant: 2, MaxQueryTextBytes: 4096}).(*queryTracker)
	entered, release := make(chan struct{}), make(chan struct{})
	ctx, cancel := context.WithTimeout(user.InjectOrgID(context.Background(), "tenant-a"), 5*time.Second)
	defer cancel()
	defer close(release)
	f := &QueryFrontend{queryTracker: tracker, streamingSearch: func(_ *tempopb.SearchRequest, srv tempopb.StreamingQuerier_SearchServer) error {
		close(entered)
		select {
		case <-release:
			return nil
		case <-srv.Context().Done():
			return srv.Context().Err()
		}
	}}
	const query = "{ .status = \"error\" }"
	done := make(chan error, 1)
	go func() {
		done <- f.Search(&tempopb.SearchRequest{Query: query}, &mockGRPCStreaming[*tempopb.SearchResponse]{ctx: ctx})
	}()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	require.Len(t, tracker.trackedQueries("tenant-a"), 1)
	require.Equal(t, query, tracker.trackedQueries("tenant-a")[0].Query)
	page := httptest.NewRecorder()
	pageRequest := httptest.NewRequest(http.MethodGet, "/status/active-queries/tenant-a", nil)
	pageRequest.Header.Set("Accept", "application/json")
	pageRequest = mux.SetURLVars(pageRequest, map[string]string{"tenant": "tenant-a"})
	tracker.tenantHandler().ServeHTTP(page, pageRequest)
	var activePage queryTrackerPage
	require.NoError(t, json.NewDecoder(page.Body).Decode(&activePage))
	require.Len(t, activePage.Queries, 1)
	require.Equal(t, query, activePage.Queries[0].Query)
	// HTTP and gRPC store the same decoded expression.
	h := newHandlerWithTracker(nil, testRoundTripper(func(*http.Request) (*http.Response, error) {
		queries := tracker.trackedQueries("tenant-a")
		require.Len(t, queries, 2)
		require.Equal(t, queries[0].Query, queries[1].Query)
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: http.NoBody}, nil
	}), log.NewNopLogger(), tracker)
	req := httptest.NewRequest(http.MethodGet, "/api/search?q="+url.QueryEscape(query), nil)
	h.ServeHTTP(httptest.NewRecorder(), req.WithContext(ctx))
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
	require.Empty(t, tracker.trackedQueries("tenant-a"))
	require.Empty(t, tracker.trackedTenants())
	after := httptest.NewRecorder()
	tracker.tenantHandler().ServeHTTP(after, pageRequest)
	var finishedPage queryTrackerPage
	require.NoError(t, json.NewDecoder(after.Body).Decode(&finishedPage))
	require.Empty(t, finishedPage.Queries)
}

func TestQueryTrackerCapDoesNotBlockQueries(t *testing.T) {
	tracker := newQueryTracker(QueryTrackerConfig{Enabled: true, MaxQueriesPerTenant: 1, MaxQueryTextBytes: 3}).(*queryTracker)
	ctx := user.InjectOrgID(context.Background(), "tenant-a")
	stop := tracker.track(ctx, "GET", "/api/search", "abé")
	defer stop()
	require.Equal(t, "ab", tracker.trackedQueries("tenant-a")[0].Query)
	require.True(t, utf8.ValidString(tracker.trackedQueries("tenant-a")[0].Query))
	ran := false
	f := &QueryFrontend{queryTracker: tracker, streamingSearch: func(*tempopb.SearchRequest, tempopb.StreamingQuerier_SearchServer) error {
		ran = true
		require.Len(t, tracker.trackedQueries("tenant-a"), 1)
		return nil
	}}
	require.NoError(t, f.Search(&tempopb.SearchRequest{Query: "{}"}, newMockStreamingServer[*tempopb.SearchResponse]("tenant-a", nil)))
	require.True(t, ran)
	require.Equal(t, uint64(1), tracker.trackedTenants()[0].Dropped)
	page := httptest.NewRecorder()
	tracker.tenantsHandler().ServeHTTP(page, httptest.NewRequest("GET", "/status/active-queries", nil))
	require.Contains(t, page.Body.String(), "Maximum tracked queries per tenant: 1")
	require.Contains(t, page.Body.String(), "<td>1</td><td>1</td>")
	stop()
	require.Empty(t, tracker.trackedQueries("tenant-a"))
	require.Zero(t, tracker.trackedTenants()[0].Active)
}
