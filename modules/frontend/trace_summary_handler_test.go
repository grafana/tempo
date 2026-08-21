package frontend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/mux"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/model/tracesummary"
	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resourcev1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func traceSummaryTestTrace() *tempopb.Trace {
	traceID := []byte{0xab, 0xc1, 0x23}
	rootSpanID := []byte{0, 0, 0, 0, 0, 0, 0, 1}
	childSpanID := []byte{0, 0, 0, 0, 0, 0, 0, 2}
	return &tempopb.Trace{
		ResourceSpans: []*tracev1.ResourceSpans{
			{
				Resource: &resourcev1.Resource{
					Attributes: []*commonv1.KeyValue{
						{Key: "service.name", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "checkout"}}},
					},
				},
				ScopeSpans: []*tracev1.ScopeSpans{
					{
						Spans: []*tracev1.Span{
							{
								TraceId:           traceID,
								SpanId:            rootSpanID,
								Name:              "root-op",
								Kind:              tracev1.Span_SPAN_KIND_SERVER,
								StartTimeUnixNano: 0,
								EndTimeUnixNano:   100,
								Status:            &tracev1.Status{Code: tracev1.Status_STATUS_CODE_OK},
							},
							{
								TraceId:           traceID,
								SpanId:            childSpanID,
								ParentSpanId:      rootSpanID,
								Name:              "child-op",
								Kind:              tracev1.Span_SPAN_KIND_CLIENT,
								StartTimeUnixNano: 10,
								EndTimeUnixNano:   50,
								Status:            &tracev1.Status{Code: tracev1.Status_STATUS_CODE_OK},
							},
						},
					},
				},
			},
		},
	}
}

// zipkinDualIDTestTrace builds a trace with the zipkin client/server
// duplicate-span-ID pattern: "call" (CLIENT) and "handle" (SERVER) share a
// span ID, and "do-work" is the true child of the SERVER half. This mirrors
// what a raw, un-deduped Zipkin import can produce before it reaches the
// combiner's deduper.
func zipkinDualIDTestTrace() *tempopb.Trace {
	traceID := []byte{0xab, 0xc1, 0x23}
	rootSpanID := []byte{0, 0, 0, 0, 0, 0, 0, 1}
	sharedSpanID := []byte{0, 0, 0, 0, 0, 0, 0, 2}
	grandchildSpanID := []byte{0, 0, 0, 0, 0, 0, 0, 3}
	return &tempopb.Trace{
		ResourceSpans: []*tracev1.ResourceSpans{
			{
				Resource: &resourcev1.Resource{
					Attributes: []*commonv1.KeyValue{
						{Key: "service.name", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "root-svc"}}},
					},
				},
				ScopeSpans: []*tracev1.ScopeSpans{{Spans: []*tracev1.Span{
					{
						TraceId:           traceID,
						SpanId:            rootSpanID,
						Name:              "root-op",
						Kind:              tracev1.Span_SPAN_KIND_SERVER,
						StartTimeUnixNano: 0,
						EndTimeUnixNano:   120,
						Status:            &tracev1.Status{Code: tracev1.Status_STATUS_CODE_OK},
					},
				}}},
			},
			{
				Resource: &resourcev1.Resource{
					Attributes: []*commonv1.KeyValue{
						{Key: "service.name", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "caller"}}},
					},
				},
				ScopeSpans: []*tracev1.ScopeSpans{{Spans: []*tracev1.Span{
					{
						TraceId:           traceID,
						SpanId:            sharedSpanID,
						ParentSpanId:      rootSpanID,
						Name:              "call",
						Kind:              tracev1.Span_SPAN_KIND_CLIENT,
						StartTimeUnixNano: 10,
						EndTimeUnixNano:   90,
						Status:            &tracev1.Status{Code: tracev1.Status_STATUS_CODE_OK},
					},
				}}},
			},
			{
				Resource: &resourcev1.Resource{
					Attributes: []*commonv1.KeyValue{
						{Key: "service.name", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "callee"}}},
					},
				},
				ScopeSpans: []*tracev1.ScopeSpans{{Spans: []*tracev1.Span{
					{
						TraceId: traceID,
						// Shares sharedSpanID with the CLIENT-kind "call" span above,
						// as produced by a raw zipkin import.
						SpanId:            sharedSpanID,
						ParentSpanId:      rootSpanID,
						Name:              "handle",
						Kind:              tracev1.Span_SPAN_KIND_SERVER,
						StartTimeUnixNano: 15,
						EndTimeUnixNano:   95,
						Status:            &tracev1.Status{Code: tracev1.Status_STATUS_CODE_OK},
					},
				}}},
			},
			{
				Resource: &resourcev1.Resource{
					Attributes: []*commonv1.KeyValue{
						{Key: "service.name", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "downstream"}}},
					},
				},
				ScopeSpans: []*tracev1.ScopeSpans{{Spans: []*tracev1.Span{
					{
						TraceId:           traceID,
						SpanId:            grandchildSpanID,
						ParentSpanId:      sharedSpanID,
						Name:              "do-work",
						Kind:              tracev1.Span_SPAN_KIND_INTERNAL,
						StartTimeUnixNano: 20,
						EndTimeUnixNano:   80,
						Status:            &tracev1.Status{Code: tracev1.Status_STATUS_CODE_OK},
					},
				}}},
			},
		},
	}
}

// TestTraceSummaryHandler_ZipkinDualIDTrace_GoesThroughDeduperBeforeSummarize
// drives a zipkin dual-ID fixture through the real endpoint path
// (newTraceSummaryHandler -> fetchTraceForSummary -> combiner.NewTypedTraceByIDV2),
// whose finalize() runs the span-ID deduper before tracesummary.Summarize
// sees the trace. The deduper reparents the SERVER-kind "handle" span under
// the CLIENT-kind "call" span (which kept the originally-shared span ID),
// and reparents "do-work" (a child of the shared ID) under "handle"'s new
// ID. The resulting critical path is therefore the full linear chain
// root -> call -> handle -> do-work, unlike Summarize called directly on
// the raw fixture (see TestSummarize_DuplicateSpanIDZipkinPattern in the
// tracesummary package), where "call" and "handle" are still siblings and
// the walk skips over "call" entirely.
func TestTraceSummaryHandler_ZipkinDualIDTrace_GoesThroughDeduperBeforeSummarize(t *testing.T) {
	tracePipeline := traceSummaryTestPipeline(t, map[string]*tempopb.TraceByIDResponse{
		"abc123": {Trace: zipkinDualIDTestTrace()},
	})
	o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	handler := newHandler(nil, newTraceSummaryHandler("", tracePipeline, o, combiner.NewTypedTraceByIDV2, log.NewNopLogger(), nil), log.NewNopLogger())

	req := newTraceSummaryTestRequest(t, http.MethodGet, "abc123", "")
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)

	var summary tracesummary.Summary
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &summary))

	require.Equal(t, 4, summary.SpanCount)
	require.Equal(t, "root-svc", summary.RootService)
	require.Equal(t, "root-op", summary.RootSpanName)

	var pathNames []string
	for _, p := range summary.CriticalPath {
		pathNames = append(pathNames, p.Name)
	}
	require.Equal(t, []string{"root-op", "call", "handle", "do-work"}, pathNames)
}

func TestBuildTraceSummaryTraceByIDRequest(t *testing.T) {
	start := time.Unix(100, 0)
	end := time.Unix(200, 0)

	req := buildTraceSummaryTraceByIDRequest(context.Background(), "", "abc123", start, end, nil)

	require.Equal(t, http.MethodGet, req.Method)
	require.Equal(t, "/api/v2/traces/abc123", req.URL.Path)
	require.Equal(t, "100", req.URL.Query().Get("start"))
	require.Equal(t, "200", req.URL.Query().Get("end"))
	require.Equal(t, api.HeaderAcceptProtobuf, req.Header.Get(api.HeaderAccept))

	_, err := api.ParseTraceID(req)
	require.NoError(t, err)

	_, _, _, startTime, endTime, err := api.ParseTraceByIDRequest(req)
	require.NoError(t, err)
	require.Equal(t, start, startTime)
	require.Equal(t, end, endTime)
}

func TestFetchTraceForSummary(t *testing.T) {
	traceID := "abc123"
	trace := traceSummaryTestTrace()
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

	actual, err := fetchTraceForSummary(context.Background(), "test-tenant", traceID, time.Time{}, time.Time{}, nil, "/tempo", tracePipeline, o, combiner.NewTypedTraceByIDV2, log.NewNopLogger(), nil)
	require.NoError(t, err)
	require.NotNil(t, actual)
	require.True(t, proto.Equal(trace, actual.Trace))
	require.Equal(t, uint64(123), actual.Metrics.GetInspectedBytes())
	require.NotNil(t, gotReq)
	require.Equal(t, "/tempo/api/v2/traces/abc123", gotReq.URL.Path)
	require.Equal(t, api.HeaderAcceptProtobuf, gotReq.Header.Get(api.HeaderAccept))
}

func traceSummaryTestPipeline(t *testing.T, traces map[string]*tempopb.TraceByIDResponse) pipeline.AsyncRoundTripper[combiner.PipelineResponse] {
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

func newTraceSummaryTestRequest(t *testing.T, method, traceID string, query string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, "/api/v2/traces/"+traceID+"/summary"+query, nil)
	req = req.WithContext(user.InjectOrgID(req.Context(), "test-tenant"))
	return mux.SetURLVars(req, map[string]string{"traceID": traceID})
}

func TestTraceSummaryHandlerReturnsSummary(t *testing.T) {
	tracePipeline := traceSummaryTestPipeline(t, map[string]*tempopb.TraceByIDResponse{
		"abc123": {Trace: traceSummaryTestTrace()},
	})
	o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	handler := newHandler(nil, newTraceSummaryHandler("", tracePipeline, o, combiner.NewTypedTraceByIDV2, log.NewNopLogger(), nil), log.NewNopLogger())

	req := newTraceSummaryTestRequest(t, http.MethodGet, "abc123", "")
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, api.HeaderAcceptJSON, resp.Header().Get(api.HeaderContentType))

	var summary tracesummary.Summary
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &summary))
	require.Equal(t, "checkout", summary.RootService)
	require.Equal(t, 2, summary.SpanCount)
}

func TestTraceSummaryHandlerTraceNotFound(t *testing.T) {
	tracePipeline := traceSummaryTestPipeline(t, map[string]*tempopb.TraceByIDResponse{
		"abc123": {Trace: &tempopb.Trace{}},
	})
	o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	handler := newHandler(nil, newTraceSummaryHandler("", tracePipeline, o, combiner.NewTypedTraceByIDV2, log.NewNopLogger(), nil), log.NewNopLogger())

	req := newTraceSummaryTestRequest(t, http.MethodGet, "abc123", "")
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestTraceSummaryHandlerInvalidTraceID(t *testing.T) {
	o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	handler := newHandler(nil, newTraceSummaryHandler("", nil, o, combiner.NewTypedTraceByIDV2, log.NewNopLogger(), nil), log.NewNopLogger())

	req := newTraceSummaryTestRequest(t, http.MethodGet, "not-hex", "")
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestTraceSummaryHandlerInvalidQueryParams(t *testing.T) {
	o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	handler := newHandler(nil, newTraceSummaryHandler("", nil, o, combiner.NewTypedTraceByIDV2, log.NewNopLogger(), nil), log.NewNopLogger())

	req := newTraceSummaryTestRequest(t, http.MethodGet, "abc123", "?start=notanumber")
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestTraceSummaryHandlerDataAccessErrorReturnsBadRequest(t *testing.T) {
	tracePipeline := traceSummaryTestPipeline(t, map[string]*tempopb.TraceByIDResponse{
		"abc123": {Trace: traceSummaryTestTrace()},
	})
	o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	dataAccessController := &traceDiffDataAccessController{err: errors.New("policy rejected")}
	handler := newHandler(nil, newTraceSummaryHandler("", tracePipeline, o, combiner.NewTypedTraceByIDV2, log.NewNopLogger(), dataAccessController), log.NewNopLogger())

	req := newTraceSummaryTestRequest(t, http.MethodGet, "abc123", "")
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
	require.Contains(t, resp.Body.String(), "policy rejected")
	require.Positive(t, dataAccessController.callCount())
}

func TestTraceSummaryHandlerWrongMethod(t *testing.T) {
	handler := newHandler(nil, newTraceSummaryHandler("", nil, nil, nil, log.NewNopLogger(), nil), log.NewNopLogger())

	req := newTraceSummaryTestRequest(t, http.MethodPost, "abc123", "")
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	require.Equal(t, http.StatusMethodNotAllowed, resp.Code)
}

func TestTraceSummaryHandlerPartialTraceStillSummarized(t *testing.T) {
	tracePipeline := traceSummaryTestPipeline(t, map[string]*tempopb.TraceByIDResponse{
		"abc123": {
			Trace:  traceSummaryTestTrace(),
			Status: tempopb.PartialStatus_PARTIAL,
		},
	})
	o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.NewRegistry())
	require.NoError(t, err)
	handler := newHandler(nil, newTraceSummaryHandler("", tracePipeline, o, combiner.NewTypedTraceByIDV2, log.NewNopLogger(), nil), log.NewNopLogger())

	req := newTraceSummaryTestRequest(t, http.MethodGet, "abc123", "")
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)

	var summary tracesummary.Summary
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &summary))
	require.Equal(t, 2, summary.SpanCount)
	require.Equal(t, "checkout", summary.RootService)
}
