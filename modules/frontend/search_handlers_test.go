package frontend

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/status"
	"github.com/google/uuid"
	"github.com/grafana/dskit/user"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/cache"
	"github.com/grafana/tempo/pkg/search"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
)

var _ http.RoundTripper = &mockRoundTripper{}

type mockRoundTripper struct {
	err           error
	statusCode    int
	statusMessage string
	once          sync.Once

	responseFn func() proto.Message
}

func (s *mockRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	// only return errors once, then do a good response to make sure that the combiner is handling the error correctly
	var err error
	var errResponse *http.Response
	s.once.Do(func() {
		err = s.err
		if s.statusCode != 0 {
			errResponse = &http.Response{
				StatusCode: s.statusCode,
				Body:       io.NopCloser(strings.NewReader(s.statusMessage)),
			}
		}
		s.err = nil
		s.statusCode = 0
	})

	time.Sleep(time.Duration(rand.Intn(500)+100) * time.Millisecond) // simulate some latency

	if err != nil {
		return nil, err
	}

	if errResponse != nil {
		return errResponse, nil
	}

	m := jsonpb.Marshaler{}
	str, err := m.MarshalToString(s.responseFn())
	if err != nil {
		panic(err)
	}

	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader([]byte(str))),
	}, nil
}

type mockGRPCStreaming[T proto.Message] struct {
	lastResponse atomic.Pointer[T]
	responses    atomic.Int32
	ctx          context.Context
	cb           func(int, T)
}

func (m *mockGRPCStreaming[T]) Send(r T) error {
	m.lastResponse.Store(&r)
	m.responses.Inc()
	if m.cb != nil {
		m.cb(int(m.responses.Load()), r)
	}
	return nil
}

func (m *mockGRPCStreaming[T]) Context() context.Context     { return m.ctx }
func (m *mockGRPCStreaming[T]) SendHeader(metadata.MD) error { return nil }
func (m *mockGRPCStreaming[T]) SetHeader(metadata.MD) error  { return nil }
func (m *mockGRPCStreaming[T]) SendMsg(interface{}) error    { return nil }
func (m *mockGRPCStreaming[T]) RecvMsg(interface{}) error    { return nil }
func (m *mockGRPCStreaming[T]) SetTrailer(metadata.MD)       {}

func newMockStreamingServer[T proto.Message](orgID string, cb func(int, T)) *mockGRPCStreaming[T] {
	ctx := context.Background()
	if orgID != "" {
		ctx = user.InjectOrgID(ctx, orgID)
	}
	return &mockGRPCStreaming[T]{
		ctx: ctx,
		cb:  cb,
	}
}

// these are integration tests against the search http and streaming pipelines. they could be extended to handle all
// endpoints as we migrate them to the new pipelines and have common expectations on behaviors.
// todo: build a test harness that extends these to all endpoints
func TestFrontendSearch(t *testing.T) {
	allRunners := []func(t *testing.T, f *QueryFrontend){
		runnerBadRequestOnOrgID,
		runnerClientCancelContext,
		runnerRequests,
	}

	for _, runner := range allRunners {
		f := frontendWithSettings(t, nil, nil, nil, nil)
		runner(t, f)
	}
}

func runnerBadRequestOnOrgID(t *testing.T, f *QueryFrontend) {
	// http
	httpReq := httptest.NewRequest("GET", "/api/search", nil)
	httpResp := httptest.NewRecorder()
	f.SearchHandler.ServeHTTP(httpResp, httpReq)
	require.Equal(t, "no org id", httpResp.Body.String())
	require.Equal(t, http.StatusBadRequest, httpResp.Code)

	// grpc
	grpcReq := &tempopb.SearchRequest{}
	err := f.streamingSearch(grpcReq, newMockStreamingServer[*tempopb.SearchResponse]("", nil))
	require.Equal(t, status.Error(codes.InvalidArgument, "no org id"), err)
}

func runnerRequests(t *testing.T, f *QueryFrontend) {
	tcs := []struct {
		name    string
		tenant  string
		request *tempopb.SearchRequest
		// http+grpc
		expectedResponse *tempopb.SearchResponse
		// http
		expectedStatusCode    int
		expectedStatusMessage string
		// grpc
		expectedErr error
	}{
		{
			name: "access 2 blocks x 2 jobs = 4",
			request: &tempopb.SearchRequest{
				Query: "{resource.service.name = `test`}",
				Start: 1,
				End:   100000,
				Limit: 10,
			},
			expectedStatusCode: 200,
			expectedResponse: &tempopb.SearchResponse{
				Traces: []*tempopb.TraceSearchMetadata{{
					TraceID:         "1",
					RootServiceName: search.RootSpanNotYetReceivedText,
				}},
				Metrics: &tempopb.SearchMetrics{
					InspectedTraces: 4,
					InspectedBytes:  4,
					TotalBlocks:     2,
					TotalJobs:       4,
					TotalBlockBytes: 4 * defaultTargetBytesPerRequest,
					CompletedJobs:   4,
				},
			},
		},
		{
			name: "bad request",
			request: &tempopb.SearchRequest{
				Query: "foo",
				Start: 1,
				End:   100000,
				Limit: 10,
			},
			expectedStatusCode:    400,
			expectedStatusMessage: "invalid TraceQL query: parse error at line 1, col 1: syntax error: unexpected IDENTIFIER",
			expectedErr:           status.Error(codes.InvalidArgument, "invalid TraceQL query: parse error at line 1, col 1: syntax error: unexpected IDENTIFIER"),
		},
		{
			name:   "multitenant - 4 jobs x 2 tenants = 8",
			tenant: "tenant-1|tenant-2",
			request: &tempopb.SearchRequest{
				Query: "{resource.service.name = `test`}",
				Start: 1,
				End:   100000,
				Limit: 10,
			},
			expectedStatusCode: 200,
			expectedResponse: &tempopb.SearchResponse{
				Traces: []*tempopb.TraceSearchMetadata{{
					TraceID:         "1",
					RootServiceName: search.RootSpanNotYetReceivedText,
				}},
				Metrics: &tempopb.SearchMetrics{
					InspectedTraces: 8,
					InspectedBytes:  8,
					TotalBlocks:     4,
					TotalJobs:       8,
					TotalBlockBytes: 8 * defaultTargetBytesPerRequest,
					CompletedJobs:   8,
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			actualResp := &tempopb.SearchResponse{}

			tenant := tc.tenant
			if tenant == "" {
				tenant = "single-tenant"
			}

			httpReq := httptest.NewRequest("GET", "/api/search", nil)
			httpReq, err := api.BuildSearchRequest(httpReq, tc.request)
			require.NoError(t, err)
			ctx := user.InjectOrgID(httpReq.Context(), tenant)
			httpReq = httpReq.WithContext(ctx)

			// http
			httpResp := httptest.NewRecorder()
			f.SearchHandler.ServeHTTP(httpResp, httpReq)

			if tc.expectedStatusMessage != "" {
				require.Equal(t, tc.expectedStatusMessage, httpResp.Body.String())
			}
			if tc.expectedResponse != nil {
				err := jsonpb.Unmarshal(httpResp.Body, actualResp)
				require.NoError(t, err)
				require.Equal(t, tc.expectedResponse, actualResp)
			}
			require.Equal(t, tc.expectedStatusCode, httpResp.Code)

			// grpc
			actualResp = nil
			err = f.streamingSearch(tc.request, newMockStreamingServer(tenant, func(i int, sr *tempopb.SearchResponse) {
				actualResp = sr
			}))

			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedResponse, actualResp)
		})
	}
}

func runnerClientCancelContext(t *testing.T, f *QueryFrontend) {
	// http
	httpReq := httptest.NewRequest("GET", "/api/search", nil)
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
	srv := newMockStreamingServer[*tempopb.SearchResponse]("tenant", nil)
	srv.ctx, cancel = context.WithCancel(srv.ctx)
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	grpcReq := &tempopb.SearchRequest{}
	err := f.streamingSearch(grpcReq, srv)
	require.Equal(t, status.Error(codes.Internal, "context canceled"), err)
}

func TestSearchLimitHonored(t *testing.T) {
	f := frontendWithSettings(t, &mockRoundTripper{
		responseFn: func() proto.Message {
			return &tempopb.SearchResponse{
				Traces: []*tempopb.TraceSearchMetadata{
					{
						TraceID: util.TraceIDToHexString(test.ValidTraceID(nil)),
					},
				},
				Metrics: &tempopb.SearchMetrics{
					InspectedTraces: 1,
					InspectedBytes:  1,
				},
			}
		},
	}, nil, &Config{
		MultiTenantQueriesEnabled: true,
		TraceByID: TraceByIDConfig{
			QueryShards: minQueryShards,
			SLO:         testSLOcfg,
		},
		Search: SearchConfig{
			Sharder: SearchSharderConfig{
				ConcurrentRequests:    defaultConcurrentRequests,
				TargetBytesPerRequest: defaultTargetBytesPerRequest,
				DefaultLimit:          10,
				MaxLimit:              15,
			},
			SLO: testSLOcfg,
		},
	}, nil)

	tcs := []struct {
		name           string
		request        *tempopb.SearchRequest
		expectedTraces int
		badRequest     bool
	}{
		{
			name: "default limit is 10",
			request: &tempopb.SearchRequest{
				Query: "{resource.service.name = `test`}",
				Start: 1,
				End:   100000,
			},
			expectedTraces: 10,
		},
		{
			name: "limit in query",
			request: &tempopb.SearchRequest{
				Query: "{resource.service.name = `test`}",
				Start: 1,
				End:   100000,
				Limit: 7,
			},
			expectedTraces: 7,
		},
		{
			name: "bad request due to limit",
			request: &tempopb.SearchRequest{
				Query: "{resource.service.name = `test`}",
				Start: 1,
				End:   100000,
				Limit: 20,
			},
			badRequest: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// each tenant returns 4 blocks, so 24 the pipeline will try to return 24 total
			tenant := "1|2|3|4|5|6"

			// due to the blocks we will have 4 trace ids normally
			httpReq := httptest.NewRequest("GET", "/api/search", nil)
			httpReq, err := api.BuildSearchRequest(httpReq, tc.request)
			require.NoError(t, err)

			ctx := user.InjectOrgID(httpReq.Context(), tenant)
			httpReq = httpReq.WithContext(ctx)

			httpResp := httptest.NewRecorder()

			f.SearchHandler.ServeHTTP(httpResp, httpReq)
			if tc.badRequest {
				require.Equal(t, 400, httpResp.Code)
			} else {
				require.Equal(t, 200, httpResp.Code)

				actualResp := &tempopb.SearchResponse{}
				err = jsonpb.Unmarshal(httpResp.Body, actualResp)
				require.NoError(t, err)
				require.Len(t, actualResp.Traces, tc.expectedTraces)
			}

			// grpc
			var actualResp *tempopb.SearchResponse
			err = f.streamingSearch(tc.request, newMockStreamingServer(tenant, func(i int, sr *tempopb.SearchResponse) {
				actualResp = sr
			}))
			if tc.badRequest {
				require.Equal(t, status.Error(codes.InvalidArgument, "adjust limit: limit 20 exceeds max limit 15"), err)
			} else {
				require.NoError(t, err)
				require.Len(t, actualResp.Traces, tc.expectedTraces)
			}
		})
	}
}

func TestSearchFailurePropagatesFromQueriers(t *testing.T) {
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
				return &tempopb.SearchResponse{
					Traces:  []*tempopb.TraceSearchMetadata{},
					Metrics: &tempopb.SearchMetrics{},
				}
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
		}, nil)

		httpReq := httptest.NewRequest("GET", "/api/search?start=1&end=10000", nil)
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
		}, nil)

		// grpc
		srv := newMockStreamingServer[*tempopb.SearchResponse]("bar", nil)
		grpcReq := &tempopb.SearchRequest{}
		err := f.streamingSearch(grpcReq, srv)
		require.Equal(t, tc.expectedErr, err)
	}
}

func TestSearchAccessesCache(t *testing.T) {
	tenant := "foo"
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

	// setup mock cache
	c := cache.NewMockCache()
	p := test.NewMockProvider()
	err := p.AddCache(cache.RoleFrontendSearch, c)
	require.NoError(t, err)
	f := frontendWithSettings(t, nil, rdr, nil, p)

	// setup query
	query := "{}"
	hash := hashForSearchRequest(&tempopb.SearchRequest{Query: query, Limit: 3, SpansPerSpanSet: 2})
	start := uint32(10)
	end := uint32(20)
	cacheKey := searchJobCacheKey(tenant, hash, int64(start), int64(end), meta, 0, 1)

	// confirm cache key coesn't exist
	_, bufs, _ := c.Fetch(context.Background(), []string{cacheKey})
	require.Equal(t, 0, len(bufs))

	// execute query
	path := fmt.Sprintf("/?start=%d&end=%d&q=%s&limit=3&spss=2", start, end, query) // encapsulates block above
	req := httptest.NewRequest("GET", path, nil)
	ctx := req.Context()
	ctx = user.InjectOrgID(ctx, tenant)
	req = req.WithContext(ctx)

	respWriter := httptest.NewRecorder()
	f.SearchHandler.ServeHTTP(respWriter, req)

	resp := respWriter.Result()
	require.Equal(t, 200, resp.StatusCode)

	actualResp := &tempopb.SearchResponse{}
	bytesResp, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	err = jsonpb.Unmarshal(bytes.NewReader(bytesResp), actualResp)
	require.NoError(t, err)

	// confirm cache key exists and matches the response above
	_, bufs, _ = c.Fetch(context.Background(), []string{cacheKey})
	require.Equal(t, 1, len(bufs))

	actualCache := &tempopb.SearchResponse{}
	err = jsonpb.Unmarshal(bytes.NewReader(bufs[0]), actualCache)
	require.NoError(t, err)

	// zeroing these out b/c they are set by the sharder and won't be in cache
	cacheResponsesEqual(t, actualCache, actualResp)

	// now let's "poison" cache by writing different values directly and confirm
	// the sharder returns them
	overwriteResp := &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{{
			TraceID:         util.TraceIDToHexString(test.ValidTraceID(nil)),
			RootServiceName: "test2",
			RootTraceName:   "bar2",
		}},
		Metrics: &tempopb.SearchMetrics{
			InspectedBytes: 11,
		},
	}
	overwriteString, err := (&jsonpb.Marshaler{}).MarshalToString(overwriteResp)
	require.NoError(t, err)

	c.Store(context.Background(), []string{cacheKey}, [][]byte{[]byte(overwriteString)})

	respWriter = httptest.NewRecorder()
	f.SearchHandler.ServeHTTP(respWriter, req)

	resp = respWriter.Result()
	require.Equal(t, 200, resp.StatusCode)

	actualResp = &tempopb.SearchResponse{}
	bytesResp, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	err = jsonpb.Unmarshal(bytes.NewReader(bytesResp), actualResp)
	require.NoError(t, err)

	cacheResponsesEqual(t, overwriteResp, actualResp)
}

func cacheResponsesEqual(t *testing.T, cacheResponse *tempopb.SearchResponse, pipelineResp *tempopb.SearchResponse) {
	// zeroing these out b/c they are set by the sharder and won't be in cache
	pipelineResp.Metrics.TotalJobs = 0
	pipelineResp.Metrics.CompletedJobs = 0
	pipelineResp.Metrics.TotalBlockBytes = 0
	pipelineResp.Metrics.TotalBlocks = 0

	// cache won't have "root span not yet received" text
	for _, tr := range pipelineResp.Traces {
		if tr.RootServiceName == search.RootSpanNotYetReceivedText {
			tr.RootServiceName = ""
		}
	}

	require.Equal(t, pipelineResp, cacheResponse)
}

// frontendWithSettings returns a new frontend with the given settings. any nil options
// are given "happy path" defaults
func frontendWithSettings(t *testing.T, next http.RoundTripper, rdr tempodb.Reader, cfg *Config, cacheProvider cache.Provider) *QueryFrontend {
	if next == nil {
		next = &mockRoundTripper{
			responseFn: func() proto.Message {
				return &tempopb.SearchResponse{
					Traces: []*tempopb.TraceSearchMetadata{
						{
							TraceID: "1",
						},
					},
					Metrics: &tempopb.SearchMetrics{
						InspectedTraces: 1,
						InspectedBytes:  1,
					},
				}
			},
		}
	}
	if rdr == nil {
		rdr = &mockReader{
			metas: []*backend.BlockMeta{ // one block with 2 records that are each the target bytes per request will force 2 sub queries
				{
					StartTime:    time.Unix(1100, 0),
					EndTime:      time.Unix(1200, 0),
					Size:         defaultTargetBytesPerRequest * 2,
					TotalRecords: 2,
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				},
				{
					StartTime:    time.Unix(1100, 0),
					EndTime:      time.Unix(1200, 0),
					Size:         defaultTargetBytesPerRequest * 2,
					TotalRecords: 2,
					BlockID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},
		}
	}
	if cfg == nil {
		cfg = &Config{
			MultiTenantQueriesEnabled: true,
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
		}
	}

	o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	f, err := New(*cfg, next, o, rdr, cacheProvider, "", log.NewNopLogger(), nil)
	require.NoError(t, err)

	return f
}
