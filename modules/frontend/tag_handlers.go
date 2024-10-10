package frontend

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/status"
	"github.com/gorilla/mux"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"google.golang.org/grpc/codes"
)

//nolint:all //deprecated

// streaming grpc handlers

// newTagsStreamingGRPCHandler returns a handler that streams results from the HTTP handler
func newTagsStreamingGRPCHandler(cfg Config, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], apiPrefix string, o overrides.Interface, logger log.Logger) streamingTagsHandler {
	downstreamPath := path.Join(apiPrefix, api.PathSearchTags)
	postSLOHook := metadataSLOPostHook(cfg.Search.MetadataSLO)

	return func(req *tempopb.SearchTagsRequest, srv tempopb.StreamingQuerier_SearchTagsServer) error {
		httpReq, tenant, err := buildTagsRequestAndExtractTenant(srv.Context(), req, downstreamPath, logger)
		if err != nil {
			return err
		}
		prepareRequestForQueriers(httpReq, tenant)

		var finalResponse *tempopb.SearchTagsResponse
		comb := combiner.NewTypedSearchTags(o.MaxBytesPerTagValuesQuery(tenant))
		collector := pipeline.NewGRPCCollector[*tempopb.SearchTagsResponse](next, cfg.ResponseConsumers, comb, func(res *tempopb.SearchTagsResponse) error {
			finalResponse = res // to get the bytes processed for SLO calculations
			return srv.Send(res)
		})

		start := time.Now()
		logTagsRequest(logger, tenant, "SearchTagsStreaming", req)
		err = collector.RoundTrip(httpReq)

		duration := time.Since(start)
		bytesProcessed := uint64(0)
		if finalResponse != nil && finalResponse.Metrics != nil {
			bytesProcessed = finalResponse.Metrics.InspectedBytes
		}
		postSLOHook(nil, tenant, bytesProcessed, duration, err)
		logTagsResult(logger, tenant, "SearchTagsStreaming", duration.Seconds(), bytesProcessed, req, err)

		return err
	}
}

// newTagsV2StreamingGRPCHandler returns a handler that streams results from the HTTP handler
func newTagsV2StreamingGRPCHandler(cfg Config, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], apiPrefix string, o overrides.Interface, logger log.Logger) streamingTagsV2Handler {
	downstreamPath := path.Join(apiPrefix, api.PathSearchTagsV2)
	postSLOHook := metadataSLOPostHook(cfg.Search.MetadataSLO)

	return func(req *tempopb.SearchTagsRequest, srv tempopb.StreamingQuerier_SearchTagsV2Server) error {
		httpReq, tenant, err := buildTagsRequestAndExtractTenant(srv.Context(), req, downstreamPath, logger)
		if err != nil {
			return err
		}
		prepareRequestForQueriers(httpReq, tenant)

		var finalResponse *tempopb.SearchTagsV2Response
		comb := combiner.NewTypedSearchTagsV2(o.MaxBytesPerTagValuesQuery(tenant))
		collector := pipeline.NewGRPCCollector[*tempopb.SearchTagsV2Response](next, cfg.ResponseConsumers, comb, func(res *tempopb.SearchTagsV2Response) error {
			finalResponse = res // to get the bytes processed for SLO calculations
			return srv.Send(res)
		})

		start := time.Now()
		logTagsRequest(logger, tenant, "SearchTagsV2Streaming", req)
		err = collector.RoundTrip(httpReq)

		duration := time.Since(start)
		bytesProcessed := uint64(0)
		if finalResponse != nil && finalResponse.Metrics != nil {
			bytesProcessed = finalResponse.Metrics.InspectedBytes
		}
		postSLOHook(nil, tenant, bytesProcessed, duration, err)
		logTagsResult(logger, tenant, "SearchTagsV2Streaming", duration.Seconds(), bytesProcessed, req, err)

		return err
	}
}

func newTagValuesStreamingGRPCHandler(cfg Config, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], apiPrefix string, o overrides.Interface, logger log.Logger) streamingTagValuesHandler {
	postSLOHook := metadataSLOPostHook(cfg.Search.MetadataSLO)

	return func(req *tempopb.SearchTagValuesRequest, srv tempopb.StreamingQuerier_SearchTagValuesServer) error {
		// we have to interpolate the tag name into the path so that when it is routed to the queriers
		// they will parse it correctly. see also the mux.SetUrlVars discussion below.
		pathWithValue := strings.Replace(api.PathSearchTagValues, "{"+api.MuxVarTagName+"}", req.TagName, 1)
		downstreamPath := path.Join(apiPrefix, pathWithValue)

		httpReq, tenant, err := buildTagValuesRequestAndExtractTenant(srv.Context(), req, downstreamPath, logger)
		if err != nil {
			return err
		}
		prepareRequestForQueriers(httpReq, tenant)

		var finalResponse *tempopb.SearchTagValuesResponse
		comb := combiner.NewTypedSearchTagValues(o.MaxBytesPerTagValuesQuery(tenant))
		collector := pipeline.NewGRPCCollector[*tempopb.SearchTagValuesResponse](next, cfg.ResponseConsumers, comb, func(res *tempopb.SearchTagValuesResponse) error {
			finalResponse = res // to get the bytes processed for SLO calculations
			return srv.Send(res)
		})

		start := time.Now()
		logTagValuesRequest(logger, tenant, "SearchTagValuesStreaming", req)
		err = collector.RoundTrip(httpReq)

		duration := time.Since(start)
		bytesProcessed := uint64(0)
		if finalResponse != nil && finalResponse.Metrics != nil {
			bytesProcessed = finalResponse.Metrics.InspectedBytes
		}
		postSLOHook(nil, tenant, bytesProcessed, duration, err)
		logTagValuesResult(logger, tenant, "SearchTagValuesStreaming", duration.Seconds(), bytesProcessed, req, err)

		return err
	}
}

func newTagValuesV2StreamingGRPCHandler(cfg Config, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], apiPrefix string, o overrides.Interface, logger log.Logger) streamingTagValuesV2Handler {
	postSLOHook := metadataSLOPostHook(cfg.Search.MetadataSLO)

	return func(req *tempopb.SearchTagValuesRequest, srv tempopb.StreamingQuerier_SearchTagValuesV2Server) error {
		// we have to interpolate the tag name into the path so that when it is routed to the queriers
		// they will parse it correctly. see also the mux.SetUrlVars discussion below.
		pathWithValue := strings.Replace(api.PathSearchTagValuesV2, "{"+api.MuxVarTagName+"}", req.TagName, 1)
		downstreamPath := path.Join(apiPrefix, pathWithValue)

		httpReq, tenant, err := buildTagValuesRequestAndExtractTenant(srv.Context(), req, downstreamPath, logger)
		if err != nil {
			return err
		}
		prepareRequestForQueriers(httpReq, tenant)

		var finalResponse *tempopb.SearchTagValuesV2Response
		comb := combiner.NewTypedSearchTagValuesV2(o.MaxBytesPerTagValuesQuery(tenant))
		collector := pipeline.NewGRPCCollector[*tempopb.SearchTagValuesV2Response](next, cfg.ResponseConsumers, comb, func(res *tempopb.SearchTagValuesV2Response) error {
			finalResponse = res // to get the bytes processed for SLO calculations
			return srv.Send(res)
		})

		start := time.Now()
		logTagValuesRequest(logger, tenant, "SearchTagValuesV2Streaming", req)
		err = collector.RoundTrip(httpReq)

		duration := time.Since(start)
		bytesProcessed := uint64(0)
		if finalResponse != nil && finalResponse.Metrics != nil {
			bytesProcessed = finalResponse.Metrics.InspectedBytes
		}
		postSLOHook(nil, tenant, bytesProcessed, duration, err)
		logTagValuesResult(logger, tenant, "SearchTagValuesV2Streaming", duration.Seconds(), bytesProcessed, req, err)

		return err
	}
}

// HTTP Handlers
func newTagsHTTPHandler(cfg Config, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], o overrides.Interface, logger log.Logger) http.RoundTripper {
	postSLOHook := metadataSLOPostHook(cfg.Search.MetadataSLO)

	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		// if error is not nil, return error Response but suppress the error
		tenant, errResp, err := extractTenantWithErrorResp(req, logger)
		if err != nil {
			return errResp, nil
		}

		// build and use round tripper
		comb := combiner.NewTypedSearchTags(o.MaxBytesPerTagValuesQuery(tenant))
		rt := pipeline.NewHTTPCollector(next, cfg.ResponseConsumers, comb)
		start := time.Now()

		resp, err := rt.RoundTrip(req)
		// ask for the typed diff and use that for the SLO hook. it will have up-to-date metrics
		var bytesProcessed uint64
		searchResp, _ := comb.GRPCDiff()
		if searchResp != nil && searchResp.Metrics != nil {
			bytesProcessed = searchResp.Metrics.InspectedBytes
		}

		duration := time.Since(start)
		postSLOHook(resp, tenant, bytesProcessed, duration, err)
		logHTTPResult(logger, tenant, "SearchTags", req.URL.Path, duration.Seconds(), bytesProcessed, err)

		return resp, err
	})
}

func newTagsV2HTTPHandler(cfg Config, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], o overrides.Interface, logger log.Logger) http.RoundTripper {
	postSLOHook := metadataSLOPostHook(cfg.Search.MetadataSLO)

	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		// if error is not nil, return error Response but suppress the error
		tenant, errResp, err := extractTenantWithErrorResp(req, logger)
		if err != nil {
			return errResp, nil
		}

		// build and use round tripper
		comb := combiner.NewTypedSearchTagsV2(o.MaxBytesPerTagValuesQuery(tenant))
		rt := pipeline.NewHTTPCollector(next, cfg.ResponseConsumers, comb)
		start := time.Now()

		resp, err := rt.RoundTrip(req)
		// ask for the typed diff and use that for the SLO hook. it will have up-to-date metrics
		var bytesProcessed uint64
		searchResp, _ := comb.GRPCDiff()
		if searchResp != nil && searchResp.Metrics != nil {
			bytesProcessed = searchResp.Metrics.InspectedBytes
		}

		duration := time.Since(start)
		postSLOHook(resp, tenant, bytesProcessed, duration, err)
		logHTTPResult(logger, tenant, "SearchTagsV2", req.URL.Path, duration.Seconds(), bytesProcessed, err)

		return resp, err
	})
}

func newTagValuesHTTPHandler(cfg Config, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], o overrides.Interface, logger log.Logger) http.RoundTripper {
	postSLOHook := metadataSLOPostHook(cfg.Search.MetadataSLO)

	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		// if error is not nil, return error Response but suppress the error
		tenant, errResp, err := extractTenantWithErrorResp(req, logger)
		if err != nil {
			return errResp, nil
		}

		// build and use round tripper
		comb := combiner.NewTypedSearchTagValues(o.MaxBytesPerTagValuesQuery(tenant))
		rt := pipeline.NewHTTPCollector(next, cfg.ResponseConsumers, comb)
		start := time.Now()

		resp, err := rt.RoundTrip(req)
		// ask for the typed diff and use that for the SLO hook. it will have up-to-date metrics
		var bytesProcessed uint64
		searchResp, _ := comb.GRPCDiff()
		if searchResp != nil && searchResp.Metrics != nil {
			bytesProcessed = searchResp.Metrics.InspectedBytes
		}

		duration := time.Since(start)
		postSLOHook(resp, tenant, bytesProcessed, duration, err)
		logHTTPResult(logger, tenant, "SearchTagValues", req.URL.Path, duration.Seconds(), bytesProcessed, err)

		return resp, err
	})
}

func newTagValuesV2HTTPHandler(cfg Config, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], o overrides.Interface, logger log.Logger) http.RoundTripper {
	postSLOHook := metadataSLOPostHook(cfg.Search.MetadataSLO)

	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		// if error is not nil, return error Response but suppress the error
		tenant, errResp, err := extractTenantWithErrorResp(req, logger)
		if err != nil {
			return errResp, nil
		}

		// build and use round tripper
		comb := combiner.NewTypedSearchTagValuesV2(o.MaxBytesPerTagValuesQuery(tenant))
		rt := pipeline.NewHTTPCollector(next, cfg.ResponseConsumers, comb)
		start := time.Now()

		resp, err := rt.RoundTrip(req)

		// ask for the typed diff and use that for the SLO hook. it will have up-to-date metrics
		var bytesProcessed uint64
		searchResp, _ := comb.GRPCDiff()
		if searchResp != nil && searchResp.Metrics != nil {
			bytesProcessed = searchResp.Metrics.InspectedBytes
		}

		duration := time.Since(start)
		postSLOHook(resp, tenant, bytesProcessed, duration, err)

		logHTTPResult(logger, tenant, "SearchTagValuesV2", req.URL.Path, duration.Seconds(), bytesProcessed, err)
		return resp, err
	})
}

// helpers
func extractTenantWithErrorResp(req *http.Request, logger log.Logger) (string, *http.Response, error) {
	tenant, err := user.ExtractOrgID(req.Context())
	if err != nil {
		level.Error(logger).Log("msg", "tags failed to extract orgid", "err", err)
		return "", &http.Response{
			StatusCode: http.StatusBadRequest,
			Status:     http.StatusText(http.StatusBadRequest),
			Body:       io.NopCloser(strings.NewReader(err.Error())),
		}, err
	}
	return tenant, nil, err
}

func buildTagsRequestAndExtractTenant(ctx context.Context, req *tempopb.SearchTagsRequest, downstreamPath string, logger log.Logger) (*http.Request, string, error) {
	httpReq, err := api.BuildSearchTagsRequest(&http.Request{
		URL:    &url.URL{Path: downstreamPath},
		Header: http.Header{},
		Body:   io.NopCloser(bytes.NewReader([]byte{})),
	}, req)
	if err != nil {
		_ = level.Error(logger).Log("msg", "search tags: build tags request failed", "err", err)
		return nil, "", status.Errorf(codes.InvalidArgument, "build tags request failed: %s", err.Error())
	}
	httpReq = httpReq.WithContext(ctx)

	tenant, err := user.ExtractOrgID(ctx)
	if err != nil {
		_ = level.Error(logger).Log("msg", "search tags: ", "err", err)
		return nil, "", status.Error(codes.InvalidArgument, err.Error())
	}

	return httpReq, tenant, nil
}

func buildTagValuesRequestAndExtractTenant(ctx context.Context, req *tempopb.SearchTagValuesRequest, downstreamPath string, logger log.Logger) (*http.Request, string, error) {
	httpReq, err := api.BuildSearchTagValuesRequest(&http.Request{
		URL:    &url.URL{Path: downstreamPath},
		Header: http.Header{},
		Body:   io.NopCloser(bytes.NewReader([]byte{})),
	}, req)
	if err != nil {
		_ = level.Error(logger).Log("msg", "search tag values: build tags values request failed", "err", err)
		return nil, "", status.Errorf(codes.InvalidArgument, "build tag values request failed: %s", err.Error())
	}
	httpReq = httpReq.WithContext(ctx)

	// the functions that parse a http request in the api package expect the tagName
	// to be parsed out of the path so we're injecting it here. this is a hack and
	// could be removed if the pipeline were swapped to be a proto.Message pipeline instead of
	// an *http.Request pipeline.
	httpReq = mux.SetURLVars(httpReq, map[string]string{api.MuxVarTagName: req.TagName})

	tenant, err := user.ExtractOrgID(ctx)
	if err != nil {
		_ = level.Error(logger).Log("msg", "search tag values: ", "err", err)
		return nil, "", status.Error(codes.InvalidArgument, err.Error())
	}

	return httpReq, tenant, nil
}

func logHTTPResult(logger log.Logger, tenantID, handler, path string, durationSeconds float64, inspectedBytes uint64, err error) {
	level.Info(logger).Log(
		"msg", "metadata response result",
		"tenant", tenantID,
		"handler", handler,
		"path", path,
		"duration_seconds", durationSeconds,
		"inspected_bytes", inspectedBytes,
		"err", err)
}

func logTagsResult(logger log.Logger, tenantID, handler string, durationSeconds float64, inspectedBytes uint64, req *tempopb.SearchTagsRequest, err error) {
	level.Info(logger).Log(
		"msg", "search tag results",
		"tenant", tenantID,
		"handler", handler,
		"scope", req.Scope,
		"range_seconds", req.End-req.Start,
		"duration_seconds", durationSeconds,
		"inspected_bytes", inspectedBytes,
		"error", err)
}

func logTagsRequest(logger log.Logger, tenantID, handler string, req *tempopb.SearchTagsRequest) {
	level.Info(logger).Log(
		"msg", "search tag request",
		"tenant", tenantID,
		"handler", handler,
		"scope", req.Scope,
		"range_seconds", req.End-req.Start)
}

func logTagValuesResult(logger log.Logger, tenantID, handler string, durationSeconds float64, inspectedBytes uint64, req *tempopb.SearchTagValuesRequest, err error) {
	level.Info(logger).Log(
		"msg", "search tag results",
		"tenant", tenantID,
		"handler", handler,
		"tag", req.TagName,
		"query", req.Query,
		"range_seconds", req.End-req.Start,
		"duration_seconds", durationSeconds,
		"inspected_bytes", inspectedBytes,
		"error", err)
}

func logTagValuesRequest(logger log.Logger, tenantID, handler string, req *tempopb.SearchTagValuesRequest) {
	level.Info(logger).Log(
		"msg", "search tag request",
		"tenant", tenantID,
		"handler", handler,
		"tag", req.TagName,
		"query", req.Query,
		"range_seconds", req.End-req.Start)
}
