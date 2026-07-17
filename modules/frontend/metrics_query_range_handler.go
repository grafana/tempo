package frontend

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level" //nolint:all //deprecated
	"github.com/gogo/status"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/util/tracing"
	"google.golang.org/grpc/codes"

	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
)

var errQueryWindowWithinEndCutoff = errors.New("query window falls entirely within query_end_cutoff")

// newQueryRangeStreamingGRPCHandler returns a handler that streams results from the HTTP handler
func newQueryRangeStreamingGRPCHandler(cfg Config, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], o overrides.Interface, apiPrefix string, logger log.Logger, dataAccessController DataAccessController) streamingQueryRangeHandler {
	postSLOHook := metricsSLOPostHook(cfg.Metrics.SLO)
	downstreamPath := path.Join(apiPrefix, api.PathMetricsQueryRange)

	return func(req *tempopb.QueryRangeRequest, srv tempopb.StreamingQuerier_MetricsQueryRangeServer) error {
		ctx := pipeline.WithQueryShapeCell(srv.Context())
		var err error

		headers := headersFromGrpcContext(ctx)
		if err := pipeline.ValidateTraceQLQuerySize(req.Query, cfg.MaxQueryExpressionSizeBytes); err != nil {
			return status.Error(codes.InvalidArgument, err.Error())
		}
		if dataAccessController != nil {
			err = dataAccessController.HandleGRPCQueryRangeReq(ctx, req)
			if err != nil {
				level.Error(logger).Log("msg", "query range streaming: access control handling failed", "err", err)
				return err
			}
		}
		// default step if not set
		if req.Step == 0 {
			req.Step = traceql.DefaultQueryRangeStep(req.Start, req.End)
		}
		if !req.HasInstant() { // if not found, set it explicitly
			req.SetInstant(false)
		}

		if err := clampQueryEndForValidation(cfg, req); err != nil {
			return status.Error(codes.InvalidArgument, err.Error())
		}

		if err := validateQueryRangeReq(ctx, cfg, o, req); err != nil {
			return status.Error(codes.InvalidArgument, err.Error())
		}

		if err := normalizeRequestExemplars(req, cfg.Metrics.Sharder.MaxExemplars); err != nil {
			return err
		}

		traceql.AlignRequest(req)
		// AlignRequest's alignEnd rounds end up to the next step boundary, which
		// can push it past now-QueryEndCutoff. Re-clamp to preserve the cutoff
		// invariant and round range-query ends back down to a step boundary.
		clampQueryEnd(cfg, req)

		httpReq := api.BuildQueryRangeRequest(&http.Request{
			URL:    &url.URL{Path: downstreamPath},
			Header: headers,
			Body:   io.NopCloser(bytes.NewReader([]byte{})),
		}, req, "") // dedicated cols are never passed from the caller

		httpReq = httpReq.WithContext(ctx)
		tenant, _ := user.ExtractOrgID(ctx)
		start := time.Now()

		var finalResponse *tempopb.QueryRangeResponse
		c, err := combiner.NewTypedQueryRange(req, cfg.Metrics.Sharder.MaxResponseSeries)
		if err != nil {
			return err
		}

		collector := pipeline.NewGRPCCollector(next, cfg.ResponseConsumers, cfg.MaxGRPCStreamingPacketSize, c, func(qrr *tempopb.QueryRangeResponse) error {
			finalResponse = qrr // sadly we can't pass srv.Send directly into the collector. we need bytesProcessed for the SLO calculations
			return srv.Send(qrr)
		})

		logQueryRangeRequest(logger, tenant, req)
		err = collector.RoundTrip(httpReq)

		duration := time.Since(start)
		bytesProcessed := uint64(0)
		if finalResponse != nil && finalResponse.Metrics != nil {
			bytesProcessed = finalResponse.Metrics.InspectedBytes
		}
		postSLOHook(nil, tenant, bytesProcessed, duration, err)
		logQueryRangeResult(ctx, logger, tenant, duration.Seconds(), req, finalResponse, err)
		return err
	}
}

// newMetricsQueryRangeHTTPHandler returns a handler that returns a single response from the HTTP handler
func newMetricsQueryRangeHTTPHandler(cfg Config, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], o overrides.Interface, logger log.Logger, dataAccessController DataAccessController) http.RoundTripper {
	postSLOHook := metricsSLOPostHook(cfg.Metrics.SLO)

	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		tenant, errResp := extractTenant(req, logger)
		if errResp != nil {
			return errResp, nil
		}
		start := time.Now()

		if err := pipeline.ValidateTraceQLQueryParamsSize(req.URL.Query(), cfg.MaxQueryExpressionSizeBytes); err != nil {
			return httpInvalidRequest(err), nil
		}
		if dataAccessController != nil {
			if err := dataAccessController.HandleHTTPQueryRangeReq(req); err != nil {
				level.Error(logger).Log("msg", "http query range: access control handling failed", "err", err)
				return httpInvalidRequest(err), nil
			}
		}
		// parse request
		queryRangeReq, err := api.ParseQueryRangeRequest(req)
		if err != nil {
			level.Error(logger).Log("msg", "query range: parse search request failed", "err", err)
			return httpInvalidRequest(err), nil
		}
		if !queryRangeReq.HasInstant() { // if not found, set it explicitly
			queryRangeReq.SetInstant(false)
		}
		logQueryRangeRequest(logger, tenant, queryRangeReq)

		// Clamp end before validation so the cap is checked against the effective range.
		if err := clampQueryEndForValidation(cfg, queryRangeReq); err != nil {
			return httpInvalidRequest(err), nil
		}

		if err := validateQueryRangeReq(req.Context(), cfg, o, queryRangeReq); err != nil {
			return httpInvalidRequest(err), nil
		}

		if err := normalizeRequestExemplars(queryRangeReq, cfg.Metrics.Sharder.MaxExemplars); err != nil {
			return httpInvalidRequest(err), nil
		}

		traceql.AlignRequest(queryRangeReq)
		// AlignRequest's alignEnd rounds end up to the next step boundary, which
		// can push it past now-QueryEndCutoff. Re-clamp to preserve the cutoff
		// invariant and round range-query ends back down to a step boundary.
		clampQueryEnd(cfg, queryRangeReq)
		req = api.BuildQueryRangeRequest(req, queryRangeReq, "")

		// build and use roundtripper
		combiner, err := combiner.NewTypedQueryRange(queryRangeReq, cfg.Metrics.Sharder.MaxResponseSeries)
		if err != nil {
			level.Error(logger).Log("msg", "query range: query range combiner failed", "err", err)
			return httpInvalidRequest(err), nil
		}
		rt := pipeline.NewHTTPCollector(next, cfg.ResponseConsumers, combiner)

		resp, err := rt.RoundTrip(req)

		// ask for the typed diff and use that for the SLO hook. it will have up to date metrics
		// todo: is there a way to remove this? it can be costly for large responses
		var bytesProcessed uint64
		queryRangeResp, finalErr := combiner.GRPCFinal()
		if queryRangeResp != nil && queryRangeResp.Metrics != nil {
			bytesProcessed = queryRangeResp.Metrics.InspectedBytes
		}

		duration := time.Since(start)
		postSLOHook(resp, tenant, bytesProcessed, duration, err)
		// When the pipeline returns an error response in-band (a frontend-generated
		// 4xx/5xx, e.g. from a sharder or the URL deny list), resp carries the status
		// code and RoundTrip's err is nil while GRPCFinal returns the reason. Fall back
		// to it for logging so the result log records why the query failed; otherwise it
		// logs "query range response - no resp" with error=null and the reason is lost.
		logErr := err
		if logErr == nil {
			logErr = finalErr
		}
		logQueryRangeResult(req.Context(), logger, tenant, duration.Seconds(), queryRangeReq, queryRangeResp, logErr)
		return resp, err
	})
}

// clampQueryEndForValidation clamps req.End to now-QueryEndCutoff without
// step-aligning it, so validation sees the effective user range instead of an
// internally aligned range. If the cutoff removes the whole query window, it
// returns a cutoff-specific error instead of the generic start/end error.
func clampQueryEndForValidation(cfg Config, req *tempopb.QueryRangeRequest) error {
	if req.Start >= req.End {
		return errEndMustBeGreaterThanStart
	}

	clamped := clampQueryEndToCutoff(cfg, req)
	if clamped && req.Start >= req.End {
		return errQueryWindowWithinEndCutoff
	}
	return nil
}

// clampQueryEnd clamps req.End to now-QueryEndCutoff when end is past that
// horizon, then rounds down to a step boundary for non-instant queries. When
// cutoff is 0 the clamp horizon is now, so clock-skewed future ends are still
// pulled back to the current frontend time.
func clampQueryEnd(cfg Config, req *tempopb.QueryRangeRequest) {
	if clampQueryEndToCutoff(cfg, req) {
		traceql.AlignEndToLeft(req)
	}
}

func clampQueryEndToCutoff(cfg Config, req *tempopb.QueryRangeRequest) bool {
	maxEnd := time.Now().Add(-cfg.QueryEndCutoff)
	reqEnd := time.Unix(0, int64(req.End))
	if reqEnd.After(maxEnd) {
		req.End = uint64(maxEnd.UnixNano())
		return true
	}
	return false
}

// normalizeRequestExemplars resolves the final exemplar limit for a query range request.
// It applies the exemplars hint from the TraceQL query if present, overriding the value
// from the HTTP parameter. req.Exemplars is then capped to maxExemplars.
// If no hint is set and req.Exemplars is 0 (unspecified), it defaults to maxExemplars.
func normalizeRequestExemplars(req *tempopb.QueryRangeRequest, maxExemplars uint32) error {
	expr, err := traceql.ParseNoOptimizations(req.Query)
	if err != nil {
		return err
	}
	if v, ok := expr.Hints.GetInt(traceql.HintExemplars, false); ok {
		req.Exemplars = uint32(max(v, 0)) //nolint: gosec // G115
	} else if v, ok := expr.Hints.GetBool(traceql.HintExemplars, false); ok && !v {
		req.Exemplars = 0
	} else if req.Exemplars == 0 {
		req.Exemplars = maxExemplars
	}
	if req.Exemplars > maxExemplars {
		req.Exemplars = maxExemplars
	}
	return nil
}

func logQueryRangeResult(ctx context.Context, logger log.Logger, tenantID string, durationSeconds float64, req *tempopb.QueryRangeRequest, resp *tempopb.QueryRangeResponse, err error) {
	traceID, _ := tracing.ExtractTraceID(ctx)

	if resp == nil {
		recordResult(level.Info(logger), ctx,
			"msg", "query range response - no resp",
			"tenant", tenantID,
			"traceID", traceID,
			"duration_seconds", durationSeconds,
			"error", err,
		)
		return
	}

	if resp.Metrics == nil {
		recordResult(level.Info(logger), ctx,
			"msg", "query range response - no metrics",
			"tenant", tenantID,
			"traceID", traceID,
			"query", req.Query,
			"range_nanos", req.End-req.Start,
			"duration_seconds", durationSeconds,
			"error", err,
		)
		return
	}

	recordResult(level.Info(logger), ctx,
		"msg", "query range response",
		"tenant", tenantID,
		"traceID", traceID,
		"query", req.Query,
		"range_nanos", req.End-req.Start,
		"max_series", req.MaxSeries,
		"duration_seconds", durationSeconds,
		"request_throughput", float64(resp.Metrics.InspectedBytes)/durationSeconds,
		"total_requests", resp.Metrics.TotalJobs,
		"total_blockBytes", resp.Metrics.TotalBlockBytes,
		"total_blocks", resp.Metrics.TotalBlocks,
		"completed_requests", resp.Metrics.CompletedJobs,
		"inspected_bytes", resp.Metrics.InspectedBytes,
		"inspected_traces", resp.Metrics.InspectedTraces,
		"inspected_spans", resp.Metrics.InspectedSpans,
		"partial_status", resp.Status,
		"partial_message", resp.Message,
		"num_response_series", len(resp.Series),
		"error", err,
	)
}

func logQueryRangeRequest(logger log.Logger, tenantID string, req *tempopb.QueryRangeRequest) {
	level.Info(logger).Log(
		"msg", "query range request",
		"tenant", tenantID,
		"query", req.Query,
		"range_nanos", req.End-req.Start,
		"mode", req.QueryMode,
		"step", req.Step)
}

func httpInvalidRequest(err error) *http.Response {
	return &http.Response{
		StatusCode: http.StatusBadRequest,
		Status:     http.StatusText(http.StatusBadRequest),
		Body:       io.NopCloser(strings.NewReader(err.Error())),
	}
}
