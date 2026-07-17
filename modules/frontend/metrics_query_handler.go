package frontend

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/status"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/tracing"
	"google.golang.org/grpc/codes"
)

func newQueryInstantStreamingGRPCHandler(cfg Config, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], o overrides.Interface, apiPrefix string, logger log.Logger, dataAccessController DataAccessController) streamingQueryInstantHandler {
	postSLOHook := metricsSLOPostHook(cfg.Metrics.SLO)
	downstreamPath := path.Join(apiPrefix, api.PathMetricsQueryRange)

	return func(req *tempopb.QueryInstantRequest, srv tempopb.StreamingQuerier_MetricsQueryInstantServer) error {
		start := time.Now()
		ctx := pipeline.WithQueryShapeCell(srv.Context())
		tenant, err := user.ExtractOrgID(ctx)
		if err != nil {
			return err
		}

		headers := headersFromGrpcContext(ctx)
		if err := pipeline.ValidateTraceQLQuerySize(req.Query, cfg.MaxQueryExpressionSizeBytes); err != nil {
			return status.Error(codes.InvalidArgument, err.Error())
		}
		if dataAccessController != nil {
			err = dataAccessController.HandleGRPCQueryInstantReq(ctx, req)
			if err != nil {
				level.Error(logger).Log("msg", "instant query streaming: access control handling failed", "err", err)
				return err
			}
		}
		// --------------------------------------------------
		// Rewrite into a query_range request.
		// --------------------------------------------------
		qr := &tempopb.QueryRangeRequest{
			Query: req.Query,
			Start: req.Start,
			End:   req.End,
			Step:  req.End - req.Start,
		}
		qr.SetInstant(true)

		if err := clampInstantQueryEnd(cfg, qr); err != nil {
			return status.Error(codes.InvalidArgument, err.Error())
		}
		qr.Step = qr.End - qr.Start // keep Step == End-Start for instant

		if err := validateQueryRangeReq(ctx, cfg, o, qr); err != nil {
			return status.Error(codes.InvalidArgument, err.Error())
		}

		httpReq := api.BuildQueryRangeRequest(&http.Request{
			URL:    &url.URL{Path: downstreamPath},
			Header: headers,
			Body:   io.NopCloser(bytes.NewReader([]byte{})),
		}, qr, "") // dedicated cols are never passed from the caller
		httpReq = httpReq.Clone(ctx)

		var finalResponse *tempopb.QueryInstantResponse
		c, err := combiner.NewTypedQueryRange(qr, cfg.Metrics.Sharder.MaxResponseSeries)
		if err != nil {
			return err
		}

		collector := pipeline.NewGRPCCollector(next, cfg.ResponseConsumers, cfg.MaxGRPCStreamingPacketSize, c, func(qrr *tempopb.QueryRangeResponse) error {
			// Translate each diff into the instant version and send it
			resp := translateQueryRangeToInstant(*qrr)
			finalResponse = &resp // Save last response for bytesProcessed for the SLO calculations
			return srv.Send(&resp)
		})

		logQueryInstantRequest(logger, tenant, req)
		err = collector.RoundTrip(httpReq)

		duration := time.Since(start)
		bytesProcessed := uint64(0)
		if finalResponse != nil && finalResponse.Metrics != nil {
			bytesProcessed = finalResponse.Metrics.InspectedBytes
		}
		postSLOHook(nil, tenant, bytesProcessed, duration, err)
		logQueryInstantResult(ctx, logger, tenant, duration.Seconds(), req, finalResponse, err)
		return err
	}
}

// newMetricsQueryInstantHTTPHandler handles instant queries.  Internally these are rewritten as query_range with single step
// to make use of the existing pipeline.
func newMetricsQueryInstantHTTPHandler(cfg Config, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], o overrides.Interface, logger log.Logger, dataAccessController DataAccessController) http.RoundTripper {
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
			if err := dataAccessController.HandleHTTPQueryInstantReq(req); err != nil {
				level.Error(logger).Log("msg", "http instant query: access control handling failed", "err", err)
				return httpInvalidRequest(err), nil
			}
		}
		// Parse request
		i, err := api.ParseQueryInstantRequest(req)
		if err != nil {
			level.Error(logger).Log("msg", "query instant: parse search request failed", "err", err)
			return httpInvalidRequest(err), nil
		}

		logQueryInstantRequest(logger, tenant, i)

		// --------------------------------------------------
		// Rewrite into a query_range request.
		// --------------------------------------------------
		qr := &tempopb.QueryRangeRequest{
			Query: i.Query,
			Start: i.Start,
			End:   i.End,
			Step:  i.End - i.Start,
		}
		qr.SetInstant(true)

		if err := clampInstantQueryEnd(cfg, qr); err != nil {
			return httpInvalidRequest(err), nil
		}
		qr.Step = qr.End - qr.Start // keep Step == End-Start for instant

		if err := validateQueryRangeReq(req.Context(), cfg, o, qr); err != nil {
			return httpInvalidRequest(err), nil
		}

		// Clone existing to keep it unaltered.
		req = req.Clone(req.Context())
		req.URL.Path = strings.ReplaceAll(req.URL.Path, api.PathMetricsQueryInstant, api.PathMetricsQueryRange)
		req = api.BuildQueryRangeRequest(req, qr, "") // dedicated cols are never passed from the caller

		combiner, err := combiner.NewTypedQueryRange(qr, cfg.Metrics.Sharder.MaxResponseSeries)
		if err != nil {
			level.Error(logger).Log("msg", "query instant: query range combiner failed", "err", err)
			return httpInvalidRequest(err), nil
		}
		rt := pipeline.NewHTTPCollector(next, cfg.ResponseConsumers, combiner)

		// Roundtrip the request and look for intermediate failures
		innerResp, err := rt.RoundTrip(req)
		if err != nil {
			return nil, err
		}
		if innerResp != nil && innerResp.StatusCode != http.StatusOK {
			return innerResp, nil
		}

		// --------------------------------------------------
		// Get the final data and translate to instant.
		// --------------------------------------------------
		qrResp, err := combiner.GRPCFinal()
		if err != nil {
			return nil, err
		}

		qiResp := translateQueryRangeToInstant(*qrResp)

		bodyString, err := new(jsonpb.Marshaler).MarshalToString(&qiResp)
		if err != nil {
			return nil, fmt.Errorf("error marshalling response body: %w", err)
		}

		resp := &http.Response{
			StatusCode: combiner.StatusCode(),
			Header: http.Header{
				api.HeaderContentType: {api.HeaderAcceptJSON},
			},
			Body:          io.NopCloser(strings.NewReader(bodyString)),
			ContentLength: int64(len([]byte(bodyString))),
		}

		duration := time.Since(start)
		var bytesProcessed uint64
		if qiResp.Metrics != nil {
			bytesProcessed = qiResp.Metrics.InspectedBytes
		}
		postSLOHook(resp, tenant, bytesProcessed, duration, err)
		logQueryInstantResult(req.Context(), logger, tenant, duration.Seconds(), i, &qiResp, err)

		return resp, nil
	})
}

// clampInstantQueryEnd pulls an instant query's ending timestamp to within the
// cutoff range. This is different from range queries because we don't align it to a step. Another
// special case is we also do it in a way as to not break caching. Instead of doing a hard overwrite
// of the end timestamp, we subtract just the right number of whole seconds to get it in range.
// For example if we query `Last 6 Hours` multiple times, we get random clock skew and network latency
// milliseconds across each run, so the naive cutoff ends up as `Last 5h59m30.xyzs` and xyz vary
// every time. This breaks caching and is unnecessarily precise. The cutoff range just provides some buffer
// and is not a hard cutoff. Subtracting in whole seconds eliminates all of the xyz ms jitter,
// and even though final range can still vary, predominantly it falls into repeated values and caches much better.
func clampInstantQueryEnd(cfg Config, req *tempopb.QueryRangeRequest) error {
	var (
		cutoff = time.Now().Add(-cfg.QueryEndCutoff)
		start  = time.Unix(0, int64(req.Start))
		end    = time.Unix(0, int64(req.End))
	)

	if !end.After(start) {
		return errEndMustBeGreaterThanStart
	}
	if end.Before(cutoff) {
		return nil
	}

	// Fewest whole seconds that land the end at or before the cutoff.
	reduce := time.Duration(math.Ceil(end.Sub(cutoff).Seconds())) * time.Second
	end = end.Add(-reduce)

	if !end.After(start) {
		return errQueryWindowWithinEndCutoff
	}

	req.End = uint64(end.UnixNano())
	return nil
}

func translateQueryRangeToInstant(input tempopb.QueryRangeResponse) tempopb.QueryInstantResponse {
	output := tempopb.QueryInstantResponse{
		Metrics: input.Metrics,
		Status:  input.Status,
		Message: input.Message,
	}
	for _, series := range input.Series {
		if len(series.Samples) == 0 {
			continue
		}
		// Use first value
		output.Series = append(output.Series, &tempopb.InstantSeries{
			Labels: series.Labels,
			Value:  series.Samples[0].Value,
		})
	}
	return output
}

func logQueryInstantResult(ctx context.Context, logger log.Logger, tenantID string, durationSeconds float64, req *tempopb.QueryInstantRequest, resp *tempopb.QueryInstantResponse, err error) {
	traceID, _ := tracing.ExtractTraceID(ctx)

	if resp == nil {
		recordResult(level.Info(logger), ctx,
			"msg", "query instant results - no resp",
			"tenant", tenantID,
			"traceID", traceID,
			"duration_seconds", durationSeconds,
			"error", err,
		)
		return
	}

	if resp.Metrics == nil {
		recordResult(level.Info(logger), ctx,
			"msg", "query instant results - no metrics",
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
		"msg", "query instant results",
		"tenant", tenantID,
		"traceID", traceID,
		"query", req.Query,
		"range_nanos", req.End-req.Start,
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

func logQueryInstantRequest(logger log.Logger, tenantID string, req *tempopb.QueryInstantRequest) {
	level.Info(logger).Log(
		"msg", "query instant request",
		"tenant", tenantID,
		"query", req.Query,
		"range_seconds", req.End-req.Start)
}
