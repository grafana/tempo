package frontend

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/status"
	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"google.golang.org/grpc/codes"
)

//nolint:all //deprecated

// jpe - do we need these
type tagResultsCombinerFactory func(limit int) combiner.Combiner

// jpe - pass limit on all of these
func tagValuesCombinerFactory(limit int) combiner.Combiner {
	return combiner.NewSearchTagValues()
}

func tagValuesV2CombinerFactory(limit int) combiner.Combiner {
	return combiner.NewSearchTagValuesV2()
}

func tagsCombinerFactory(limit int) combiner.Combiner { // jpe can we remove this bridge by requiring the param on combiner.NewSearchTags?
	return combiner.NewSearchTags()
}

func tagsV2CombinerFactory(limit int) combiner.Combiner {
	return combiner.NewSearchTagsV2()
}

// newSearchStreamingGRPCHandler returns a handler that streams results from the HTTP handler
func newTagStreamingGRPCHandler(cfg Config, next pipeline.AsyncRoundTripper[*http.Response], apiPrefix string, o overrides.Interface, logger log.Logger) streamingTagsHandler {
	downstreamPath := path.Join(apiPrefix, api.PathSearch)

	return func(req *tempopb.SearchTagsRequest, srv tempopb.StreamingQuerier_SearchTagsServer) error {
		httpReq, err := api.BuildSearchTagsRequest(&http.Request{
			URL: &url.URL{
				Path: downstreamPath,
			},
			Header:     http.Header{},
			Body:       io.NopCloser(bytes.NewReader([]byte{})),
			RequestURI: buildUpstreamRequestURI(downstreamPath, nil),
		}, req)
		if err != nil {
			level.Error(logger).Log("msg", "search tags: build tags request failed", "err", err)
			return status.Errorf(codes.InvalidArgument, "build tags request failed: %s", err.Error())
		}

		ctx := srv.Context()
		httpReq = httpReq.WithContext(ctx)
		// tenant, _ := user.ExtractOrgID(ctx) // jpe return bad request
		// start := time.Now()

		// limit := o.MaxBytesPerTagValuesQuery(tenant) // jpe do we need a default here? make combiner take limit
		c := combiner.NewTypedSearchTags()
		collector := pipeline.NewGRPCCollector[*tempopb.SearchTagsResponse](next, c, srv.Send)

		// logRequest(logger, tenant, req) - jpe log request
		err = collector.RoundTrip(httpReq)

		return err
	}
}

// newTagHTTPHandler returns a handler that returns a single response from the HTTP handler
func newTagHTTPHandler(next pipeline.AsyncRoundTripper[*http.Response], o overrides.Interface, combinerFn tagResultsCombinerFactory, logger log.Logger) http.RoundTripper {
	return pipeline.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		// tenant, _ := user.ExtractOrgID(req.Context())

		// logRequest(logger, tenant, searchReq)

		// build and use roundtripper
		combiner := combinerFn(0) // jpe - need to use overrides and pass limit
		rt := pipeline.NewHTTPCollector(next, combiner)

		return rt.RoundTrip(req)
	})
}

/*
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
		"msg", "search results - no resp",
		"tenant", tenantID,
		"query", req.Query)

	return
}
*/
