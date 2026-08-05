package frontend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level" //nolint:all //deprecated
	"github.com/gorilla/mux"
	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/model/tracesummary"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var errTraceSummaryTraceNotFound = errors.New("trace not found")

// newTraceSummaryHandler creates an HTTP handler for trace summary requests.
// EXPERIMENTAL: this endpoint is not yet a stable API contract.
func newTraceSummaryHandler(cfg Config, apiPrefix string, tracePipeline pipeline.AsyncRoundTripper[combiner.PipelineResponse], o overrides.Interface, combinerFn func(int, api.MarshallingFormat, combiner.TraceRedactor) combiner.GRPCCombiner[*tempopb.TraceByIDResponse], logger log.Logger, dataAccessController DataAccessController) http.RoundTripper {
	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			return &http.Response{
				StatusCode: http.StatusMethodNotAllowed,
				Status:     http.StatusText(http.StatusMethodNotAllowed),
				Body:       io.NopCloser(strings.NewReader(http.StatusText(http.StatusMethodNotAllowed))),
			}, nil
		}

		tenant, errResp := extractTenant(req, logger)
		if errResp != nil {
			return errResp, nil
		}

		traceIDBytes, err := api.ParseTraceID(req)
		if err != nil {
			return httpInvalidRequest(err), nil
		}
		traceID := util.TraceIDToHexString(traceIDBytes)

		_, _, _, startTime, endTime, err := api.ParseTraceByIDRequest(req)
		if err != nil {
			return httpInvalidRequest(err), nil
		}

		level.Info(logger).Log(
			"msg", "trace summary request",
			"tenant", tenant,
			"path", req.URL.Path,
			"trace_id", traceID)

		traceResp, err := fetchTraceForSummary(req.Context(), cfg, tenant, traceID, startTime, endTime, req.Header, apiPrefix, tracePipeline, o, combinerFn, logger, dataAccessController)
		if err != nil {
			return traceSummaryErrorResponse(err), nil
		}

		summary, err := tracesummary.Summarize(traceResp.Trace)
		if err != nil {
			return nil, fmt.Errorf("summarize trace %s: %w", traceID, err)
		}

		body, err := json.Marshal(summary)
		if err != nil {
			return nil, fmt.Errorf("marshal trace summary response: %w", err)
		}
		return jsonResponse(body), nil
	})
}

// buildTraceSummaryTraceByIDRequest builds an internal http request to pass to the TraceByIdHandler.
func buildTraceSummaryTraceByIDRequest(ctx context.Context, apiPrefix, traceID string, startTime, endTime time.Time, headers http.Header) *http.Request {
	u := &url.URL{
		Path: path.Join(apiPrefix, "/api/v2/traces", traceID),
	}
	q := u.Query()
	if !startTime.IsZero() {
		q.Set(traceByIDStartParam, strconv.FormatInt(startTime.Unix(), 10))
	}
	if !endTime.IsZero() {
		q.Set(traceByIDEndParam, strconv.FormatInt(endTime.Unix(), 10))
	}
	u.RawQuery = q.Encode()

	reqHeaders := headers.Clone()
	if reqHeaders == nil {
		reqHeaders = http.Header{}
	}

	req := (&http.Request{
		Method: http.MethodGet,
		URL:    u,
		Header: reqHeaders,
		Body:   http.NoBody,
	}).WithContext(ctx)
	req.Header.Set(api.HeaderAccept, api.HeaderAcceptProtobuf)

	return mux.SetURLVars(req, map[string]string{"traceID": traceID})
}

func fetchTraceForSummary(ctx context.Context, cfg Config, tenant, traceID string, startTime, endTime time.Time, headers http.Header, apiPrefix string, tracePipeline pipeline.AsyncRoundTripper[combiner.PipelineResponse], o overrides.Interface, combinerFn func(int, api.MarshallingFormat, combiner.TraceRedactor) combiner.GRPCCombiner[*tempopb.TraceByIDResponse], logger log.Logger, dataAccessController DataAccessController) (*tempopb.TraceByIDResponse, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	traceByIDReq := buildTraceSummaryTraceByIDRequest(ctx, apiPrefix, traceID, startTime, endTime, headers)

	// check marshalling format
	marshallingFormat := api.MarshalingFormatFromAcceptHeader(traceByIDReq.Header)

	// enforce all communication internal to Tempo to be in protobuf bytes
	traceByIDReq.Header.Set(api.HeaderAccept, api.HeaderAcceptProtobuf)

	var traceRedactor combiner.TraceRedactor
	if dataAccessController != nil {
		redactor, err := dataAccessController.HandleHTTPTraceByIDReq(traceByIDReq)
		if err != nil {
			level.Error(logger).Log("msg", "trace summary: failed to get trace redactor", "err", err)
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		traceRedactor = redactor
	}

	comb := combinerFn(o.MaxBytesPerTrace(tenant), marshallingFormat, traceRedactor)
	rt := pipeline.NewHTTPCollector(tracePipeline, cfg.ResponseConsumers, comb)

	start := time.Now()
	resp, err := rt.RoundTrip(traceByIDReq)
	elapsed := time.Since(start)

	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("fetch trace %s: %w", traceID, err)
	}

	traceResp, err := comb.GRPCFinal()
	if err != nil {
		return nil, fmt.Errorf("finalize trace %s response: %w", traceID, err)
	}
	if traceResp == nil || !traceHasSpans(traceResp.Trace) {
		return nil, fmt.Errorf("trace %s: %w", traceID, errTraceSummaryTraceNotFound)
	}

	logWithShape(level.Info(logger), traceByIDReq.Context(),
		"msg", "trace id response",
		"tenant", tenant,
		"traceID", traceID,
		"path", traceByIDReq.URL.Path,
		"duration_seconds", elapsed.Seconds(),
		"err", err,
	)
	return traceResp, nil
}

func traceSummaryErrorResponse(err error) *http.Response {
	statusCode := http.StatusInternalServerError
	switch {
	case errors.Is(err, errTraceSummaryTraceNotFound):
		statusCode = http.StatusNotFound
	case status.Code(err) == codes.NotFound:
		statusCode = http.StatusNotFound
	case status.Code(err) == codes.InvalidArgument:
		statusCode = http.StatusBadRequest
	case status.Code(err) == codes.ResourceExhausted:
		statusCode = http.StatusTooManyRequests
	}

	body := err.Error()
	if statusCode == http.StatusNotFound {
		body = http.StatusText(statusCode)
	}

	return &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
