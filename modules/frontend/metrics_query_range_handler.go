package frontend

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level" //nolint:all //deprecated
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/modules/frontend/pipeline"

	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
)

// newQueryRangeStreamingGRPCHandler returns a handler that streams results from the HTTP handler
func newQueryRangeStreamingGRPCHandler(cfg Config, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], apiPrefix string, logger log.Logger, dataAccessController DataAccessController) streamingQueryRangeHandler {
	postSLOHook := metricsSLOPostHook(cfg.Metrics.SLO)
	downstreamPath := path.Join(apiPrefix, api.PathMetricsQueryRange)

	return func(req *tempopb.QueryRangeRequest, srv tempopb.StreamingQuerier_MetricsQueryRangeServer) error {
		ctx := srv.Context()
		var err error

		headers := headersFromGrpcContext(ctx)
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
		if err := validateQueryRangeReq(cfg, req); err != nil {
			return err
		}

		// Normalize exemplars before combiner creation: 0 (unspecified) or above the configured
		// maximum defaults to MaxExemplars.
		if req.Exemplars == 0 || req.Exemplars > cfg.Metrics.Sharder.MaxExemplars {
			req.Exemplars = cfg.Metrics.Sharder.MaxExemplars
		}

		traceql.AlignRequest(req)

		// the end time cutoff is applied here because it has to be done before combiner creation
		// TODO: this is a copy of ClampDateRangeReq and needs to be removed after a proper fix
		if cfg.QueryEndCutoff > 0 {
			now := time.Now()
			maxEnd := now.Add(-cfg.QueryEndCutoff)
			reqEnd := time.Unix(0, int64(req.End))
			if maxEnd.Before(reqEnd) {
				req.End = uint64(maxEnd.UnixNano())
				traceql.AlignEndToLeft(req) // realign, but always to the left
			}
		}

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

		collector := pipeline.NewGRPCCollector(next, cfg.ResponseConsumers, c, func(qrr *tempopb.QueryRangeResponse) error {
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
		logQueryRangeResult(logger, tenant, duration.Seconds(), req, finalResponse, err)
		return err
	}
}

// newMetricsQueryRangeHTTPHandler returns a handler that returns a single response from the HTTP handler
func newMetricsQueryRangeHTTPHandler(cfg Config, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], logger log.Logger, dataAccessController DataAccessController) http.RoundTripper {
	postSLOHook := metricsSLOPostHook(cfg.Metrics.SLO)

	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		tenant, errResp := extractTenant(req, logger)
		if errResp != nil {
			return errResp, nil
		}
		start := time.Now()

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

		if err := validateQueryRangeReq(cfg, queryRangeReq); err != nil {
			return httpInvalidRequest(err), nil
		}

		// Normalize exemplars before combiner creation: 0 (unspecified) or above the configured
		// maximum defaults to MaxExemplars.
		if queryRangeReq.Exemplars == 0 || queryRangeReq.Exemplars > cfg.Metrics.Sharder.MaxExemplars {
			queryRangeReq.Exemplars = cfg.Metrics.Sharder.MaxExemplars
		}

		traceql.AlignRequest(queryRangeReq)

		// the end time cutoff is applied here because it has to be done before combiner creation
		// TODO: this is a copy of ClampDateRangeReq and needs to be removed after a proper fix
		if cfg.QueryEndCutoff > 0 {
			now := time.Now()
			maxEnd := now.Add(-cfg.QueryEndCutoff)
			reqEnd := time.Unix(0, int64(queryRangeReq.End))
			if maxEnd.Before(reqEnd) {
				queryRangeReq.End = uint64(maxEnd.UnixNano())
				traceql.AlignEndToLeft(queryRangeReq) // realign, but always to the left
			}
		}
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
		queryRangeResp, _ := combiner.GRPCDiff()
		if queryRangeResp != nil && queryRangeResp.Metrics != nil {
			bytesProcessed = queryRangeResp.Metrics.InspectedBytes
		}

		duration := time.Since(start)
		postSLOHook(resp, tenant, bytesProcessed, duration, err)
		logQueryRangeResult(logger, tenant, duration.Seconds(), queryRangeReq, queryRangeResp, err)
		return resp, err
	})
}

func logQueryRangeResult(logger log.Logger, tenantID string, durationSeconds float64, req *tempopb.QueryRangeRequest, resp *tempopb.QueryRangeResponse, err error) {
	if resp == nil {
		level.Info(logger).Log(
			"msg", "query range response - no resp",
			"tenant", tenantID,
			"duration_seconds", durationSeconds,
			"error", err)

		return
	}

	if resp.Metrics == nil {
		level.Info(logger).Log(
			"msg", "query range response - no metrics",
			"tenant", tenantID,
			"query", req.Query,
			"range_nanos", req.End-req.Start,
			"duration_seconds", durationSeconds,
			"error", err)
		return
	}

	level.Info(logger).Log(
		"msg", "query range response",
		"tenant", tenantID,
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
		"error", err)
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

func validateQueryRangeReq(cfg Config, req *tempopb.QueryRangeRequest) error {
	if req.Start > req.End {
		return errors.New("end must be greater than start")
	}
	if cfg.Metrics.MaxIntervals != 0 && (req.Step == 0 || (req.End-req.Start)/req.Step > cfg.Metrics.MaxIntervals) {
		minimumStep := (req.End - req.Start) / cfg.Metrics.MaxIntervals
		return fmt.Errorf(
			"step of %s is too small, minimum step for given range is %s",
			time.Duration(req.Step).String(),
			time.Duration(minimumStep).String(),
		)
	}
	return nil
}
