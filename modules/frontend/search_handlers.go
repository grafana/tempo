package frontend

import (
	"bytes"
	"fmt"
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
	"google.golang.org/grpc/codes"

	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
)

// newSearchStreamingGRPCHandler returns a handler that streams results from the HTTP handler
func newSearchStreamingGRPCHandler(cfg Config, next pipeline.AsyncRoundTripper[*http.Response], apiPrefix string, logger log.Logger) streamingSearchHandler {
	postSLOHook := searchSLOPostHook(cfg.Search.SLO)
	downstreamPath := path.Join(apiPrefix, api.PathSearch)

	return func(req *tempopb.SearchRequest, srv tempopb.StreamingQuerier_SearchServer) error {
		httpReq, err := api.BuildSearchRequest(&http.Request{
			URL:    &url.URL{Path: downstreamPath},
			Header: http.Header{},
			Body:   io.NopCloser(bytes.NewReader([]byte{})),
		}, req)
		if err != nil {
			level.Error(logger).Log("msg", "search streaming: build search request failed", "err", err)
			return status.Errorf(codes.InvalidArgument, "build search request failed: %s", err.Error())
		}

		ctx := srv.Context()
		httpReq = httpReq.WithContext(ctx)
		tenant, _ := user.ExtractOrgID(ctx)
		start := time.Now()

		limit, err := adjustLimit(req.Limit, cfg.Search.Sharder.DefaultLimit, cfg.Search.Sharder.MaxLimit)
		if err != nil {
			level.Error(logger).Log("msg", "search streaming: adjust limit failed", "err", err)
			return status.Errorf(codes.InvalidArgument, "adjust limit: %s", err.Error())
		}

		var finalResponse *tempopb.SearchResponse
		c := combiner.NewTypedSearch(int(limit))
		collector := pipeline.NewGRPCCollector[*tempopb.SearchResponse](next, c, func(sr *tempopb.SearchResponse) error {
			finalResponse = sr // sadly we can't srv.Send directly into the collector. we need bytesProcessed for the SLO calculations
			return srv.Send(sr)
		})

		logRequest(logger, tenant, req)
		err = collector.RoundTrip(httpReq)

		duration := time.Since(start)
		bytesProcessed := uint64(0)
		if finalResponse != nil && finalResponse.Metrics != nil {
			bytesProcessed = finalResponse.Metrics.InspectedBytes
		}
		postSLOHook(nil, tenant, bytesProcessed, duration, err)
		logResult(logger, tenant, duration.Seconds(), req, finalResponse, err)
		return err
	}
}

// newSearchHTTPHandler returns a handler that returns a single response from the HTTP handler
func newSearchHTTPHandler(cfg Config, next pipeline.AsyncRoundTripper[*http.Response], logger log.Logger) http.RoundTripper {
	postSLOHook := searchSLOPostHook(cfg.Search.SLO)

	return pipeline.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		tenant, _ := user.ExtractOrgID(req.Context())
		start := time.Now()

		// parse request
		searchReq, err := api.ParseSearchRequest(req)
		if err != nil {
			level.Error(logger).Log("msg", "search: parse search request failed", "err", err)
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Status:     http.StatusText(http.StatusBadRequest),
				Body:       io.NopCloser(strings.NewReader(err.Error())),
			}, nil
		}

		// build combiner with limit
		limit, err := adjustLimit(searchReq.Limit, cfg.Search.Sharder.DefaultLimit, cfg.Search.Sharder.MaxLimit)
		if err != nil {
			level.Error(logger).Log("msg", "search: adjust limit failed", "err", err)
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Status:     http.StatusText(http.StatusBadRequest),
				Body:       io.NopCloser(strings.NewReader(err.Error())),
			}, nil
		}

		logRequest(logger, tenant, searchReq)

		// build and use roundtripper
		combiner := combiner.NewTypedSearch(int(limit))
		rt := pipeline.NewHTTPCollector(next, combiner)

		resp, err := rt.RoundTrip(req)

		// ask for the typed diff and use that for the SLO hook. it will have up to date metrics
		var bytesProcessed uint64
		searchResp, _ := combiner.GRPCDiff()
		if searchResp != nil && searchResp.Metrics != nil {
			bytesProcessed = searchResp.Metrics.InspectedBytes
		}

		duration := time.Since(start)
		postSLOHook(resp, tenant, bytesProcessed, duration, err)
		logResult(logger, tenant, duration.Seconds(), searchReq, searchResp, err)
		return resp, err
	})
}

// adjusts the limit based on provided config
func adjustLimit(limit, defaultLimit, maxLimit uint32) (uint32, error) {
	if limit == 0 {
		return defaultLimit, nil
	}

	if maxLimit != 0 && limit > maxLimit {
		return 0, fmt.Errorf("limit %d exceeds max limit %d", limit, maxLimit)
	}

	return limit, nil
}

func logResult(logger log.Logger, tenantID string, durationSeconds float64, req *tempopb.SearchRequest, resp *tempopb.SearchResponse, err error) {
	if resp == nil {
		level.Info(logger).Log(
			"msg", "search results - no resp",
			"tenant", tenantID,
			"duration_seconds", durationSeconds,
			"error", err)

		return
	}

	if resp.Metrics == nil {
		level.Info(logger).Log(
			"msg", "search results - no metrics",
			"tenant", tenantID,
			"query", req.Query,
			"range_seconds", req.End-req.Start,
			"duration_seconds", durationSeconds,
			"error", err)
		return
	}

	level.Info(logger).Log(
		"msg", "search results",
		"tenant", tenantID,
		"query", req.Query,
		"range_seconds", req.End-req.Start,
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

func logRequest(logger log.Logger, tenantID string, req *tempopb.SearchRequest) {
	level.Info(logger).Log(
		"msg", "search request",
		"tenant", tenantID,
		"query", req.Query,
		"range_seconds", req.End-req.Start,
		"limit", req.Limit,
		"spans_per_spanset", req.SpansPerSpanSet)
}
