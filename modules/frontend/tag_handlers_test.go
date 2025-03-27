package frontend

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"sort"
	"testing"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/status"
	"github.com/gorilla/mux"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/cache"
	"github.com/grafana/tempo/pkg/search"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/segmentio/fasthash/fnv1a"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

// these are integration tests against the search http and streaming pipelines. they could be extended to handle all
// endpoints as we migrate them to the new pipelines and have common expectations on behaviors.
// todo: build a test harness that extends these to all endpoints
func TestFrontendTags(t *testing.T) {
	allRunners := []func(t *testing.T, f *QueryFrontend){
		runnerTagsBadRequestOnOrgID,
		runnerTagsV2BadRequestOnOrgID,
		runnerTagValuesBadRequestOnOrgID,
		runnerTagValuesV2BadRequestOnOrgID,
		runnerTagsV2ClientCancelContext,
		runnerTagValuesV2ClientCancelContext,
	}

	for _, runner := range allRunners {
		f := frontendWithSettings(t, nil, nil, nil, nil)
		runner(t, f)
	}
}

func runnerTagsBadRequestOnOrgID(t *testing.T, f *QueryFrontend) {
	// http
	httpReq := httptest.NewRequest("GET", "/api/search/tags", nil)
	httpResp := httptest.NewRecorder()
	f.SearchTagsHandler.ServeHTTP(httpResp, httpReq)
	require.Equal(t, "no org id", httpResp.Body.String())
	require.Equal(t, http.StatusBadRequest, httpResp.Code)

	// grpc
	grpcReq := &tempopb.SearchTagsRequest{}
	err := f.streamingTags(grpcReq, newMockStreamingServer[*tempopb.SearchTagsResponse]("", nil))
	require.Equal(t, status.Error(codes.InvalidArgument, "no org id"), err)
}

func runnerTagsV2BadRequestOnOrgID(t *testing.T, f *QueryFrontend) {
	// http
	httpReq := httptest.NewRequest("GET", "/api/v2/search/tags", nil)
	httpResp := httptest.NewRecorder()
	f.SearchTagsV2Handler.ServeHTTP(httpResp, httpReq)
	require.Equal(t, "no org id", httpResp.Body.String())
	require.Equal(t, http.StatusBadRequest, httpResp.Code)

	// grpc
	grpcReq := &tempopb.SearchTagsRequest{}
	err := f.streamingTagsV2(grpcReq, newMockStreamingServer[*tempopb.SearchTagsV2Response]("", nil))
	require.Equal(t, status.Error(codes.InvalidArgument, "no org id"), err)
}

func runnerTagValuesBadRequestOnOrgID(t *testing.T, f *QueryFrontend) {
	// http
	httpReq := httptest.NewRequest("GET", "/api/search/tag/foo/values", nil)
	httpReq = mux.SetURLVars(httpReq, map[string]string{"tagName": "foo"})
	httpResp := httptest.NewRecorder()
	f.SearchTagsValuesHandler.ServeHTTP(httpResp, httpReq)
	require.Equal(t, "no org id", httpResp.Body.String())
	require.Equal(t, http.StatusBadRequest, httpResp.Code)

	// grpc
	grpcReq := &tempopb.SearchTagValuesRequest{}
	err := f.streamingTagValues(grpcReq, newMockStreamingServer[*tempopb.SearchTagValuesResponse]("", nil))
	require.Equal(t, status.Error(codes.InvalidArgument, "no org id"), err)
}

func runnerTagValuesV2BadRequestOnOrgID(t *testing.T, f *QueryFrontend) {
	// http
	httpReq := httptest.NewRequest("GET", "/api/v2/search/tag/foo/values", nil)
	httpReq = mux.SetURLVars(httpReq, map[string]string{"tagName": "foo"})
	httpResp := httptest.NewRecorder()
	f.SearchTagsValuesV2Handler.ServeHTTP(httpResp, httpReq)
	require.Equal(t, "no org id", httpResp.Body.String())
	require.Equal(t, http.StatusBadRequest, httpResp.Code)

	// grpc
	grpcReq := &tempopb.SearchTagValuesRequest{}
	err := f.streamingTagValuesV2(grpcReq, newMockStreamingServer[*tempopb.SearchTagValuesV2Response]("", nil))
	require.Equal(t, status.Error(codes.InvalidArgument, "no org id"), err)
}

func runnerTagsV2ClientCancelContext(t *testing.T, f *QueryFrontend) {
	// http
	httpReq := httptest.NewRequest("GET", "/api/v2/search/tags", nil)
	httpResp := httptest.NewRecorder()

	ctx, cancel := context.WithCancel(httpReq.Context())
	ctx = user.InjectOrgID(ctx, "tenant")

	httpReq = httpReq.WithContext(ctx)
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	f.SearchTagsV2Handler.ServeHTTP(httpResp, httpReq)
	require.Equal(t, "context canceled", httpResp.Body.String())
	require.Equal(t, 499, httpResp.Code) // todo: is this 499 valid?

	// grpc
	srv := newMockStreamingServer[*tempopb.SearchTagsV2Response]("tenant", nil)
	srv.ctx, cancel = context.WithCancel(srv.ctx)
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	grpcReq := &tempopb.SearchTagsRequest{}
	err := f.streamingTagsV2(grpcReq, srv)
	require.Equal(t, status.Error(codes.Canceled, "context canceled"), err)
}

func runnerTagValuesV2ClientCancelContext(t *testing.T, f *QueryFrontend) {
	// http
	httpReq := httptest.NewRequest("GET", "/api/v2/search/tag/foo/values", nil)
	httpReq = mux.SetURLVars(httpReq, map[string]string{"tagName": "foo"})
	httpResp := httptest.NewRecorder()

	ctx, cancel := context.WithCancel(httpReq.Context())
	ctx = user.InjectOrgID(ctx, "tenant")

	httpReq = httpReq.WithContext(ctx)
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	f.SearchTagsValuesV2Handler.ServeHTTP(httpResp, httpReq)
	require.Equal(t, "context canceled", httpResp.Body.String())
	require.Equal(t, 499, httpResp.Code) // todo: is this 499 valid?

	// grpc
	srv := newMockStreamingServer[*tempopb.SearchTagValuesV2Response]("tenant", nil)
	srv.ctx, cancel = context.WithCancel(srv.ctx)
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	grpcReq := &tempopb.SearchTagValuesRequest{
		TagName: "foo",
	}
	err := f.streamingTagValuesV2(grpcReq, srv)
	require.Equal(t, status.Error(codes.Canceled, "context canceled"), err)
}

func TestSearchTagsV2Intrinsics(t *testing.T) {
	mockScope := "span"
	mockTags := []string{"foo", "bar"}

	tcs := []struct {
		name        string
		maxTagBytes int
		expected    *tempopb.SearchTagsV2Response
	}{
		{
			name:        "unlimited",
			maxTagBytes: 0,
			expected: &tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
					{
						Name: api.ParamScopeIntrinsic,
						Tags: search.GetVirtualIntrinsicValues(),
					},
					{
						Name: mockScope,
						Tags: mockTags,
					},
				},
			},
		},
		{
			name:        "when_limited_intrinsics_first",
			maxTagBytes: 100,
			expected: &tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
					{
						// Only a subset of intrinsic tags will fit
						Name: api.ParamScopeIntrinsic,
						Tags: search.GetVirtualIntrinsicValues()[0:10],
					},
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// This is the mocked data returned by querier/ingester jobs downstream.
			next := &mockRoundTripper{
				responseFn: func() proto.Message {
					return &tempopb.SearchTagsV2Response{
						Scopes: []*tempopb.SearchTagsV2Scope{
							{
								Name: mockScope,
								Tags: mockTags,
							},
						},
					}
				},
			}

			f := frontendWithSettings(t, next, nil, nil, nil, func(_ *Config, overridesCfg *overrides.Config) {
				overridesCfg.Defaults.Read.MaxBytesPerTagValuesQuery = tc.maxTagBytes
			})

			// http
			httpReq := httptest.NewRequest("GET", "/api/v2/search/tags", nil)
			httpResp := httptest.NewRecorder()

			ctx, cancel := context.WithCancel(httpReq.Context())
			defer cancel()
			ctx = user.InjectOrgID(ctx, "tenant")
			httpReq = httpReq.WithContext(ctx)

			f.SearchTagsV2Handler.ServeHTTP(httpResp, httpReq)
			require.Equal(t, http.StatusOK, httpResp.Code)

			resp := &tempopb.SearchTagsV2Response{}
			bytesResp, err := io.ReadAll(httpResp.Body)
			require.NoError(t, err)
			err = jsonpb.Unmarshal(bytes.NewReader(bytesResp), resp)

			require.NoError(t, err)

			// Sort scopes to give stable comparison
			sort.Slice(tc.expected.Scopes, func(i, j int) bool {
				return tc.expected.Scopes[i].Name < tc.expected.Scopes[j].Name
			})
			sort.Slice(resp.Scopes, func(i, j int) bool {
				return resp.Scopes[i].Name < resp.Scopes[j].Name
			})
			require.Equal(t, len(tc.expected.Scopes), len(resp.Scopes))
			for i := range tc.expected.Scopes {
				require.ElementsMatch(t, tc.expected.Scopes[i].Tags, resp.Scopes[i].Tags)
			}
		})
	}
}

// todo: a lot of code is replicated between all of these "failure propagates from queriers" tests. we should refactor
// to a framework that tests this against all endpoints
func TestSearchTagsV2FailurePropagatesFromQueriers(t *testing.T) {
	tcs := []struct {
		name           string
		querierCode    int
		querierMessage string
		querierErr     error

		expectedCode    int
		expectedMessage string
		expectedErr     error
	}{
		{
			name:            "querier 500s",
			querierCode:     500,
			querierMessage:  "querier 500",
			expectedCode:    500,
			expectedMessage: "querier 500",
			expectedErr:     status.Error(codes.Internal, "querier 500"),
		},
		{
			name:            "querier errors",
			querierErr:      errors.New("querier error"),
			expectedCode:    500,
			expectedMessage: "querier error\n", // i don't know why there's a newline here, but there is
			expectedErr:     status.Error(codes.Internal, "querier error"),
		},
		{
			name:            "querier 404s - translated to 500",
			querierCode:     404,
			querierMessage:  "not found!",
			expectedCode:    500,
			expectedMessage: "not found!",
			expectedErr:     status.Error(codes.Internal, "not found!"),
		},
		{
			name:            "querier 429 - stays 429",
			querierCode:     429,
			querierMessage:  "too fast!",
			expectedCode:    429,
			expectedMessage: "too fast!",
			expectedErr:     status.Error(codes.ResourceExhausted, "too fast!"),
		},
	}

	for _, tc := range tcs {
		// queriers will return one errr
		f := frontendWithSettings(t, &mockRoundTripper{
			statusCode:    tc.querierCode,
			statusMessage: tc.querierMessage,
			err:           tc.querierErr,
			responseFn: func() proto.Message {
				return &tempopb.SearchTagsResponse{}
			},
		}, nil, &Config{
			MultiTenantQueriesEnabled: true,
			MaxRetries:                0, // disable retries or it will try twice and get success. the querier response is designed to fail exactly once
			TraceByID: TraceByIDConfig{
				QueryShards: minQueryShards,
				SLO:         testSLOcfg,
			},
			Search: SearchConfig{
				Sharder: SearchSharderConfig{
					ConcurrentRequests:    defaultConcurrentRequests,
					TargetBytesPerRequest: defaultTargetBytesPerRequest,
					MostRecentShards:      defaultMostRecentShards,
				},
				SLO: testSLOcfg,
			},
			Metrics: MetricsConfig{
				Sharder: QueryRangeSharderConfig{
					ConcurrentRequests:    defaultConcurrentRequests,
					TargetBytesPerRequest: defaultTargetBytesPerRequest,
					Interval:              1 * time.Second,
				},
				SLO: testSLOcfg,
			},
		}, nil)

		httpReq := httptest.NewRequest("GET", "/api/search/tags?start=1&end=10000", nil)
		httpResp := httptest.NewRecorder()

		ctx := user.InjectOrgID(httpReq.Context(), "foo")
		httpReq = httpReq.WithContext(ctx)

		f.SearchTagsV2Handler.ServeHTTP(httpResp, httpReq)
		require.Equal(t, tc.expectedMessage, httpResp.Body.String())
		require.Equal(t, tc.expectedCode, httpResp.Code)

		// have to recreate the frontend to reset the querier response
		f = frontendWithSettings(t, &mockRoundTripper{
			statusCode:    tc.querierCode,
			statusMessage: tc.querierMessage,
			err:           tc.querierErr,
			responseFn: func() proto.Message {
				return &tempopb.SearchResponse{
					Traces:  []*tempopb.TraceSearchMetadata{},
					Metrics: &tempopb.SearchMetrics{},
				}
			},
		}, nil, &Config{
			MultiTenantQueriesEnabled: true,
			MaxRetries:                0, // disable retries or it will try twice and get success
			TraceByID: TraceByIDConfig{
				QueryShards: minQueryShards,
				SLO:         testSLOcfg,
			},
			Search: SearchConfig{
				Sharder: SearchSharderConfig{
					ConcurrentRequests:    defaultConcurrentRequests,
					TargetBytesPerRequest: defaultTargetBytesPerRequest,
					MostRecentShards:      defaultMostRecentShards,
				},
				SLO: testSLOcfg,
			},
			Metrics: MetricsConfig{
				Sharder: QueryRangeSharderConfig{
					ConcurrentRequests:    defaultConcurrentRequests,
					TargetBytesPerRequest: defaultTargetBytesPerRequest,
					Interval:              1 * time.Second,
				},
				SLO: testSLOcfg,
			},
		}, nil)

		// grpc
		srv := newMockStreamingServer[*tempopb.SearchTagsV2Response]("bar", nil)
		grpcReq := &tempopb.SearchTagsRequest{}
		err := f.streamingTagsV2(grpcReq, srv)
		require.Equal(t, tc.expectedErr, err)
	}
}

func TestSearchTagValuesV2FailurePropagatesFromQueriers(t *testing.T) {
	tcs := []struct {
		name           string
		querierCode    int
		querierMessage string
		querierErr     error

		expectedCode    int
		expectedMessage string
		expectedErr     error
	}{
		{
			name:            "querier 500s",
			querierCode:     500,
			querierMessage:  "querier 500",
			expectedCode:    500,
			expectedMessage: "querier 500",
			expectedErr:     status.Error(codes.Internal, "querier 500"),
		},
		{
			name:            "querier errors",
			querierErr:      errors.New("querier error"),
			expectedCode:    500,
			expectedMessage: "querier error\n", // i don't know why there's a newline here, but there is
			expectedErr:     status.Error(codes.Internal, "querier error"),
		},
		{
			name:            "querier 404s - translated to 500",
			querierCode:     404,
			querierMessage:  "not found!",
			expectedCode:    500,
			expectedMessage: "not found!",
			expectedErr:     status.Error(codes.Internal, "not found!"),
		},
		{
			name:            "querier 429 - stays 429",
			querierCode:     429,
			querierMessage:  "too fast!",
			expectedCode:    429,
			expectedMessage: "too fast!",
			expectedErr:     status.Error(codes.ResourceExhausted, "too fast!"),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// queriers will return one err
			f := frontendWithSettings(t, &mockRoundTripper{
				statusCode:    tc.querierCode,
				statusMessage: tc.querierMessage,
				err:           tc.querierErr,
				responseFn: func() proto.Message {
					return &tempopb.SearchTagsResponse{}
				},
			}, nil, &Config{
				MultiTenantQueriesEnabled: true,
				MaxRetries:                0, // disable retries or it will try twice and get success. the querier response is designed to fail exactly once
				TraceByID: TraceByIDConfig{
					QueryShards: minQueryShards,
					SLO:         testSLOcfg,
				},
				Search: SearchConfig{
					Sharder: SearchSharderConfig{
						ConcurrentRequests:    defaultConcurrentRequests,
						TargetBytesPerRequest: defaultTargetBytesPerRequest,
						MostRecentShards:      defaultMostRecentShards,
					},
					SLO: testSLOcfg,
				},
				Metrics: MetricsConfig{
					Sharder: QueryRangeSharderConfig{
						ConcurrentRequests:    defaultConcurrentRequests,
						TargetBytesPerRequest: defaultTargetBytesPerRequest,
						Interval:              1 * time.Second,
					},
					SLO: testSLOcfg,
				},
			}, nil)

			httpReq := httptest.NewRequest("GET", "/api/v2/search/tag/foo/values?start=1&end=10000", nil)
			httpReq = mux.SetURLVars(httpReq, map[string]string{"tagName": "foo"})
			httpResp := httptest.NewRecorder()

			ctx := user.InjectOrgID(httpReq.Context(), "foo")
			httpReq = httpReq.WithContext(ctx)

			f.SearchTagsValuesV2Handler.ServeHTTP(httpResp, httpReq)
			require.Equal(t, tc.expectedMessage, httpResp.Body.String())
			require.Equal(t, tc.expectedCode, httpResp.Code)

			// have to recreate the frontend to reset the querier response
			f = frontendWithSettings(t, &mockRoundTripper{
				statusCode:    tc.querierCode,
				statusMessage: tc.querierMessage,
				err:           tc.querierErr,
				responseFn: func() proto.Message {
					return &tempopb.SearchResponse{
						Traces:  []*tempopb.TraceSearchMetadata{},
						Metrics: &tempopb.SearchMetrics{},
					}
				},
			}, nil, &Config{
				MultiTenantQueriesEnabled: true,
				MaxRetries:                0, // disable retries or it will try twice and get success
				TraceByID: TraceByIDConfig{
					QueryShards: minQueryShards,
					SLO:         testSLOcfg,
				},
				Search: SearchConfig{
					Sharder: SearchSharderConfig{
						ConcurrentRequests:    defaultConcurrentRequests,
						TargetBytesPerRequest: defaultTargetBytesPerRequest,
						MostRecentShards:      defaultMostRecentShards,
					},
					SLO: testSLOcfg,
				},
				Metrics: MetricsConfig{
					Sharder: QueryRangeSharderConfig{
						ConcurrentRequests:    defaultConcurrentRequests,
						TargetBytesPerRequest: defaultTargetBytesPerRequest,
						Interval:              1 * time.Second,
					},
					SLO: testSLOcfg,
				},
			}, nil)

			// grpc
			srv := newMockStreamingServer[*tempopb.SearchTagValuesV2Response]("bar", nil)
			grpcReq := &tempopb.SearchTagValuesRequest{
				TagName: "foo",
			}
			err := f.streamingTagValuesV2(grpcReq, srv)
			require.Equal(t, tc.expectedErr, err)
		})
	}
}

func TestSearchTagsV2AccessesCache(t *testing.T) {
	meta := &backend.BlockMeta{
		StartTime:    time.Unix(15, 0),
		EndTime:      time.Unix(16, 0),
		Size_:        defaultTargetBytesPerRequest,
		TotalRecords: 1,
		BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000123"),
	}

	rdr := &mockReader{
		metas: []*backend.BlockMeta{meta},
	}

	expectedResp := &tempopb.SearchTagsV2Response{
		Scopes: []*tempopb.SearchTagsV2Scope{
			{
				Name: "resource",
				Tags: []string{"foo", "bar"},
			},
		},
	}

	// setup mock cache
	c := test.NewMockClient()
	p := test.NewMockProvider()
	err := p.AddCache(cache.RoleFrontendSearch, c)
	require.NoError(t, err)
	f := frontendWithSettings(t, &mockRoundTripper{
		responseFn: func() proto.Message {
			return expectedResp
		},
	}, rdr, nil, p)

	// setup query
	tenant := "foo"
	scope := "resource"
	hash := fnv1a.HashString64(scope)
	start := uint32(10)
	end := uint32(20)
	startTime := time.Unix(int64(start), 0)
	endTime := time.Unix(int64(end), 0)
	cacheKey := cacheKey(cacheKeyPrefixSearchTag, tenant, hash, startTime, endTime, meta, 0, 1)

	// confirm cache key coesn't exist
	_, bufs, _ := c.Fetch(context.Background(), []string{cacheKey})
	require.Equal(t, 0, len(bufs))

	// execute query
	path := fmt.Sprintf("/?start=%d&end=%d&scope=%s", start, end, scope) // encapsulates block above
	req := httptest.NewRequest("GET", path, nil)
	ctx := req.Context()
	ctx = user.InjectOrgID(ctx, tenant)
	req = req.WithContext(ctx)

	respWriter := httptest.NewRecorder()
	f.SearchTagsV2Handler.ServeHTTP(respWriter, req)

	resp := respWriter.Result()
	require.Equal(t, 200, resp.StatusCode)

	actualResp := &tempopb.SearchTagsV2Response{}
	bytesResp, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	err = jsonpb.Unmarshal(bytes.NewReader(bytesResp), actualResp)
	require.NoError(t, err)

	// confirm cache key exists and matches the response above
	_, bufs, _ = c.Fetch(context.Background(), []string{cacheKey})
	require.Equal(t, 1, len(bufs))

	actualCache := &tempopb.SearchTagsV2Response{}
	err = jsonpb.Unmarshal(bytes.NewReader(bufs[0]), actualCache)
	require.NoError(t, err)

	// zeroing these out b/c they are set by the sharder and won't be in cache
	require.Equal(t, expectedResp, actualCache)

	// now let's "poison" cache by writing different values directly and confirm
	// the sharder returns them

	overwriteResp := &tempopb.SearchTagsV2Response{
		Scopes: []*tempopb.SearchTagsV2Scope{
			{
				Name: "resource",
				Tags: []string{"blarg", "blerg"},
			},
		},
		Metrics: &tempopb.MetadataMetrics{},
	}
	overwriteString, err := (&jsonpb.Marshaler{}).MarshalToString(overwriteResp)
	require.NoError(t, err)

	c.Store(context.Background(), []string{cacheKey}, [][]byte{[]byte(overwriteString)})

	respWriter = httptest.NewRecorder()
	f.SearchTagsV2Handler.ServeHTTP(respWriter, req)

	resp = respWriter.Result()
	require.Equal(t, 200, resp.StatusCode)

	actualResp = &tempopb.SearchTagsV2Response{}
	bytesResp, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	err = jsonpb.Unmarshal(bytes.NewReader(bytesResp), actualResp)
	require.NoError(t, err)

	require.Equal(t, overwriteResp, actualResp)
}

func TestTagValuesCachedMetrics(t *testing.T) {
	// set up backend
	meta := &backend.BlockMeta{
		StartTime:    time.Unix(15, 0),
		EndTime:      time.Unix(16, 0),
		Size_:        defaultTargetBytesPerRequest,
		TotalRecords: 1,
		BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000123"),
	}
	rdr := &mockReader{
		metas: []*backend.BlockMeta{meta},
	}

	// set up cache
	c := test.NewMockClient()
	p := test.NewMockProvider()
	err := p.AddCache(cache.RoleFrontendSearch, c)
	require.NoError(t, err)

	// Set up a mock response with specific metrics that should be cleared when cached
	f := frontendWithSettings(t, &mockRoundTripper{
		responseFn: func() proto.Message {
			return &tempopb.SearchTagValuesV2Response{
				TagValues: []*tempopb.TagValue{
					{
						Type:  "keyword",
						Value: "frontend",
					},
					{
						Type:  "keyword",
						Value: "backend",
					},
					{
						Type:  "keyword",
						Value: "database",
					},
				},
				Metrics: &tempopb.MetadataMetrics{
					InspectedBytes: 1024,
				},
			}
		},
	}, rdr, nil, p)

	// setup query
	tenant := "foo"
	tagName := "service.name"
	hash := fnv1a.HashString64(tagName)
	start := uint32(10)
	end := uint32(20)
	startTime := time.Unix(int64(start), 0)
	endTime := time.Unix(int64(end), 0)
	cacheKey := cacheKey(cacheKeyPrefixSearchTagValues, tenant, hash, startTime, endTime, meta, 0, 1)

	// confirm cache key doesn't exist
	_, bufs, _ := c.Fetch(context.Background(), []string{cacheKey})
	require.Equal(t, 0, len(bufs))

	// execute query
	path := fmt.Sprintf("/api/v2/search/tag/%s/values?start=%d&end=%d", tagName, start, end)
	req := httptest.NewRequest("GET", path, nil)
	req = mux.SetURLVars(req, map[string]string{"tagName": tagName})
	ctx := req.Context()
	ctx = user.InjectOrgID(ctx, tenant)
	req = req.WithContext(ctx)

	respWriter := httptest.NewRecorder()
	f.SearchTagsValuesV2Handler.ServeHTTP(respWriter, req)
	resp := respWriter.Result()
	require.Equal(t, 200, resp.StatusCode)

	// parse response
	actualResp := &tempopb.SearchTagValuesV2Response{}
	bytesResp, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	err = jsonpb.Unmarshal(bytes.NewReader(bytesResp), actualResp)
	require.NoError(t, err)

	// verify metrics are collected
	require.Equal(t, uint64(1024), actualResp.Metrics.InspectedBytes)

	// execute query again
	respWriter = httptest.NewRecorder()
	f.SearchTagsValuesV2Handler.ServeHTTP(respWriter, req)
	resp = respWriter.Result()
	require.Equal(t, 200, resp.StatusCode)

	// parse cached response
	actualResp = &tempopb.SearchTagValuesV2Response{}
	bytesResp, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	err = jsonpb.Unmarshal(bytes.NewReader(bytesResp), actualResp)
	require.NoError(t, err)

	// verify metrics are 0 because the response was cached
	require.Equal(t, uint64(0), actualResp.Metrics.InspectedBytes)
}

func TestTagsCachedMetrics(t *testing.T) {
	// set up backend
	meta := &backend.BlockMeta{
		StartTime:    time.Unix(15, 0),
		EndTime:      time.Unix(16, 0),
		Size_:        defaultTargetBytesPerRequest,
		TotalRecords: 1,
		BlockID:      backend.MustParse("00000000-0000-0000-0000-000000000123"),
	}
	rdr := &mockReader{
		metas: []*backend.BlockMeta{meta},
	}

	// set up cache
	c := test.NewMockClient()
	p := test.NewMockProvider()
	err := p.AddCache(cache.RoleFrontendSearch, c)
	require.NoError(t, err)

	// Set up a mock response with specific metrics that should be cleared when cached
	f := frontendWithSettings(t, &mockRoundTripper{
		responseFn: func() proto.Message {
			return &tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
					{
						Name: "resource",
						Tags: []string{"service.name", "http.method", "http.status_code"},
					},
					{
						Name: "span",
						Tags: []string{"span.kind", "span.name", "span.status"},
					},
				},
				Metrics: &tempopb.MetadataMetrics{
					InspectedBytes: 2048,
				},
			}
		},
	}, rdr, nil, p)

	// setup query
	tenant := "foo"
	scope := "resource"
	hash := fnv1a.HashString64(scope)
	start := uint32(10)
	end := uint32(20)
	startTime := time.Unix(int64(start), 0)
	endTime := time.Unix(int64(end), 0)
	cacheKey := cacheKey(cacheKeyPrefixSearchTag, tenant, hash, startTime, endTime, meta, 0, 1)

	// confirm cache key doesn't exist
	_, bufs, _ := c.Fetch(context.Background(), []string{cacheKey})
	require.Equal(t, 0, len(bufs))

	// execute query
	path := fmt.Sprintf("/api/v2/search/tags?start=%d&end=%d&scope=%s", start, end, scope)
	req := httptest.NewRequest("GET", path, nil)
	ctx := req.Context()
	ctx = user.InjectOrgID(ctx, tenant)
	req = req.WithContext(ctx)

	respWriter := httptest.NewRecorder()
	f.SearchTagsV2Handler.ServeHTTP(respWriter, req)
	resp := respWriter.Result()
	require.Equal(t, 200, resp.StatusCode)

	// parse response
	actualResp := &tempopb.SearchTagsV2Response{}
	bytesResp, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	err = jsonpb.Unmarshal(bytes.NewReader(bytesResp), actualResp)
	require.NoError(t, err)

	// verify metrics are collected
	require.Equal(t, uint64(2048), actualResp.Metrics.InspectedBytes)

	// execute query again
	respWriter = httptest.NewRecorder()
	f.SearchTagsV2Handler.ServeHTTP(respWriter, req)
	resp = respWriter.Result()
	require.Equal(t, 200, resp.StatusCode)

	// parse cached response
	actualResp = &tempopb.SearchTagsV2Response{}
	bytesResp, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	err = jsonpb.Unmarshal(bytes.NewReader(bytesResp), actualResp)
	require.NoError(t, err)

	// verify metrics are 0 because the response was cached
	require.Equal(t, uint64(0), actualResp.Metrics.InspectedBytes)
}

func TestParseParams(t *testing.T) {
	tests := []struct {
		name                   string
		queryParams            map[string]string
		expectedScope          string
		expectedQ              string
		expectedDuration       uint32
		expectedLimit          uint32
		expectedValueThreshold uint32
	}{
		{
			name:                   "all params present",
			queryParams:            map[string]string{"start": "1723667082", "end": "1723839882", "scope": "resource", "q": "some_query"},
			expectedScope:          "resource",
			expectedQ:              "some_query",
			expectedDuration:       172800,
			expectedLimit:          0,
			expectedValueThreshold: 0,
		},
		{
			name:                   "missing start",
			queryParams:            map[string]string{"end": "1723839882", "scope": "resource"},
			expectedScope:          "resource",
			expectedQ:              "",
			expectedDuration:       0,
			expectedLimit:          0,
			expectedValueThreshold: 0,
		},
		{
			name:                   "missing end",
			queryParams:            map[string]string{"start": "1723667082", "scope": "resource"},
			expectedScope:          "resource",
			expectedQ:              "",
			expectedDuration:       0,
			expectedLimit:          0,
			expectedValueThreshold: 0,
		},
		{
			name:                   "missing scope",
			queryParams:            map[string]string{"start": "1723667082", "end": "1723839882"},
			expectedScope:          "",
			expectedQ:              "",
			expectedDuration:       172800,
			expectedLimit:          0,
			expectedValueThreshold: 0,
		},
		{
			name:                   "missing q",
			queryParams:            map[string]string{"start": "1723667082", "end": "1723839882", "scope": "resource"},
			expectedScope:          "resource",
			expectedQ:              "",
			expectedDuration:       172800,
			expectedLimit:          0,
			expectedValueThreshold: 0,
		},
		{
			name:                   "invalid start",
			queryParams:            map[string]string{"start": "invalid", "end": "1723839882", "scope": "resource"},
			expectedScope:          "resource",
			expectedQ:              "",
			expectedDuration:       0,
			expectedLimit:          0,
			expectedValueThreshold: 0,
		},
		{
			name:                   "invalid end",
			queryParams:            map[string]string{"start": "1723667082", "end": "invalid", "scope": "resource"},
			expectedScope:          "resource",
			expectedQ:              "",
			expectedDuration:       0,
			expectedLimit:          0,
			expectedValueThreshold: 0,
		},
		{
			name:                   "no params",
			queryParams:            map[string]string{},
			expectedScope:          "",
			expectedQ:              "",
			expectedDuration:       0,
			expectedLimit:          0,
			expectedValueThreshold: 0,
		},
		{
			name:                   "negative start and end",
			queryParams:            map[string]string{"start": "-1000", "end": "-2000", "scope": "negative_case"},
			expectedScope:          "negative_case",
			expectedQ:              "",
			expectedDuration:       0,
			expectedLimit:          0,
			expectedValueThreshold: 0,
		},
		{
			name:                   "end less than start",
			queryParams:            map[string]string{"start": "1723839882", "end": "1723667082", "scope": "resource"},
			expectedScope:          "resource",
			expectedQ:              "",
			expectedDuration:       0,
			expectedLimit:          0,
			expectedValueThreshold: 0,
		},
		{
			name:                   "start and end are the same",
			queryParams:            map[string]string{"start": "1723839882", "end": "1723839882", "scope": "zero_duration"},
			expectedScope:          "zero_duration",
			expectedQ:              "",
			expectedDuration:       0,
			expectedLimit:          0,
			expectedValueThreshold: 0,
		},
		{
			name:                   "invalid limit",
			queryParams:            map[string]string{"limit": "-1000"},
			expectedScope:          "",
			expectedQ:              "",
			expectedDuration:       0,
			expectedLimit:          0,
			expectedValueThreshold: 0,
		},
		{
			name:                   "invalid too large limit",
			queryParams:            map[string]string{"limit": "1000000000000000000"},
			expectedScope:          "",
			expectedQ:              "",
			expectedDuration:       0,
			expectedLimit:          0,
			expectedValueThreshold: 0,
		},
		{
			name:                   "valid limit",
			queryParams:            map[string]string{"limit": "100"},
			expectedScope:          "",
			expectedQ:              "",
			expectedDuration:       0,
			expectedLimit:          100,
			expectedValueThreshold: 0,
		},
		{
			name:                   "valid threshold",
			queryParams:            map[string]string{"maxStaleValues": "100"},
			expectedScope:          "",
			expectedQ:              "",
			expectedDuration:       0,
			expectedLimit:          0,
			expectedValueThreshold: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &url.URL{Path: "/my/test/path"}
			query := u.Query()
			for key, value := range tt.queryParams {
				query.Add(key, value)
			}
			u.RawQuery = query.Encode()
			req := &http.Request{URL: u}

			scope, q, duration, limit, staleValueThreshold := parseParams(req)

			require.Equal(t, tt.expectedScope, scope)
			require.Equal(t, tt.expectedQ, q)
			require.Equal(t, tt.expectedDuration, duration)
			require.Equal(t, tt.expectedLimit, limit)
			require.Equal(t, tt.expectedValueThreshold, staleValueThreshold)
		})
	}
}

func TestExtractTagName(t *testing.T) {
	// Define the base of our test cases table
	var testCases []struct {
		name     string
		urlPath  string
		pattern  *regexp.Regexp
		expected string
	}

	prefixes := []string{
		"/tempo",
		"/otherprefix",
		"", // No prefix
	}
	tagNames := []string{
		".X-Ab-TraceID",
		".__name__",
		".action",
		".app",
		".application_id",
		"span.name",
		"hello",
		"name",
		"$tag_name",
		"\u00E9:tag\\escaped_tag",
	}
	patterns := []struct {
		name   string
		regex  *regexp.Regexp
		suffix string
	}{
		{"WithoutV2", tagNameRegexV1, "/api/search/tag/"},
		{"WithV2", tagNameRegexV2, "/api/v2/search/tag/"},
	}

	// build test cases
	for _, prefix := range prefixes {
		for _, tagName := range tagNames {
			for _, pattern := range patterns {
				// Construct the full path
				fullPath := prefix + pattern.suffix + tagName + "/values"

				// Add the test case to the array
				testCases = append(testCases, struct {
					name     string
					urlPath  string
					pattern  *regexp.Regexp
					expected string
				}{
					name:     "Prefix: " + prefix + ", Tag: " + tagName + ", Pattern: " + pattern.name,
					urlPath:  fullPath,
					pattern:  pattern.regex,
					expected: tagName,
				})
			}
		}
	}

	// Additional edge cases
	edgeCases := []struct {
		name     string
		urlPath  string
		pattern  *regexp.Regexp
		expected string
	}{
		{"Missing tag name V1", "/api/search/tag//values", tagNameRegexV1, ""},
		{"Missing tag name V2", "/api/v2/search/tag//values", tagNameRegexV2, ""},
		{"Non-matching path V1", "/some/other/path/without/tag/values", tagNameRegexV1, ""},
		{"Non-matching path V2", "/different/path/without/tag/values", tagNameRegexV2, ""},
	}
	testCases = append(testCases, edgeCases...)

	// Run all test cases
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			extractedTagName := extractTagName(tt.urlPath, tt.pattern)
			require.Equal(t, tt.expected, extractedTagName)
		})
	}
}
