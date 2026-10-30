package frontend

import (
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

func TestNewQueryFrontendTracksEnabledRequests(t *testing.T) {
	cfg := &Config{}
	cfg.RegisterFlagsAndApplyDefaults("", flag.NewFlagSet("test", flag.ContinueOnError))
	cfg.QueryTracker.Enabled = true
	f := frontendWithSettings(t, nil, nil, cfg, nil)
	require.NotNil(t, f.QueryTrackerHandler)
	require.NotNil(t, f.QueryTrackerTenantHandler)
	const tenantID = "tenant-a"
	const query = "{ .service.name = \"api\" }"

	trackedQueries := func(t *testing.T) []trackedQuery {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/status/active-queries/"+tenantID, nil)
		req.Header.Set("Accept", "application/json")
		req = mux.SetURLVars(req, map[string]string{"tenant": tenantID})
		response := httptest.NewRecorder()
		f.QueryTrackerTenantHandler.ServeHTTP(response, req)
		require.Equal(t, http.StatusOK, response.Code)
		var page queryTrackerPage
		require.NoError(t, json.NewDecoder(response.Body).Decode(&page))
		return page.Queries
	}
	assertTracked := func(t *testing.T, method, path string) {
		t.Helper()
		queries := trackedQueries(t)
		require.Len(t, queries, 1)
		require.Equal(t, tenantID, queries[0].Tenant)
		require.Equal(t, method, queries[0].Method)
		require.Equal(t, path, queries[0].Path)
		require.Equal(t, query, queries[0].Query)
	}

	t.Run("HTTP", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			path    string
			handler http.Handler
		}{
			{name: "trace", path: "/api/traces/1234", handler: f.TraceByIDHandler},
			{name: "trace v2", path: "/api/v2/traces/1234", handler: f.TraceByIDHandlerV2},
			{name: "trace diff", path: "/api/v2/traces/diff", handler: f.TraceDiffHandler},
			{name: "search", path: "/api/search", handler: f.SearchHandler},
			{name: "tags", path: "/api/search/tags", handler: f.SearchTagsHandler},
			{name: "tags v2", path: "/api/v2/search/tags", handler: f.SearchTagsV2Handler},
			{name: "tag values", path: "/api/search/tag/service.name/values", handler: f.SearchTagsValuesHandler},
			{name: "tag values v2", path: "/api/v2/search/tag/service.name/values", handler: f.SearchTagsValuesV2Handler},
			{name: "metrics instant", path: "/api/metrics/query", handler: f.MetricsQueryInstantHandler},
			{name: "metrics range", path: "/api/metrics/query_range", handler: f.MetricsQueryRangeHandler},
		} {
			t.Run(tc.name, func(t *testing.T) {
				// Replace the downstream work, preserving the tracker wired by New.
				tc.handler.(*handler).roundTripper = testRoundTripper(func(*http.Request) (*http.Response, error) {
					assertTracked(t, http.MethodGet, tc.path)
					return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: http.NoBody}, nil
				})
				req := httptest.NewRequest(http.MethodGet, tc.path+"?q="+url.QueryEscape(query), nil)
				req = req.WithContext(user.InjectOrgID(req.Context(), tenantID))
				response := httptest.NewRecorder()
				tc.handler.ServeHTTP(response, req)
				require.Equal(t, http.StatusOK, response.Code)
				require.Empty(t, trackedQueries(t))
			})
		}
	})

	t.Run("gRPC", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			invoke func(func() error) error
		}{
			{name: "Search", invoke: func(run func() error) error {
				f.streamingSearch = func(*tempopb.SearchRequest, tempopb.StreamingQuerier_SearchServer) error { return run() }
				return f.Search(&tempopb.SearchRequest{Query: query}, newMockStreamingServer[*tempopb.SearchResponse](tenantID, nil))
			}},
			{name: "SearchTags", invoke: func(run func() error) error {
				f.streamingTags = func(*tempopb.SearchTagsRequest, tempopb.StreamingQuerier_SearchTagsServer) error { return run() }
				return f.SearchTags(&tempopb.SearchTagsRequest{Query: query}, newMockStreamingServer[*tempopb.SearchTagsResponse](tenantID, nil))
			}},
			{name: "SearchTagsV2", invoke: func(run func() error) error {
				f.streamingTagsV2 = func(*tempopb.SearchTagsRequest, tempopb.StreamingQuerier_SearchTagsV2Server) error { return run() }
				return f.SearchTagsV2(&tempopb.SearchTagsRequest{Query: query}, newMockStreamingServer[*tempopb.SearchTagsV2Response](tenantID, nil))
			}},
			{name: "SearchTagValues", invoke: func(run func() error) error {
				f.streamingTagValues = func(*tempopb.SearchTagValuesRequest, tempopb.StreamingQuerier_SearchTagValuesServer) error {
					return run()
				}
				return f.SearchTagValues(&tempopb.SearchTagValuesRequest{Query: query}, newMockStreamingServer[*tempopb.SearchTagValuesResponse](tenantID, nil))
			}},
			{name: "SearchTagValuesV2", invoke: func(run func() error) error {
				f.streamingTagValuesV2 = func(*tempopb.SearchTagValuesRequest, tempopb.StreamingQuerier_SearchTagValuesV2Server) error {
					return run()
				}
				return f.SearchTagValuesV2(&tempopb.SearchTagValuesRequest{Query: query}, newMockStreamingServer[*tempopb.SearchTagValuesV2Response](tenantID, nil))
			}},
			{name: "MetricsQueryRange", invoke: func(run func() error) error {
				f.streamingQueryRange = func(*tempopb.QueryRangeRequest, tempopb.StreamingQuerier_MetricsQueryRangeServer) error { return run() }
				return f.MetricsQueryRange(&tempopb.QueryRangeRequest{Query: query}, newMockStreamingServer[*tempopb.QueryRangeResponse](tenantID, nil))
			}},
			{name: "MetricsQueryInstant", invoke: func(run func() error) error {
				f.streamingQueryInstant = func(*tempopb.QueryInstantRequest, tempopb.StreamingQuerier_MetricsQueryInstantServer) error {
					return run()
				}
				return f.MetricsQueryInstant(&tempopb.QueryInstantRequest{Query: query}, newMockStreamingServer[*tempopb.QueryInstantResponse](tenantID, nil))
			}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				require.NoError(t, tc.invoke(func() error {
					assertTracked(t, "GRPC", "/tempopb.StreamingQuerier/"+tc.name)
					return nil
				}))
				require.Empty(t, trackedQueries(t))
			})
		}
	})
}

func TestQueryFrontendTracksStreamingQueries(t *testing.T) {
	const query = "{ .status = \"error\" }"
	for _, outcome := range []struct {
		name string
		fail bool
	}{
		{name: "success", fail: false},
		{name: "error", fail: true},
	} {
		t.Run(outcome.name, func(t *testing.T) {
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
					recorder := &recordingQueryTracker{}
					f := &QueryFrontend{queryTracker: recorder}
					expectedErr := errors.New("handler failed")
					err := tc.invoke(f, func() error {
						require.Equal(t, 1, recorder.pushes)
						require.Zero(t, recorder.pops)
						if outcome.fail {
							return expectedErr
						}
						return nil
					})
					if outcome.fail {
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

func TestQueryFrontendTracksHTTPHandlers(t *testing.T) {
	f := frontendWithSettings(t, nil, nil, nil, nil)
	for name, h := range map[string]http.Handler{
		"trace": f.TraceByIDHandler, "trace v2": f.TraceByIDHandlerV2, "trace diff": f.TraceDiffHandler,
		"search": f.SearchHandler, "tags": f.SearchTagsHandler, "tags v2": f.SearchTagsV2Handler,
		"values": f.SearchTagsValuesHandler, "values v2": f.SearchTagsValuesV2Handler,
		"instant": f.MetricsQueryInstantHandler, "range": f.MetricsQueryRangeHandler,
	} {
		t.Run(name, func(t *testing.T) {
			for _, outcome := range []struct {
				name string
				fail bool
			}{
				{name: "success", fail: false},
				{name: "error", fail: true},
			} {
				t.Run(outcome.name, func(t *testing.T) {
					recorder := &recordingQueryTracker{}
					wrapped := h.(*handler)
					wrapped.queryTracker = recorder
					wrapped.roundTripper = testRoundTripper(func(*http.Request) (*http.Response, error) {
						require.Equal(t, 1, recorder.pushes)
						require.Zero(t, recorder.pops)
						if outcome.fail {
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
	tracker := newQueryTracker(QueryTrackerConfig{Enabled: true, MaxQueriesPerTenant: 2, MaxQueryTextBytes: 4096}, log.NewNopLogger(), nil).(*queryTracker)
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
	h := newHandler(nil, testRoundTripper(func(*http.Request) (*http.Response, error) {
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
	tracker := newQueryTracker(QueryTrackerConfig{Enabled: true, MaxQueriesPerTenant: 1, MaxQueryTextBytes: 3}, log.NewNopLogger(), nil).(*queryTracker)
	ctx := user.InjectOrgID(context.Background(), "tenant-a")
	stop := tracker.track(ctx, "GET", "/api/search", func() string { return "abé" })
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
