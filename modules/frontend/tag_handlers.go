package frontend

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
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
	"github.com/grafana/tempo/pkg/search"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"google.golang.org/grpc/codes"
)

// regex patterns for tag values endpoints, precompile for performance
var (
	tagNameRegexV1 = regexp.MustCompile(`.*/api/search/tag/([^/]+)/values`)
	tagNameRegexV2 = regexp.MustCompile(`.*/api/v2/search/tag/([^/]+)/values`)
)

//nolint:all //deprecated

// streaming grpc handlers

// newTagsStreamingGRPCHandler returns a handler that streams results from the HTTP handler
func newTagsStreamingGRPCHandler(cfg Config, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], apiPrefix string, o overrides.Interface, logger log.Logger, dataAccessController DataAccessController) streamingTagsHandler {
	downstreamPath := path.Join(apiPrefix, api.PathSearchTags)
	postSLOHook := metadataSLOPostHook(cfg.Search.MetadataSLO)

	return func(req *tempopb.SearchTagsRequest, srv tempopb.StreamingQuerier_SearchTagsServer) error {
		if dataAccessController != nil {
			err := dataAccessController.HandleGRPCTagsReq(srv.Context(), req)
			if err != nil {
				level.Error(logger).Log("msg", "SearchTags streaming: access control handling failed", "err", err)
				return err
			}
		}
		httpReq, tenant, err := buildTagsRequestAndExtractTenant(srv.Context(), req, downstreamPath, logger)
		if err != nil {
			return err
		}

		var finalResponse *tempopb.SearchTagsResponse
		comb := combiner.NewTypedSearchTags(o.MaxBytesPerTagValuesQuery(tenant), req.MaxTagsPerScope, req.StaleValuesThreshold)
		collector := pipeline.NewGRPCCollector(next, cfg.ResponseConsumers, comb, func(res *tempopb.SearchTagsResponse) error {
			finalResponse = res // to get the bytes processed for SLO calculations
			return srv.Send(res)
		})

		// Add intrinsics first so that they aren't dropped by the response size limit
		// NOTE - V1 tag lookup only returns intrinsics when scope is set explicitly.
		if req.Scope == api.ParamScopeIntrinsic {
			err := comb.AddTypedResponse(&tempopb.SearchTagsResponse{
				TagNames: search.GetVirtualIntrinsicValues(),
			})
			if err != nil {
				return err
			}
			// TODO: Exit early here, no need to issue more requests downstream, but some
			//  work needed to ensure things are still logged/metriced correctly.
		}

		start := time.Now()
		logTagsRequest(logger, tenant, "SearchTagsStreaming", req.Scope, req.End-req.Start)
		err = collector.RoundTrip(httpReq)

		duration := time.Since(start)
		bytesProcessed := uint64(0)
		if finalResponse != nil && finalResponse.Metrics != nil {
			bytesProcessed = finalResponse.Metrics.InspectedBytes
		}
		postSLOHook(nil, tenant, bytesProcessed, duration, err)
		logTagsResult(logger, tenant, "SearchTagsStreaming", req.Scope, req.End-req.Start, duration.Seconds(), bytesProcessed, err)

		return err
	}
}

func newTagsV2StreamingGRPCHandler(cfg Config, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], apiPrefix string, o overrides.Interface, logger log.Logger, dataAccessController DataAccessController) streamingTagsV2Handler {
	downstreamPath := path.Join(apiPrefix, api.PathSearchTagsV2)
	postSLOHook := metadataSLOPostHook(cfg.Search.MetadataSLO)

	return func(req *tempopb.SearchTagsRequest, srv tempopb.StreamingQuerier_SearchTagsV2Server) error {
		ctx := srv.Context()

		if dataAccessController != nil {
			err := dataAccessController.HandleGRPCTagsV2Req(ctx, req)
			if err != nil {
				level.Error(logger).Log("msg", "SearchTagsV2 streaming: access control handling failed", "err", err)
				return err
			}
		}
		httpReq, tenant, err := buildTagsRequestAndExtractTenant(ctx, req, downstreamPath, logger)
		if err != nil {
			return err
		}

		var finalResponse *tempopb.SearchTagsV2Response
		comb := combiner.NewTypedSearchTagsV2(o.MaxBytesPerTagValuesQuery(tenant), req.MaxTagsPerScope, req.StaleValuesThreshold)
		collector := pipeline.NewGRPCCollector(next, cfg.ResponseConsumers, comb, func(res *tempopb.SearchTagsV2Response) error {
			finalResponse = res // to get the bytes processed for SLO calculations
			return srv.Send(res)
		})

		// Add intrinsics first so that they aren't dropped by the response size limit
		// NOTE - V2 tag lookup returns intrinsics for both unscoped and explicit scope requests.
		if req.Scope == "" ||
			req.Scope == api.ParamScopeIntrinsic ||
			req.Scope == traceql.AttributeScopeNone.String() {
			err := comb.AddTypedResponse(&tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
					{
						Name: api.ParamScopeIntrinsic,
						Tags: search.GetVirtualIntrinsicValues(),
					},
				},
			})
			if err != nil {
				return err
			}
			// TODO: For intrinsic scope only, exit early here, no need to issue more requests downstream, but some
			//  work needed to ensure things are still logged/metriced correctly.
		}

		start := time.Now()
		logTagsRequest(logger, tenant, "SearchTagsV2Streaming", req.Scope, req.End-req.Start)
		err = collector.RoundTrip(httpReq)

		duration := time.Since(start)
		bytesProcessed := uint64(0)
		if finalResponse != nil && finalResponse.Metrics != nil {
			bytesProcessed = finalResponse.Metrics.InspectedBytes
		}
		postSLOHook(nil, tenant, bytesProcessed, duration, err)
		logTagsResult(logger, tenant, "SearchTagsV2Streaming", req.Scope, req.End-req.Start, duration.Seconds(), bytesProcessed, err)

		return err
	}
}

func newTagValuesStreamingGRPCHandler(cfg Config, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], apiPrefix string, o overrides.Interface, logger log.Logger, dataAccessController DataAccessController) streamingTagValuesHandler {
	postSLOHook := metadataSLOPostHook(cfg.Search.MetadataSLO)

	return func(req *tempopb.SearchTagValuesRequest, srv tempopb.StreamingQuerier_SearchTagValuesServer) error {
		ctx := srv.Context()
		var err error

		if dataAccessController != nil {
			err = dataAccessController.HandleGRPCTagValuesReq(ctx, req)
			if err != nil {
				level.Error(logger).Log("msg", "SearchTagValues streaming: access control handling failed", "err", err)
				return err
			}
		}

		// we have to interpolate the tag name into the path so that when it is routed to the queriers
		// they will parse it correctly. see also the mux.SetUrlVars discussion below.
		pathWithValue := strings.Replace(api.PathSearchTagValues, "{"+api.MuxVarTagName+"}", req.TagName, 1)
		downstreamPath := path.Join(apiPrefix, pathWithValue)

		httpReq, tenant, err := buildTagValuesRequestAndExtractTenant(srv.Context(), req, downstreamPath, logger)
		if err != nil {
			return err
		}

		var finalResponse *tempopb.SearchTagValuesResponse
		comb := combiner.NewTypedSearchTagValues(o.MaxBytesPerTagValuesQuery(tenant), req.MaxTagValues, req.StaleValueThreshold)
		collector := pipeline.NewGRPCCollector(next, cfg.ResponseConsumers, comb, func(res *tempopb.SearchTagValuesResponse) error {
			finalResponse = res // to get the bytes processed for SLO calculations
			return srv.Send(res)
		})

		start := time.Now()
		logTagValuesRequest(logger, tenant, "SearchTagValuesStreaming", req.TagName, req.Query, req.End-req.Start)
		err = collector.RoundTrip(httpReq)

		duration := time.Since(start)
		bytesProcessed := uint64(0)
		if finalResponse != nil && finalResponse.Metrics != nil {
			bytesProcessed = finalResponse.Metrics.InspectedBytes
		}
		postSLOHook(nil, tenant, bytesProcessed, duration, err)
		logTagValuesResult(logger, tenant, "SearchTagValuesStreaming", req.TagName, req.Query, req.End-req.Start, duration.Seconds(), bytesProcessed, err)

		return err
	}
}

func newTagValuesV2StreamingGRPCHandler(cfg Config, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], apiPrefix string, o overrides.Interface, logger log.Logger, dataAccessController DataAccessController) streamingTagValuesV2Handler {
	postSLOHook := metadataSLOPostHook(cfg.Search.MetadataSLO)

	return func(req *tempopb.SearchTagValuesRequest, srv tempopb.StreamingQuerier_SearchTagValuesV2Server) error {
		ctx := srv.Context()

		if dataAccessController != nil {
			err := dataAccessController.HandleGRPCTagValuesV2Req(ctx, req)
			if err != nil {
				level.Error(logger).Log("msg", "SearchTagValues streaming: access control handling failed", "err", err)
				return err
			}
		}
		// we have to interpolate the tag name into the path so that when it is routed to the queriers
		// they will parse it correctly. see also the mux.SetUrlVars discussion below.
		pathWithValue := strings.Replace(api.PathSearchTagValuesV2, "{"+api.MuxVarTagName+"}", req.TagName, 1)
		downstreamPath := path.Join(apiPrefix, pathWithValue)

		httpReq, tenant, err := buildTagValuesRequestAndExtractTenant(ctx, req, downstreamPath, logger)
		if err != nil {
			return err
		}

		var finalResponse *tempopb.SearchTagValuesV2Response
		comb := combiner.NewTypedSearchTagValuesV2(o.MaxBytesPerTagValuesQuery(tenant), req.MaxTagValues, req.StaleValueThreshold)
		collector := pipeline.NewGRPCCollector(next, cfg.ResponseConsumers, comb, func(res *tempopb.SearchTagValuesV2Response) error {
			finalResponse = res // to get the bytes processed for SLO calculations
			return srv.Send(res)
		})

		start := time.Now()
		logTagValuesRequest(logger, tenant, "SearchTagValuesV2Streaming", req.TagName, req.Query, req.End-req.Start)
		err = collector.RoundTrip(httpReq)

		duration := time.Since(start)
		bytesProcessed := uint64(0)
		if finalResponse != nil && finalResponse.Metrics != nil {
			bytesProcessed = finalResponse.Metrics.InspectedBytes
		}
		postSLOHook(nil, tenant, bytesProcessed, duration, err)
		logTagValuesResult(logger, tenant, "SearchTagValuesV2Streaming", req.TagName, req.Query, req.End-req.Start, duration.Seconds(), bytesProcessed, err)

		return err
	}
}

// HTTP Handlers
func newTagsHTTPHandler(cfg Config, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], o overrides.Interface, logger log.Logger, dataAccessController DataAccessController) http.RoundTripper {
	postSLOHook := metadataSLOPostHook(cfg.Search.MetadataSLO)

	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		tenant, errResp := extractTenant(req, logger)
		if errResp != nil {
			return errResp, nil
		}

		if dataAccessController != nil {
			if err := dataAccessController.HandleHTTPTagsReq(req); err != nil {
				level.Error(logger).Log("msg", "SearchTags http: access control handling failed", "err", err)
				return httpInvalidRequest(err), nil
			}
		}

		scope, _, rangeDur, maxTagsPerScope, staleValueThreshold := parseParams(req)
		// build and use round tripper
		comb := combiner.NewTypedSearchTags(o.MaxBytesPerTagValuesQuery(tenant), maxTagsPerScope, staleValueThreshold)

		// Add intrinsics first so that they aren't dropped by the response size limit
		// NOTE - V1 tag lookup only returns intrinsics when scope is set explicitly.
		if scope == api.ParamScopeIntrinsic {
			err := comb.AddTypedResponse(&tempopb.SearchTagsResponse{
				TagNames: search.GetVirtualIntrinsicValues(),
			})
			if err != nil {
				return nil, err
			}
			// TODO: Exit early here, no need to issue more requests downstream, but some
			//  work needed to ensure things are still logged/metriced correctly.
		}

		rt := pipeline.NewHTTPCollector(next, cfg.ResponseConsumers, comb)
		start := time.Now()
		logTagsRequest(logger, tenant, "SearchTags", scope, rangeDur)

		resp, err := rt.RoundTrip(req)

		// ask for the typed diff and use that for the SLO hook. it will have up-to-date metrics
		var bytesProcessed uint64
		searchResp, _ := comb.GRPCDiff()
		if searchResp != nil && searchResp.Metrics != nil {
			bytesProcessed = searchResp.Metrics.InspectedBytes
		}

		duration := time.Since(start)
		postSLOHook(resp, tenant, bytesProcessed, duration, err)
		logTagsResult(logger, tenant, "SearchTags", scope, rangeDur, duration.Seconds(), bytesProcessed, err)

		return resp, err
	})
}

func newTagsV2HTTPHandler(cfg Config, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], o overrides.Interface, logger log.Logger, dataAccessController DataAccessController) http.RoundTripper {
	postSLOHook := metadataSLOPostHook(cfg.Search.MetadataSLO)

	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		tenant, errResp := extractTenant(req, logger)
		if errResp != nil {
			return errResp, nil
		}

		if dataAccessController != nil {
			if err := dataAccessController.HandleHTTPTagsV2Req(req); err != nil {
				level.Error(logger).Log("msg", "SearchTagsV2 http: access control handling failed", "err", err)
				return httpInvalidRequest(err), nil
			}
		}

		scope, _, rangeDur, maxTagsPerScope, staleValueThreshold := parseParams(req)
		// build and use round tripper
		comb := combiner.NewTypedSearchTagsV2(o.MaxBytesPerTagValuesQuery(tenant), maxTagsPerScope, staleValueThreshold)

		// Add intrinsics first so that they aren't dropped by the response size limit
		// NOTE - V2 tag lookup returns intrinsics for both unscoped and explicit scope requests.
		if scope == "" ||
			scope == api.ParamScopeIntrinsic ||
			scope == traceql.AttributeScopeNone.String() {
			err := comb.AddTypedResponse(&tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
					{
						Name: api.ParamScopeIntrinsic,
						Tags: search.GetVirtualIntrinsicValues(),
					},
				},
			})
			if err != nil {
				return nil, err
			}
			// TODO: For intrinsic scope only, exit early here, no need to issue more requests downstream, but some
			//  work needed to ensure things are still logged/metriced correctly.
		}

		rt := pipeline.NewHTTPCollector(next, cfg.ResponseConsumers, comb)
		start := time.Now()
		logTagsRequest(logger, tenant, "SearchTagsV2", scope, rangeDur)

		resp, err := rt.RoundTrip(req)

		// ask for the typed diff and use that for the SLO hook. it will have up-to-date metrics
		var bytesProcessed uint64
		searchResp, _ := comb.GRPCDiff()
		if searchResp != nil && searchResp.Metrics != nil {
			bytesProcessed = searchResp.Metrics.InspectedBytes
		}

		duration := time.Since(start)
		postSLOHook(resp, tenant, bytesProcessed, duration, err)
		logTagsResult(logger, tenant, "SearchTagsV2", scope, rangeDur, duration.Seconds(), bytesProcessed, err)

		return resp, err
	})
}

func newTagValuesHTTPHandler(cfg Config, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], o overrides.Interface, logger log.Logger, dataAccessController DataAccessController) http.RoundTripper {
	postSLOHook := metadataSLOPostHook(cfg.Search.MetadataSLO)

	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		tenant, errResp := extractTenant(req, logger)
		if errResp != nil {
			return errResp, nil
		}

		if dataAccessController != nil {
			if err := dataAccessController.HandleHTTPTagValuesReq(req); err != nil {
				level.Error(logger).Log("msg", "SearchTagValues http: access control handling failed", "err", err)
				return httpInvalidRequest(err), nil
			}
		}

		_, query, rangeDur, maxTagsValues, staleValueThreshold := parseParams(req)
		tagName := extractTagName(req.URL.Path, tagNameRegexV1)

		// build and use round tripper
		comb := combiner.NewTypedSearchTagValues(o.MaxBytesPerTagValuesQuery(tenant), maxTagsValues, staleValueThreshold)
		rt := pipeline.NewHTTPCollector(next, cfg.ResponseConsumers, comb)
		start := time.Now()
		logTagValuesRequest(logger, tenant, "SearchTagValues", tagName, query, rangeDur)

		resp, err := rt.RoundTrip(req)

		// ask for the typed diff and use that for the SLO hook. it will have up-to-date metrics
		var bytesProcessed uint64
		searchResp, _ := comb.GRPCDiff()
		if searchResp != nil && searchResp.Metrics != nil {
			bytesProcessed = searchResp.Metrics.InspectedBytes
		}

		duration := time.Since(start)
		postSLOHook(resp, tenant, bytesProcessed, duration, err)
		logTagValuesResult(logger, tenant, "SearchTagValues", tagName, query, rangeDur, duration.Seconds(), bytesProcessed, err)

		return resp, err
	})
}

func newTagValuesV2HTTPHandler(cfg Config, next pipeline.AsyncRoundTripper[combiner.PipelineResponse], o overrides.Interface, logger log.Logger, dataAccessController DataAccessController) http.RoundTripper {
	postSLOHook := metadataSLOPostHook(cfg.Search.MetadataSLO)

	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		tenant, errResp := extractTenant(req, logger)
		if errResp != nil {
			return errResp, nil
		}

		if dataAccessController != nil {
			if err := dataAccessController.HandleHTTPTagValuesV2Req(req); err != nil {
				level.Error(logger).Log("msg", "SearchTagValuesV2 http: access control handling failed", "err", err)
				return httpInvalidRequest(err), nil
			}
		}

		_, query, rangeDur, maxTagsValues, staleValueThreshold := parseParams(req)
		tagName := extractTagName(req.URL.Path, tagNameRegexV2)

		// build and use round tripper
		comb := combiner.NewTypedSearchTagValuesV2(o.MaxBytesPerTagValuesQuery(tenant), maxTagsValues, staleValueThreshold)
		rt := pipeline.NewHTTPCollector(next, cfg.ResponseConsumers, comb)
		start := time.Now()
		logTagValuesRequest(logger, tenant, "SearchTagValuesV2", tagName, query, rangeDur)

		resp, err := rt.RoundTrip(req)

		// ask for the typed diff and use that for the SLO hook. it will have up-to-date metrics
		var bytesProcessed uint64
		searchResp, _ := comb.GRPCDiff()
		if searchResp != nil && searchResp.Metrics != nil {
			bytesProcessed = searchResp.Metrics.InspectedBytes
		}

		duration := time.Since(start)
		postSLOHook(resp, tenant, bytesProcessed, duration, err)
		logTagValuesResult(logger, tenant, "SearchTagValuesV2", tagName, query, rangeDur, duration.Seconds(), bytesProcessed, err)

		return resp, err
	})
}

// helpers
func buildTagsRequestAndExtractTenant(ctx context.Context, req *tempopb.SearchTagsRequest, downstreamPath string, logger log.Logger) (*http.Request, string, error) {
	headers := headersFromGrpcContext(ctx)

	httpReq, err := api.BuildSearchTagsRequest(&http.Request{
		URL:    &url.URL{Path: downstreamPath},
		Header: headers,
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
	headers := headersFromGrpcContext(ctx)

	httpReq, err := api.BuildSearchTagValuesRequest(&http.Request{
		URL:    &url.URL{Path: downstreamPath},
		Header: headers,
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

func logTagsRequest(logger log.Logger, tenantID, handler, scope string, rangeSeconds uint32) {
	level.Info(logger).Log(
		"msg", "search tag request",
		"tenant", tenantID,
		"handler", handler,
		"scope", scope,
		"range_seconds", rangeSeconds)
}

func logTagsResult(logger log.Logger, tenantID, handler, scope string, rangeSeconds uint32, durationSeconds float64, inspectedBytes uint64, err error) {
	level.Info(logger).Log(
		"msg", "search tag response",
		"tenant", tenantID,
		"handler", handler,
		"scope", scope,
		"range_seconds", rangeSeconds,
		"duration_seconds", durationSeconds,
		"inspected_bytes", inspectedBytes,
		"request_throughput", float64(inspectedBytes)/durationSeconds,
		"error", err)
}

func logTagValuesRequest(logger log.Logger, tenantID, handler, tagName, query string, rangeSeconds uint32) {
	level.Info(logger).Log(
		"msg", "search tag values request",
		"tenant", tenantID,
		"handler", handler,
		"tag", tagName,
		"query", query,
		"range_seconds", rangeSeconds)
}

func logTagValuesResult(logger log.Logger, tenantID, handler, tagName, query string, rangeSeconds uint32, durationSeconds float64, inspectedBytes uint64, err error) {
	level.Info(logger).Log(
		"msg", "search tag values response",
		"tenant", tenantID,
		"handler", handler,
		"tag", tagName,
		"query", query,
		"range_seconds", rangeSeconds,
		"duration_seconds", durationSeconds,
		"inspected_bytes", inspectedBytes,
		"request_throughput", float64(inspectedBytes)/durationSeconds,
		"error", err)
}

// parseParams parses optional 'start', 'end', 'scope', and 'q' params from a http.Request
// returns scope, query and duration (end - start). returns "", and 0 if these params are invalid or absent
func parseParams(req *http.Request) (scope string, q string, duration uint32, limit uint32, staleValueThreshold uint32) {
	query := req.URL.Query()
	scope = query.Get("scope")
	q = query.Get("q")
	// ignore errors, we default to 0 as params are not always present.
	start, _ := strconv.ParseInt(query.Get("start"), 10, 64)
	end, _ := strconv.ParseInt(query.Get("end"), 10, 64)
	maxItems, err := strconv.ParseUint(query.Get("limit"), 10, 32)
	if err != nil {
		maxItems = 0
	}
	maxStaleValues, err := strconv.ParseUint(query.Get("maxStaleValues"), 10, 32)
	if err != nil {
		maxStaleValues = 0
	}
	// duration only makes sense if start and end are present and end is greater than start
	if start > 0 && end > 0 && end > start {
		duration = uint32(end - start)
	}
	return scope, q, duration, uint32(maxItems), uint32(maxStaleValues)
}

// extractTagName extracts the tagName based on the provided regex pattern
func extractTagName(path string, pattern *regexp.Regexp) string {
	matches := pattern.FindStringSubmatch(path)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
