package frontend

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"time"

	"github.com/go-kit/log/level"
	"github.com/gorilla/mux"
	"github.com/grafana/tempo/docs"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *MCPServer) handleTraceQLQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	level.Info(s.logger).Log("msg", "traceql query tool requested")

	return mcp.NewToolResultText(trimDocs(docs.TraceQLMain)), nil
}

func (s *MCPServer) handleTraceQLMetrics(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	level.Info(s.logger).Log("msg", "traceql metrics tool requested")

	return mcp.NewToolResultText(trimDocs(docs.TraceQLMetrics)), nil
}

// handleSearch handles the traceql-search tool
func (s *MCPServer) handleSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := request.RequireString("query")
	if err != nil {
		return nil, err
	}

	var startEpoch, endEpoch int64

	start := request.GetString("start", "")
	end := request.GetString("end", "")

	level.Info(s.logger).Log("msg", "searching traces", "query", query, "start", start, "end", end)

	if start == "" {
		startEpoch = time.Now().Add(-1 * time.Hour).Unix()
	} else {
		startTS, err := time.Parse(time.RFC3339, start)
		if err != nil {
			return nil, fmt.Errorf("invalid start time: %w", err)
		}
		startEpoch = startTS.Unix()
	}
	if end == "" {
		endEpoch = time.Now().Unix()
	} else {
		endTS, err := time.Parse(time.RFC3339, end)
		if err != nil {
			return nil, fmt.Errorf("invalid end time: %w", err)
		}
		endEpoch = endTS.Unix()
	}

	parsed, err := traceql.Parse(query)
	if err != nil {
		return nil, fmt.Errorf("Query parse error. Consult TraceQL docs tools: %w", err)
	}

	if parsed.MetricsPipeline != nil || parsed.MetricsSecondStage != nil {
		return nil, fmt.Errorf("TraceQL metrics query received on traceql-search tool. Use the traceql-metrics-instant or traceql-metrics-range tool instead.")
	}

	searchReq := &tempopb.SearchRequest{
		Query: query,
		Start: uint32(startEpoch),
		End:   uint32(endEpoch),
	}

	req, err := api.BuildSearchRequest(nil, searchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to build search request: %w", err)
	}
	req.URL.Path = s.buildPath(api.PathSearch)

	body, err := handleHTTP(ctx, s.frontend.SearchHandler, req)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(body), nil
}

// handleInstantQuery handles the traceql-metrics-instant tool
func (s *MCPServer) handleInstantQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := request.RequireString("query")
	if err != nil {
		return nil, err
	}

	var startEpochNanos, endEpochNanos int64

	start := request.GetString("start", "")
	end := request.GetString("end", "")

	level.Info(s.logger).Log("msg", "executing instant metrics query", "query", query, "start", start, "end", end)

	if start == "" {
		startEpochNanos = time.Now().Add(-1 * time.Hour).UnixNano()
	} else {
		startTS, err := time.Parse(time.RFC3339, start)
		if err != nil {
			return nil, fmt.Errorf("invalid start time: %w", err)
		}
		startEpochNanos = startTS.UnixNano()
	}
	if end == "" {
		endEpochNanos = time.Now().UnixNano()
	} else {
		endTS, err := time.Parse(time.RFC3339, end)
		if err != nil {
			return nil, fmt.Errorf("invalid end time: %w", err)
		}
		endEpochNanos = endTS.UnixNano()
	}

	parsed, err := traceql.Parse(query)
	if err != nil {
		return nil, fmt.Errorf("Query parse error. Consult TraceQL docs tools: %w", err)
	}

	if parsed.MetricsPipeline == nil {
		return nil, fmt.Errorf("TraceQL search query received on instant query tool. Use the traceql-search tool instead.")
	}

	queryInstantReq := &tempopb.QueryInstantRequest{
		Query: query,
		Start: uint64(startEpochNanos),
		End:   uint64(endEpochNanos),
	}

	req := api.BuildQueryInstantRequest(nil, queryInstantReq)
	req.URL.Path = s.buildPath(api.PathMetricsQueryInstant)

	body, err := handleHTTP(ctx, s.frontend.MetricsQueryInstantHandler, req)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(body), nil
}

func (s *MCPServer) handleRangeQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := request.RequireString("query")
	if err != nil {
		return nil, err
	}

	var startEpochNanos, endEpochNanos int64

	start := request.GetString("start", "")
	end := request.GetString("end", "")

	level.Info(s.logger).Log("msg", "executing range metrics query", "query", query, "start", start, "end", end)

	if start == "" {
		startEpochNanos = time.Now().Add(-1 * time.Hour).UnixNano()
	} else {
		startTS, err := time.Parse(time.RFC3339, start)
		if err != nil {
			return nil, fmt.Errorf("invalid start time: %w", err)
		}
		startEpochNanos = startTS.UnixNano()
	}
	if end == "" {
		endEpochNanos = time.Now().UnixNano()
	} else {
		endTS, err := time.Parse(time.RFC3339, end)
		if err != nil {
			return nil, fmt.Errorf("invalid end time: %w", err)
		}
		endEpochNanos = endTS.UnixNano()
	}

	parsed, err := traceql.Parse(query)
	if err != nil {
		return nil, fmt.Errorf("Query parse error. Consult TraceQL docs tools: %w", err)
	}

	if parsed.MetricsPipeline == nil {
		return nil, fmt.Errorf("TraceQL search query received on range query tool. Use the traceql-search tool instead.")
	}

	queryRangeReq := &tempopb.QueryRangeRequest{
		Query: query,
		Start: uint64(startEpochNanos),
		End:   uint64(endEpochNanos),
	}

	req := api.BuildQueryRangeRequest(nil, queryRangeReq, "")
	req.URL.Path = s.buildPath(api.PathMetricsQueryRange)

	body, err := handleHTTP(ctx, s.frontend.MetricsQueryRangeHandler, req)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(body), nil
}

// handleGetTrace handles the get-trace tool
func (s *MCPServer) handleGetTrace(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	traceID, err := request.RequireString("trace_id")
	if err != nil {
		return nil, err
	}

	level.Info(s.logger).Log("msg", "getting trace", "trace_id", traceID)

	httpReq := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: s.buildPath("/api/v2/traces/" + url.PathEscape(traceID))},
	}
	httpReq, ctx = injectMuxVars(ctx, httpReq, map[string]string{"traceID": traceID})

	body, err := handleHTTP(ctx, s.frontend.TraceByIDHandlerV2, httpReq)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(body), nil
}

// handleGetAttributeNames handles the get-attribute-names tool
func (s *MCPServer) handleGetAttributeNames(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	level.Info(s.logger).Log("msg", "getting attribute names")

	searchTagsReq := &tempopb.SearchTagsRequest{
		Scope: request.GetString("scope", ""),
	}

	req, err := api.BuildSearchTagsRequest(nil, searchTagsReq)
	if err != nil {
		return nil, fmt.Errorf("failed to build search request: %w", err)
	}
	req.URL.Path = s.buildPath(api.PathSearchTagsV2)

	body, err := handleHTTP(ctx, s.frontend.SearchTagsV2Handler, req)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(body), nil
}

// handleGetAttributeValues handles the get-attribute-values tool
func (s *MCPServer) handleGetAttributeValues(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := request.RequireString("name")
	if err != nil {
		return nil, err
	}

	query := request.GetString("filter-query", "")
	if query != "" {
		q := traceql.ExtractMatchers(query)
		if traceql.IsEmptyQuery(q) {
			return nil, fmt.Errorf("filter-query invalid. It can only have one spanset and only &&'ed conditions like { <cond> && <cond> && ... }")
		}
	}

	level.Info(s.logger).Log("msg", "getting attribute values", "name", name, "filter query", query)

	searchTagValuesReq := &tempopb.SearchTagValuesRequest{
		TagName: name,
		Query:   query,
	}

	req, err := api.BuildSearchTagValuesRequest(nil, searchTagValuesReq)
	if err != nil {
		return nil, fmt.Errorf("failed to build search request: %w", err)
	}
	req.URL.Path = s.buildPath("/api/v2/search/tag/" + url.PathEscape(name) + "/values")

	req, ctx = injectMuxVars(ctx, req, map[string]string{api.MuxVarTagName: name})

	body, err := handleHTTP(ctx, s.frontend.SearchTagsValuesV2Handler, req)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(body), nil
}

func handleHTTP(ctx context.Context, handler http.Handler, req *http.Request) (string, error) {
	rw := httptest.NewRecorder()
	req = req.WithContext(ctx)

	if req.Body == nil {
		req.Body = io.NopCloser(bytes.NewReader([]byte{})) // prevents panic
	}

	if req.Header == nil {
		req.Header = make(http.Header)
	}

	if req.RequestURI == "" {
		req.RequestURI = req.URL.RequestURI()
	}

	handler.ServeHTTP(rw, req)

	body := rw.Body.String()

	if rw.Code != http.StatusOK {
		return "", fmt.Errorf("tool failed with http status code %d and reason %s", rw.Code, body)
	}

	return body, nil
}

// injectMuxVars uses the mux.SetVars method to add vars into the context that can be used by downstream handlers.
// a few Tempo endpoints rely on the mux routing package extracting vars from the request path. this method allows
// us to do the same for MCP tools.
func injectMuxVars(ctx context.Context, req *http.Request, vars map[string]string) (*http.Request, context.Context) {
	req = req.WithContext(ctx)
	req = mux.SetURLVars(req, vars)
	ctx = req.Context()

	return req, req.Context()
}

// buildPath is a helper method to build a path with the path prefix
func (s *MCPServer) buildPath(p string) string {
	return path.Join(s.pathPrefix, p)
}
