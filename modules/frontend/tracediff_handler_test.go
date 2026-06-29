package frontend

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/gogo/protobuf/proto"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	tempotest "github.com/grafana/tempo/pkg/util/test"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestBuildTraceDiffTraceByIDRequest(t *testing.T) {
	start := int64(100)
	end := int64(200)

	req := buildTraceDiffTraceByIDRequest(context.Background(), "", api.TraceDiffTraceRequest{
		TraceID: "abc123",
		Start:   &start,
		End:     &end,
	}, nil)

	require.Equal(t, http.MethodGet, req.Method)
	require.Equal(t, "/api/v2/traces/abc123", req.URL.Path)
	require.Equal(t, "100", req.URL.Query().Get("start"))
	require.Equal(t, "200", req.URL.Query().Get("end"))
	require.Equal(t, api.HeaderAcceptProtobuf, req.Header.Get(api.HeaderAccept))

	_, err := api.ParseTraceID(req)
	require.NoError(t, err)

	_, _, queryMode, startTime, endTime, err := api.ParseTraceByIDRequest(req)
	require.NoError(t, err)
	require.Equal(t, api.QueryModeAll, queryMode)
	require.Equal(t, time.Unix(start, 0), startTime)
	require.Equal(t, time.Unix(end, 0), endTime)
}

func TestTraceDiffHandlerFetchTraceForDiff(t *testing.T) {
	traceID := "abc123"
	trace := tempotest.MakeTrace(1, []byte{0xab, 0xc1, 0x23})
	traceByIDResp := &tempopb.TraceByIDResponse{
		Trace: trace,
		Metrics: &tempopb.TraceByIDMetrics{
			InspectedBytes: 123,
		},
	}
	respBytes, err := proto.Marshal(traceByIDResp)
	require.NoError(t, err)

	var gotReq *http.Request
	tracePipeline := pipeline.AsyncRoundTripperFunc[combiner.PipelineResponse](func(req pipeline.Request) (pipeline.Responses[combiner.PipelineResponse], error) {
		gotReq = req.HTTPRequest()
		return pipeline.NewHTTPToAsyncResponse(&http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				api.HeaderContentType: []string{api.HeaderAcceptProtobuf},
			},
			Body: io.NopCloser(bytes.NewReader(respBytes)),
		}), nil
	})

	o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.NewRegistry())
	require.NoError(t, err)

	actual, err := fetchTraceForDiff(context.Background(), "test-tenant", api.TraceDiffTraceRequest{TraceID: traceID}, nil, "/tempo", tracePipeline, o, combiner.NewTypedTraceByIDV2, log.NewNopLogger(), nil)
	require.NoError(t, err)
	require.NotNil(t, actual)
	require.True(t, proto.Equal(trace, actual.Trace))
	require.Equal(t, uint64(123), actual.Metrics.GetInspectedBytes())
	require.NotNil(t, gotReq)
	require.Equal(t, "/tempo/api/v2/traces/abc123", gotReq.URL.Path)
	require.Equal(t, api.HeaderAcceptProtobuf, gotReq.Header.Get(api.HeaderAccept))
}

func traceDiffTestPipeline(t *testing.T, traces map[string]*tempopb.TraceByIDResponse) pipeline.AsyncRoundTripper[combiner.PipelineResponse] {
	t.Helper()
	return pipeline.AsyncRoundTripperFunc[combiner.PipelineResponse](func(req pipeline.Request) (pipeline.Responses[combiner.PipelineResponse], error) {
		tracePath := req.HTTPRequest().URL.Path
		traceID := tracePath[strings.LastIndex(tracePath, "/")+1:]
		traceResp, ok := traces[traceID]
		if !ok {
			return pipeline.NewHTTPToAsyncResponse(&http.Response{
				StatusCode: http.StatusNotFound,
				Status:     http.StatusText(http.StatusNotFound),
				Body:       io.NopCloser(strings.NewReader("trace not found")),
			}), nil
		}

		respBytes, err := proto.Marshal(traceResp)
		require.NoError(t, err)
		return pipeline.NewHTTPToAsyncResponse(&http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				api.HeaderContentType: []string{api.HeaderAcceptProtobuf},
			},
			Body: io.NopCloser(bytes.NewReader(respBytes)),
		}), nil
	})
}

type traceDiffDataAccessController struct {
	mu       sync.Mutex
	redactor combiner.TraceRedactor
	err      error
	calls    int
}

func (c *traceDiffDataAccessController) HandleHTTPSearchReq(_ *http.Request) error { return nil }

func (c *traceDiffDataAccessController) HandleHTTPTagsReq(_ *http.Request) error { return nil }

func (c *traceDiffDataAccessController) HandleHTTPTagsV2Req(_ *http.Request) error { return nil }

func (c *traceDiffDataAccessController) HandleHTTPTagValuesReq(_ *http.Request) error { return nil }

func (c *traceDiffDataAccessController) HandleHTTPTagValuesV2Req(_ *http.Request) error { return nil }

func (c *traceDiffDataAccessController) HandleHTTPQueryRangeReq(_ *http.Request) error { return nil }

func (c *traceDiffDataAccessController) HandleHTTPQueryInstantReq(_ *http.Request) error { return nil }

func (c *traceDiffDataAccessController) HandleHTTPTraceByIDReq(_ *http.Request) (combiner.TraceRedactor, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	return c.redactor, c.err
}

func (c *traceDiffDataAccessController) HandleGRPCSearchReq(_ context.Context, _ *tempopb.SearchRequest) error {
	return nil
}

func (c *traceDiffDataAccessController) HandleGRPCTagsReq(_ context.Context, _ *tempopb.SearchTagsRequest) error {
	return nil
}

func (c *traceDiffDataAccessController) HandleGRPCTagsV2Req(_ context.Context, _ *tempopb.SearchTagsRequest) error {
	return nil
}

func (c *traceDiffDataAccessController) HandleGRPCTagValuesReq(_ context.Context, _ *tempopb.SearchTagValuesRequest) error {
	return nil
}

func (c *traceDiffDataAccessController) HandleGRPCTagValuesV2Req(_ context.Context, _ *tempopb.SearchTagValuesRequest) error {
	return nil
}

func (c *traceDiffDataAccessController) HandleGRPCQueryRangeReq(_ context.Context, _ *tempopb.QueryRangeRequest) error {
	return nil
}

func (c *traceDiffDataAccessController) HandleGRPCQueryInstantReq(_ context.Context, _ *tempopb.QueryInstantRequest) error {
	return nil
}

func (c *traceDiffDataAccessController) callCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

type traceDiffHidingRedactor struct{}

func (traceDiffHidingRedactor) RedactTraceAttributes(_ *tempopb.Trace) error {
	return combiner.ErrTraceHidden
}

func TestTraceDiffHandlerReturnsDiff(t *testing.T) {
	tracePipeline := traceDiffTestPipeline(t, map[string]*tempopb.TraceByIDResponse{
		"abc123": {Trace: tempotest.MakeTrace(1, []byte{0xab, 0xc1, 0x23})},
		"def456": {Trace: tempotest.MakeTrace(1, []byte{0xde, 0xf4, 0x56})},
	})
	o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	handler := newHandler(nil, newTraceDiffHandler(Config{}, "", tracePipeline, o, combiner.NewTypedTraceByIDV2, nil, log.NewNopLogger(), nil), log.NewNopLogger())

	req := httptest.NewRequest(http.MethodPost, "/api/v2/traces/diff", strings.NewReader(`{"base":{"traceId":"abc123"},"compare":{"traceId":"def456"}}`))
	req = req.WithContext(user.InjectOrgID(req.Context(), "test-tenant"))
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, api.HeaderAcceptJSON, resp.Header().Get(api.HeaderContentType))
	require.Contains(t, resp.Body.String(), `"version":"trace-patch-v0"`)
}

func TestTraceDiffHandlerAppliesTraceRedactor(t *testing.T) {
	tracePipeline := traceDiffTestPipeline(t, map[string]*tempopb.TraceByIDResponse{
		"abc123": {Trace: tempotest.MakeTrace(1, []byte{0xab, 0xc1, 0x23})},
		"def456": {Trace: tempotest.MakeTrace(1, []byte{0xde, 0xf4, 0x56})},
	})
	o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	dataAccessController := &traceDiffDataAccessController{redactor: traceDiffHidingRedactor{}}
	handler := newHandler(nil, newTraceDiffHandler(Config{}, "", tracePipeline, o, combiner.NewTypedTraceByIDV2, nil, log.NewNopLogger(), dataAccessController), log.NewNopLogger())

	req := httptest.NewRequest(http.MethodPost, "/api/v2/traces/diff", strings.NewReader(`{"base":{"traceId":"abc123"},"compare":{"traceId":"def456"}}`))
	req = req.WithContext(user.InjectOrgID(req.Context(), "test-tenant"))
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	require.Equal(t, http.StatusNotFound, resp.Code)
	require.Equal(t, http.StatusText(http.StatusNotFound), resp.Body.String())
	require.Positive(t, dataAccessController.callCount())
}

func TestTraceDiffHandlerMapsWrappedTraceFetchStatus(t *testing.T) {
	tracePipeline := traceDiffTestPipeline(t, map[string]*tempopb.TraceByIDResponse{
		"abc123": {Trace: tempotest.MakeTrace(1, []byte{0xab, 0xc1, 0x23})},
	})
	o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	handler := newHandler(nil, newTraceDiffHandler(Config{}, "", tracePipeline, o, combiner.NewTypedTraceByIDV2, nil, log.NewNopLogger(), nil), log.NewNopLogger())

	req := httptest.NewRequest(http.MethodPost, "/api/v2/traces/diff", strings.NewReader(`{"base":{"traceId":"abc123"},"compare":{"traceId":"def456"}}`))
	req = req.WithContext(user.InjectOrgID(req.Context(), "test-tenant"))
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	require.Equal(t, http.StatusNotFound, resp.Code)
	require.Equal(t, http.StatusText(http.StatusNotFound), resp.Body.String())
}

func TestTraceDiffHandlerDataAccessErrorReturnsBadRequest(t *testing.T) {
	tracePipeline := traceDiffTestPipeline(t, map[string]*tempopb.TraceByIDResponse{
		"abc123": {Trace: tempotest.MakeTrace(1, []byte{0xab, 0xc1, 0x23})},
		"def456": {Trace: tempotest.MakeTrace(1, []byte{0xde, 0xf4, 0x56})},
	})
	o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	dataAccessController := &traceDiffDataAccessController{err: errors.New("policy rejected")}
	handler := newHandler(nil, newTraceDiffHandler(Config{}, "", tracePipeline, o, combiner.NewTypedTraceByIDV2, nil, log.NewNopLogger(), dataAccessController), log.NewNopLogger())

	req := httptest.NewRequest(http.MethodPost, "/api/v2/traces/diff", strings.NewReader(`{"base":{"traceId":"abc123"},"compare":{"traceId":"def456"}}`))
	req = req.WithContext(user.InjectOrgID(req.Context(), "test-tenant"))
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
	require.Contains(t, resp.Body.String(), "policy rejected")
	require.Positive(t, dataAccessController.callCount())
}

func TestTraceDiffHandlerRejectsPartialTraces(t *testing.T) {
	tracePipeline := traceDiffTestPipeline(t, map[string]*tempopb.TraceByIDResponse{
		"abc123": {Trace: tempotest.MakeTrace(1, []byte{0xab, 0xc1, 0x23})},
		"def456": {
			Trace:  tempotest.MakeTrace(1, []byte{0xde, 0xf4, 0x56}),
			Status: tempopb.PartialStatus_PARTIAL,
		},
	})
	o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	handler := newHandler(nil, newTraceDiffHandler(Config{}, "", tracePipeline, o, combiner.NewTypedTraceByIDV2, nil, log.NewNopLogger(), nil), log.NewNopLogger())

	req := httptest.NewRequest(http.MethodPost, "/api/v2/traces/diff", strings.NewReader(`{"base":{"traceId":"abc123"},"compare":{"traceId":"def456"}}`))
	req = req.WithContext(user.InjectOrgID(req.Context(), "test-tenant"))
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	require.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	require.Contains(t, resp.Body.String(), errTraceDiffPartialTrace.Error())
}

func TestTraceDiffHandlerSkeleton(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		body       string
		statusCode int
	}{
		{
			name:       "invalid post returns bad request",
			method:     http.MethodPost,
			body:       `{"base":{"traceId":"abc123"}}`,
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "get returns method not allowed",
			method:     http.MethodGet,
			statusCode: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newHandler(nil, newTraceDiffHandler(Config{}, "", nil, nil, nil, nil, log.NewNopLogger(), nil), log.NewNopLogger())
			req := httptest.NewRequest(tt.method, "/api/v2/traces/diff", strings.NewReader(tt.body))
			req = req.WithContext(user.InjectOrgID(req.Context(), "test-tenant"))
			resp := httptest.NewRecorder()

			handler.ServeHTTP(resp, req)

			require.Equal(t, tt.statusCode, resp.Code)
		})
	}
}
