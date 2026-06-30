package frontend

import (
	"bytes"
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

	"github.com/go-kit/log"
	"github.com/go-kit/log/level" //nolint:all //deprecated
	"github.com/gorilla/mux"
	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/cache"
	"github.com/grafana/tempo/pkg/model/tracediff"
	"github.com/grafana/tempo/pkg/tempopb"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	traceByIDStartParam = "start"
	traceByIDEndParam   = "end"
)

var (
	errTraceDiffTraceNotFound = errors.New("trace not found")
	errTraceDiffPartialTrace  = errors.New("partial traces cannot be diffed")
)

// newTraceDiffHandler creates an HTTP handler for trace diff requests.
// EXPERIMENTAL: this endpoint is not yet a stable API contract.
func newTraceDiffHandler(_ Config, apiPrefix string, tracePipeline pipeline.AsyncRoundTripper[combiner.PipelineResponse], o overrides.Interface, combinerFn func(int, api.MarshallingFormat, combiner.TraceRedactor, combiner.TraceByIDV2Options) combiner.GRPCCombiner[*tempopb.TraceByIDResponse], _ cache.Provider, logger log.Logger, dataAccessController DataAccessController) http.RoundTripper {
	fetchTrace := func(ctx context.Context, tenant string, traceReq api.TraceDiffTraceRequest, headers http.Header) (*tempopb.TraceByIDResponse, error) {
		return fetchTraceForDiff(ctx, tenant, traceReq, headers, apiPrefix, tracePipeline, o, combinerFn, logger, dataAccessController)
	}

	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
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

		diffReq, err := api.ParseTraceDiffRequest(req)
		if err != nil {
			return httpInvalidRequest(err), nil
		}

		level.Info(logger).Log(
			"msg", "trace diff request",
			"tenant", tenant,
			"path", req.URL.Path,
			"base_trace_id", diffReq.Base.TraceID,
			"base_start", traceDiffTimeParam(diffReq.Base.Start),
			"base_end", traceDiffTimeParam(diffReq.Base.End),
			"compare_trace_id", diffReq.Compare.TraceID,
			"compare_start", traceDiffTimeParam(diffReq.Compare.Start),
			"compare_end", traceDiffTimeParam(diffReq.Compare.End))

		baseResp, compareResp, err := fetchTracesForDiff(req.Context(), tenant, diffReq, req.Header, fetchTrace)
		if err != nil {
			return traceDiffErrorResponse(err), nil
		}
		// Trace diff intentionally uses the single-trace limit as a combined input budget.
		// Diffing holds both traces plus normalized/indexed structures in memory, so allowing
		// two max-sized traces would preserve the worst case this guard is meant to avoid.
		if err := validateTraceDiffInputSize(baseResp.Trace, compareResp.Trace, o.MaxBytesPerTrace(tenant)); err != nil {
			return traceDiffErrorResponse(err), nil
		}

		result, err := tracediff.Diff(baseResp.Trace, compareResp.Trace, tracediff.Format(diffReq.Format))
		if err != nil {
			return traceDiffErrorResponse(err), nil
		}

		body, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("marshal trace diff response: %w", err)
		}
		return traceDiffJSONResponse(body), nil
	})
}

func traceDiffTimeParam(v *int64) any {
	if v == nil {
		return ""
	}
	return *v
}

func validateTraceDiffInputSize(base, compare *tempopb.Trace, maxBytes int) error {
	if maxBytes <= 0 {
		return nil
	}

	maxBytes64 := int64(maxBytes)
	inputBytes := traceDiffInputSize(base) + traceDiffInputSize(compare)
	if inputBytes <= maxBytes64 {
		return nil
	}

	return status.Errorf(codes.ResourceExhausted, "trace diff input too large: combined trace size %d bytes exceeds limit %d bytes", inputBytes, maxBytes64)
}

func traceDiffInputSize(trace *tempopb.Trace) int64 {
	if trace == nil {
		return 0
	}
	return int64(trace.Size())
}

// It builds an http request to pass to the TraceByIdHandler
func buildTraceDiffTraceByIDRequest(ctx context.Context, apiPrefix string, traceReq api.TraceDiffTraceRequest, headers http.Header) *http.Request {
	u := &url.URL{
		Path: path.Join(apiPrefix, "/api/v2/traces", traceReq.TraceID),
	}
	q := u.Query()
	if traceReq.Start != nil {
		q.Set(traceByIDStartParam, strconv.FormatInt(*traceReq.Start, 10))
	}
	if traceReq.End != nil {
		q.Set(traceByIDEndParam, strconv.FormatInt(*traceReq.End, 10))
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

	return mux.SetURLVars(req, map[string]string{"traceID": traceReq.TraceID})
}

func fetchTraceForDiff(ctx context.Context, tenant string, traceReq api.TraceDiffTraceRequest, headers http.Header, apiPrefix string, tracePipeline pipeline.AsyncRoundTripper[combiner.PipelineResponse], o overrides.Interface, combinerFn func(int, api.MarshallingFormat, combiner.TraceRedactor, combiner.TraceByIDV2Options) combiner.GRPCCombiner[*tempopb.TraceByIDResponse], logger log.Logger, dataAccessController DataAccessController) (*tempopb.TraceByIDResponse, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	traceByIDReq := buildTraceDiffTraceByIDRequest(ctx, apiPrefix, traceReq, headers)
	traceRedactor, err := traceRedactorForDiff(traceByIDReq, logger, dataAccessController)
	if err != nil {
		return nil, fmt.Errorf("authorize trace %s: %w", traceReq.TraceID, err)
	}

	resps, err := tracePipeline.RoundTrip(pipeline.NewHTTPRequest(traceByIDReq))
	if err != nil {
		return nil, fmt.Errorf("fetch trace %s: %w", traceReq.TraceID, err)
	}

	// diff works on full traces, so no spanset filter.
	comb := combinerFn(o.MaxBytesPerTrace(tenant), api.MarshallingFormatProtobuf, traceRedactor, combiner.TraceByIDV2Options{})
	for {
		resp, done, err := resps.Next(ctx)
		if err != nil {
			return nil, fmt.Errorf("read trace %s response: %w", traceReq.TraceID, err)
		}
		if resp != nil {
			if err := comb.AddResponse(resp); err != nil {
				return nil, fmt.Errorf("combine trace %s response: %w", traceReq.TraceID, err)
			}
		}
		if comb.ShouldQuit() || done {
			break
		}
	}

	traceResp, err := comb.GRPCFinal()
	if err != nil {
		return nil, fmt.Errorf("finalize trace %s response: %w", traceReq.TraceID, err)
	}
	if traceResp == nil || !traceHasSpans(traceResp.Trace) {
		return nil, fmt.Errorf("trace %s: %w", traceReq.TraceID, errTraceDiffTraceNotFound)
	}
	if traceResp.GetStatus() == tempopb.PartialStatus_PARTIAL {
		return nil, fmt.Errorf("trace %s: %w", traceReq.TraceID, errTraceDiffPartialTrace)
	}
	return traceResp, nil
}

func fetchTracesForDiff(ctx context.Context, tenant string, diffReq *api.TraceDiffRequest, headers http.Header, fetchTrace func(context.Context, string, api.TraceDiffTraceRequest, http.Header) (*tempopb.TraceByIDResponse, error)) (*tempopb.TraceByIDResponse, *tempopb.TraceByIDResponse, error) {
	g, ctx := errgroup.WithContext(ctx)
	var (
		baseResp       *tempopb.TraceByIDResponse
		compareResp    *tempopb.TraceByIDResponse
		baseHeaders    = headers.Clone()
		compareHeaders = headers.Clone()
	)

	g.Go(func() error {
		resp, err := fetchTrace(ctx, tenant, diffReq.Base, baseHeaders)
		if err != nil {
			return fmt.Errorf("fetch base trace: %w", err)
		}
		baseResp = resp
		return nil
	})
	g.Go(func() error {
		resp, err := fetchTrace(ctx, tenant, diffReq.Compare, compareHeaders)
		if err != nil {
			return fmt.Errorf("fetch compare trace: %w", err)
		}
		compareResp = resp
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, nil, err
	}
	return baseResp, compareResp, nil
}

func traceRedactorForDiff(req *http.Request, logger log.Logger, dataAccessController DataAccessController) (combiner.TraceRedactor, error) {
	if dataAccessController == nil {
		return nil, nil
	}

	traceRedactor, err := dataAccessController.HandleHTTPTraceByIDReq(req)
	if err != nil {
		level.Error(logger).Log("msg", "trace diff: failed to get trace redactor", "err", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return traceRedactor, nil
}

func traceHasSpans(trace *tempopb.Trace) bool {
	if trace == nil {
		return false
	}
	for _, rs := range trace.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			if len(ss.Spans) > 0 {
				return true
			}
		}
	}
	return false
}

func traceDiffJSONResponse(body []byte) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     http.StatusText(http.StatusOK),
		Header: http.Header{
			api.HeaderContentType: []string{api.HeaderAcceptJSON},
		},
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
	}
}

func traceDiffErrorResponse(err error) *http.Response {
	statusCode := http.StatusInternalServerError
	switch {
	case errors.Is(err, errTraceDiffTraceNotFound):
		statusCode = http.StatusNotFound
	case errors.Is(err, errTraceDiffPartialTrace):
		statusCode = http.StatusUnprocessableEntity
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
