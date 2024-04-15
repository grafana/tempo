package frontend

import (
	"bytes"
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
)

// jpe - remove docs on prom compat

// newQueryRangeStreamingGRPCHandler returns a handler that streams results from the HTTP handler
func newQueryRangeStreamingGRPCHandler(cfg Config, next pipeline.AsyncRoundTripper[*http.Response], apiPrefix string, logger log.Logger) streamingQueryRangeHandler {
	postSLOHook := searchSLOPostHook(cfg.Search.SLO)
	downstreamPath := path.Join(apiPrefix, api.PathMetricsQueryRange)

	return func(req *tempopb.QueryRangeRequest, srv tempopb.StreamingQuerier_QueryRangeServer) error {
		httpReq := api.BuildQueryRangeRequest(&http.Request{
			URL:    &url.URL{Path: downstreamPath},
			Header: http.Header{},
			Body:   io.NopCloser(bytes.NewReader([]byte{})),
		}, req)

		ctx := srv.Context()
		httpReq = httpReq.WithContext(ctx)
		tenant, _ := user.ExtractOrgID(ctx)
		start := time.Now()

		var finalResponse *tempopb.QueryRangeResponse
		c := combiner.NewTypedQueryRange()
		collector := pipeline.NewGRPCCollector(next, c, func(qrr *tempopb.QueryRangeResponse) error {
			finalResponse = qrr // sadly we can't pass srv.Send directly into the collector. we need bytesProcessed for the SLO calculations
			return srv.Send(qrr)
		})

		logQueryRangeRequest(logger, tenant, req)
		err := collector.RoundTrip(httpReq)

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
func newMetricsQueryRangeHTTPHandler(cfg Config, next pipeline.AsyncRoundTripper[*http.Response], logger log.Logger) http.RoundTripper {
	postSLOHook := searchSLOPostHook(cfg.Search.SLO)

	return pipeline.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		tenant, _ := user.ExtractOrgID(req.Context())
		start := time.Now()

		// parse request
		queryRangeReq, err := api.ParseQueryRangeRequest(req)
		if err != nil {
			level.Error(logger).Log("msg", "query range: parse search request failed", "err", err)
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Status:     http.StatusText(http.StatusBadRequest),
				Body:       io.NopCloser(strings.NewReader(err.Error())),
			}, nil
		}

		logQueryRangeRequest(logger, tenant, queryRangeReq)

		// build and use roundtripper
		combiner := combiner.NewTypedQueryRange()
		rt := pipeline.NewHTTPCollector(next, combiner)

		resp, err := rt.RoundTrip(req)

		// ask for the typed diff and use that for the SLO hook. it will have up to date metrics
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
			"msg", "query range results - no resp",
			"tenant", tenantID,
			"duration_seconds", durationSeconds,
			"error", err)

		return
	}

	if resp.Metrics == nil {
		level.Info(logger).Log(
			"msg", "query range results - no metrics",
			"tenant", tenantID,
			"query", req.Query,
			"range_nanos", req.End-req.Start,
			"duration_seconds", durationSeconds,
			"error", err)
		return
	}

	level.Info(logger).Log(
		"msg", "query range results",
		"tenant", tenantID,
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
		"error", err)
}

func logQueryRangeRequest(logger log.Logger, tenantID string, req *tempopb.QueryRangeRequest) {
	level.Info(logger).Log(
		"msg", "query range request",
		"tenant", tenantID,
		"query", req.Query,
		"range_seconds", req.End-req.Start,
		"mode", req.QueryMode,
		"step", req.Step)
}
