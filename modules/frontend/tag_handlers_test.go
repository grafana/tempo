package frontend

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/status"
	"github.com/google/uuid"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/pkg/cache"
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
	f.SearchHandler.ServeHTTP(httpResp, httpReq)
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
	f.SearchHandler.ServeHTTP(httpResp, httpReq)
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
	httpResp := httptest.NewRecorder()
	f.SearchHandler.ServeHTTP(httpResp, httpReq)
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
	httpResp := httptest.NewRecorder()
	f.SearchHandler.ServeHTTP(httpResp, httpReq)
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

	f.SearchHandler.ServeHTTP(httpResp, httpReq)
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
	require.Equal(t, status.Error(codes.Internal, "context canceled"), err)
}

func runnerTagValuesV2ClientCancelContext(t *testing.T, f *QueryFrontend) {
	// http
	httpReq := httptest.NewRequest("GET", "/api/v2/search/tag/foo/values", nil)
	httpResp := httptest.NewRecorder()

	ctx, cancel := context.WithCancel(httpReq.Context())
	ctx = user.InjectOrgID(ctx, "tenant")

	httpReq = httpReq.WithContext(ctx)
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	f.SearchHandler.ServeHTTP(httpResp, httpReq)
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
	require.Equal(t, status.Error(codes.Internal, "context canceled"), err)
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
				},
				SLO: testSLOcfg,
			},
			Metrics: MetricsConfig{
				Sharder: QueryRangeSharderConfig{
					ConcurrentRequests:    defaultConcurrentRequests,
					TargetBytesPerRequest: defaultTargetBytesPerRequest,
					Interval:              1 * time.Second,
				},
			},
		}, nil)

		httpReq := httptest.NewRequest("GET", "/api/search/tags?start=1&end=10000", nil)
		httpResp := httptest.NewRecorder()

		ctx := user.InjectOrgID(httpReq.Context(), "foo")
		httpReq = httpReq.WithContext(ctx)

		f.SearchHandler.ServeHTTP(httpResp, httpReq)
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
				},
				SLO: testSLOcfg,
			},
			Metrics: MetricsConfig{
				Sharder: QueryRangeSharderConfig{
					ConcurrentRequests:    defaultConcurrentRequests,
					TargetBytesPerRequest: defaultTargetBytesPerRequest,
					Interval:              1 * time.Second,
				},
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
				},
				SLO: testSLOcfg,
			},
			Metrics: MetricsConfig{
				Sharder: QueryRangeSharderConfig{
					ConcurrentRequests:    defaultConcurrentRequests,
					TargetBytesPerRequest: defaultTargetBytesPerRequest,
					Interval:              1 * time.Second,
				},
			},
		}, nil)

		httpReq := httptest.NewRequest("GET", "/api/v2/search/tags?start=1&end=10000", nil)
		httpResp := httptest.NewRecorder()

		ctx := user.InjectOrgID(httpReq.Context(), "foo")
		httpReq = httpReq.WithContext(ctx)

		f.SearchHandler.ServeHTTP(httpResp, httpReq)
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
				},
				SLO: testSLOcfg,
			},
			Metrics: MetricsConfig{
				Sharder: QueryRangeSharderConfig{
					ConcurrentRequests:    defaultConcurrentRequests,
					TargetBytesPerRequest: defaultTargetBytesPerRequest,
					Interval:              1 * time.Second,
				},
			},
		}, nil)

		// grpc
		srv := newMockStreamingServer[*tempopb.SearchTagValuesV2Response]("bar", nil)
		grpcReq := &tempopb.SearchTagValuesRequest{
			TagName: "foo",
		}
		err := f.streamingTagValuesV2(grpcReq, srv)
		require.Equal(t, tc.expectedErr, err)
	}
}

func TestSearchTagsV2AccessesCache(t *testing.T) {
	meta := &backend.BlockMeta{
		StartTime:    time.Unix(15, 0),
		EndTime:      time.Unix(16, 0),
		Size:         defaultTargetBytesPerRequest,
		TotalRecords: 1,
		BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000123"),
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
	c := cache.NewMockCache()
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
	cacheKey := cacheKey(cacheKeyPrefixSearchTag, tenant, hash, int64(start), int64(end), meta, 0, 1)

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
