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
	"github.com/gogo/protobuf/proto"
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

// newTagStreamingGRPCHandler returns a handler that streams results from the HTTP handler
func newTagStreamingGRPCHandler(cfg Config, next pipeline.AsyncRoundTripper[*http.Response], apiPrefix string, o overrides.Interface, logger log.Logger) streamingTagsHandler {
	downstreamPath := path.Join(apiPrefix, api.PathSearchTags)

	return func(req *tempopb.SearchTagsRequest, srv tempopb.StreamingQuerier_SearchTagsServer) error {
		return streamingTags(srv.Context(), next, req, downstreamPath, "", api.BuildSearchTagsRequest, srv.Send, combiner.NewTypedSearchTags, logTagsRequest, logTagsResult, logger)
	}
}

// newTagStreamingGRPCHandler returns a handler that streams results from the HTTP handler
func newTagV2StreamingGRPCHandler(cfg Config, next pipeline.AsyncRoundTripper[*http.Response], apiPrefix string, o overrides.Interface, logger log.Logger) streamingTagsV2Handler {
	downstreamPath := path.Join(apiPrefix, api.PathSearchTagsV2)

	return func(req *tempopb.SearchTagsRequest, srv tempopb.StreamingQuerier_SearchTagsV2Server) error {
		return streamingTags(srv.Context(), next, req, downstreamPath, "", api.BuildSearchTagsRequest, srv.Send, combiner.NewTypedSearchTagsV2, logTagsRequest, logTagsResult, logger)
	}
}

func newTagValuesStreamingGRPCHandler(cfg Config, next pipeline.AsyncRoundTripper[*http.Response], apiPrefix string, o overrides.Interface, logger log.Logger) streamingTagValuesHandler {
	return func(req *tempopb.SearchTagValuesRequest, srv tempopb.StreamingQuerier_SearchTagValuesServer) error {
		// we have to interpolate the tag name into the path so that when it is routed to the queriers
		// they will parse it correctly. see also the mux.SetUrlVars discussion below.
		pathWithValue := strings.Replace(api.PathSearchTagValues, "{"+api.MuxVarTagName+"}", req.TagName, 1)
		downstreamPath := path.Join(apiPrefix, pathWithValue)

		return streamingTags(srv.Context(), next, req, downstreamPath, req.TagName, api.BuildSearchTagValuesRequest, srv.Send, combiner.NewTypedSearchTagValues, logTagValuesRequest, logTagValuesResult, logger)
	}
}

func newTagValuesV2StreamingGRPCHandler(cfg Config, next pipeline.AsyncRoundTripper[*http.Response], apiPrefix string, o overrides.Interface, logger log.Logger) streamingTagValuesV2Handler {
	return func(req *tempopb.SearchTagValuesRequest, srv tempopb.StreamingQuerier_SearchTagValuesV2Server) error {
		// we have to interpolate the tag name into the path so that when it is routed to the queriers
		// they will parse it correctly. see also the mux.SetUrlVars discussion below.
		pathWithValue := strings.Replace(api.PathSearchTagValuesV2, "{"+api.MuxVarTagName+"}", req.TagName, 1)
		downstreamPath := path.Join(apiPrefix, pathWithValue)

		return streamingTags(srv.Context(), next, req, downstreamPath, req.TagName, api.BuildSearchTagValuesRequest, srv.Send, combiner.NewTypedSearchTagValuesV2, logTagValuesRequest, logTagValuesResult, logger)
	}
}

// streamingTags abstracts the boilerplate for streaming tags and tag values
func streamingTags[TReq proto.Message, TResp proto.Message](ctx context.Context,
	next pipeline.AsyncRoundTripper[*http.Response],
	req TReq,
	downstreamPath string,
	tagName string,
	fnBuild func(*http.Request, TReq) (*http.Request, error),
	fnSend func(TResp) error,
	fnCombiner func() combiner.GRPCCombiner[TResp],
	logRequest func(log.Logger, string, TReq),
	logResult func(log.Logger, string, float64, TReq, error),
	logger log.Logger) error {

	httpReq, err := fnBuild(&http.Request{
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

	httpReq = httpReq.WithContext(ctx)

	if tagName != "" {
		// the functions that parse an http request in the api package expect the tagName
		// to be parsed out of the path so we're injecting it here. this is a hack and
		// could be removed if the pipeline were swapped to be a proto.Message pipeline instead of
		// an *http.Request pipeline.
		httpReq = mux.SetURLVars(httpReq, map[string]string{api.MuxVarTagName: tagName})
	}
	tenant, _ := user.ExtractOrgID(ctx) // jpe return bad request

	// limit := o.MaxBytesPerTagValuesQuery(tenant) // jpe do we need a default here? make combiner take limit
	c := fnCombiner() // jpe limits!
	collector := pipeline.NewGRPCCollector[TResp](next, c, fnSend)

	start := time.Now()
	logRequest(logger, tenant, req)
	err = collector.RoundTrip(httpReq)
	logResult(logger, tenant, time.Since(start).Seconds(), req, err)

	return err
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

func logTagsResult(logger log.Logger, tenantID string, durationSeconds float64, req *tempopb.SearchTagsRequest, err error) {
	level.Info(logger).Log(
		"msg", "search tag results",
		"tenant", tenantID,
		"scope", req.Scope,
		"range_seconds", req.End-req.Start,
		"duration_seconds", durationSeconds,
		"error", err)
}

func logTagsRequest(logger log.Logger, tenantID string, req *tempopb.SearchTagsRequest) {
	level.Info(logger).Log(
		"msg", "search tag request",
		"tenant", tenantID,
		"scope", req.Scope,
		"range_seconds", req.End-req.Start)
}

func logTagValuesResult(logger log.Logger, tenantID string, durationSeconds float64, req *tempopb.SearchTagValuesRequest, err error) {
	level.Info(logger).Log(
		"msg", "search tag results",
		"tenant", tenantID,
		"tag", req.TagName,
		"query", req.Query,
		"range_seconds", req.End-req.Start,
		"duration_seconds", durationSeconds,
		"error", err)
}

func logTagValuesRequest(logger log.Logger, tenantID string, req *tempopb.SearchTagValuesRequest) {
	level.Info(logger).Log(
		"msg", "search tag request",
		"tenant", tenantID,
		"tag", req.TagName,
		"query", req.Query,
		"range_seconds", req.End-req.Start)
}
