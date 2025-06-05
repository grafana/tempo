package frontend

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/go-kit/log"
	"github.com/gorilla/mux"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

type expectedResult struct {
	err    error
	path   string
	params map[string]string
}

func TestInjectMuxVars(t *testing.T) {
	ctx := context.Background()
	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/test"},
	}

	vars := map[string]string{
		"traceID": "test-trace-id",
		"tagName": "test-tag",
	}

	newReq, newCtx := injectMuxVars(ctx, req, vars)

	// Verify vars are set in the request
	actualVars := mux.Vars(newReq)
	require.Equal(t, "test-trace-id", actualVars["traceID"])
	require.Equal(t, "test-tag", actualVars["tagName"])

	require.Equal(t, newReq.Context(), newCtx)
}

func TestHandleSearch(t *testing.T) {
	server, callAndTestResults := testFrontend()

	tests := []struct {
		name     string
		request  mcp.CallToolRequest
		expected expectedResult
	}{
		{
			name: "query only",
			request: callToolRequest(map[string]any{
				"query": "{ span.foo = \"bar\" }",
			}),
			expected: expectedResult{
				err:  nil,
				path: "/api/search",
				params: map[string]string{
					"q": "{ span.foo = \"bar\" }",
				},
			},
		},
		{
			name:    "no query!",
			request: callToolRequest(map[string]any{}),
			expected: expectedResult{
				err: errors.New("required argument \"query\" not found"),
			},
		},
		{
			name: "query + start + end",
			request: callToolRequest(map[string]any{
				"query": "{ span.foo = \"bar\" }",
				"start": "2022-01-01T00:00:00Z",
				"end":   "2022-01-02T00:00:00Z",
			}),
			expected: expectedResult{
				err:  nil,
				path: "/api/search",
				params: map[string]string{
					"q":     "{ span.foo = \"bar\" }",
					"start": "1640995200",
					"end":   "1641081600",
				},
			},
		},
		{
			name: "bad query",
			request: callToolRequest(map[string]any{
				"query": "{ foo bar baz }",
			}),
			expected: expectedResult{
				err: errors.New("Query parse error. Consult TraceQL docs tools: parse error at line 1, col 3: syntax error: unexpected IDENTIFIER"),
			},
		},
		{
			name: "metrics query",
			request: callToolRequest(map[string]any{
				"query": "{} | rate()",
			}),
			expected: expectedResult{
				err: errors.New("TraceQL metrics query received on traceql-search tool. Use the traceql-metrics-instant or traceql-metrics-range tool instead."),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callAndTestResults(t, tt.request, server.handleSearch, tt.expected)
		})
	}
}

func TestHandleInstantQuery(t *testing.T) {
	server, callAndTestResults := testFrontend()

	tests := []struct {
		name     string
		request  mcp.CallToolRequest
		expected expectedResult
	}{
		{
			name: "query only",
			request: callToolRequest(map[string]any{
				"query": "{} | rate()",
			}),
			expected: expectedResult{
				err:  nil,
				path: "/api/metrics/query",
				params: map[string]string{
					"q": "{} | rate()",
				},
			},
		},
		{
			name:    "no query",
			request: callToolRequest(map[string]any{}),
			expected: expectedResult{
				err: errors.New("required argument \"query\" not found"),
			},
		},
		{
			name: "query + start + end",
			request: callToolRequest(map[string]any{
				"query": "{} | rate()",
				"start": "2022-01-01T00:00:00Z",
				"end":   "2022-01-02T00:00:00Z",
			}),
			expected: expectedResult{
				err:  nil,
				path: "/api/metrics/query",
				params: map[string]string{
					"q":     "{} | rate()",
					"start": "1640995200000000000",
					"end":   "1641081600000000000",
				},
			},
		},
		{
			name: "bad query",
			request: callToolRequest(map[string]any{
				"query": "{ foo bar baz }",
			}),
			expected: expectedResult{
				err: errors.New("Query parse error. Consult TraceQL docs tools: parse error at line 1, col 3: syntax error: unexpected IDENTIFIER"),
			},
		},
		{
			name: "no metrics query",
			request: callToolRequest(map[string]any{
				"query": "{}",
			}),
			expected: expectedResult{
				err: errors.New("TraceQL search query received on instant query tool. Use the traceql-search tool instead."),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callAndTestResults(t, tt.request, server.handleInstantQuery, tt.expected)
		})
	}
}

func TestHandleRangeQuery(t *testing.T) {
	server, callAndTestResults := testFrontend()

	tests := []struct {
		name     string
		request  mcp.CallToolRequest
		expected expectedResult
	}{
		{
			name: "query only",
			request: callToolRequest(map[string]any{
				"query": "{} | rate()",
			}),
			expected: expectedResult{
				err:  nil,
				path: "/api/metrics/query_range",
				params: map[string]string{
					"q": "{} | rate()",
				},
			},
		},
		{
			name:    "no query",
			request: callToolRequest(map[string]any{}),
			expected: expectedResult{
				err: errors.New("required argument \"query\" not found"),
			},
		},
		{
			name: "query + start + end",
			request: callToolRequest(map[string]any{
				"query": "{} | rate()",
				"start": "2022-01-01T00:00:00Z",
				"end":   "2022-01-02T00:00:00Z",
			}),
			expected: expectedResult{
				err:  nil,
				path: "/api/metrics/query_range",
				params: map[string]string{
					"q":     "{} | rate()",
					"start": "1640995200000000000", // query range uses nanos
					"end":   "1641081600000000000",
				},
			},
		},
		{
			name: "bad query",
			request: callToolRequest(map[string]any{
				"query": "{ foo bar baz }",
			}),
			expected: expectedResult{
				err: errors.New("Query parse error. Consult TraceQL docs tools: parse error at line 1, col 3: syntax error: unexpected IDENTIFIER"),
			},
		},
		{
			name: "no metrics query",
			request: callToolRequest(map[string]any{
				"query": "{}",
			}),
			expected: expectedResult{
				err: errors.New("TraceQL search query received on range query tool. Use the traceql-search tool instead."),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callAndTestResults(t, tt.request, server.handleRangeQuery, tt.expected)
		})
	}
}

func TestHandleGetTrace(t *testing.T) {
	server, callAndTestResults := testFrontend()

	tests := []struct {
		name     string
		request  mcp.CallToolRequest
		expected expectedResult
	}{
		{
			name: "valid trace ID",
			request: callToolRequest(map[string]any{
				"trace_id": "12345678abcdef90",
			}),
			expected: expectedResult{
				err:    nil,
				path:   "/api/v2/traces/12345678abcdef90",
				params: map[string]string{},
			},
		},
		{
			name:    "no trace ID",
			request: callToolRequest(map[string]any{}),
			expected: expectedResult{
				err: errors.New("required argument \"trace_id\" not found"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callAndTestResults(t, tt.request, server.handleGetTrace, tt.expected)
		})
	}
}

func TestHandleGetAttributeNames(t *testing.T) {
	server, callAndTestResults := testFrontend()

	tests := []struct {
		name     string
		request  mcp.CallToolRequest
		expected expectedResult
	}{
		{
			name:    "no scope",
			request: callToolRequest(map[string]any{}),
			expected: expectedResult{
				err:    nil,
				path:   "/api/v2/search/tags",
				params: map[string]string{},
			},
		},
		{
			name: "with scope",
			request: callToolRequest(map[string]any{
				"scope": "span",
			}),
			expected: expectedResult{
				err:  nil,
				path: "/api/v2/search/tags",
				params: map[string]string{
					"scope": "span",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callAndTestResults(t, tt.request, server.handleGetAttributeNames, tt.expected)
		})
	}
}

func TestHandleGetAttributeValues(t *testing.T) {
	server, callAndTestResults := testFrontend()

	tests := []struct {
		name     string
		request  mcp.CallToolRequest
		expected expectedResult
	}{
		{
			name: "name only",
			request: callToolRequest(map[string]any{
				"name": "service.name",
			}),
			expected: expectedResult{
				err:    nil,
				path:   "/api/v2/search/tag/service.name/values",
				params: map[string]string{},
			},
		},
		{
			name:    "no name",
			request: callToolRequest(map[string]any{}),
			expected: expectedResult{
				err: errors.New("required argument \"name\" not found"),
			},
		},
		{
			name: "name + filter query",
			request: callToolRequest(map[string]any{
				"name":         "service.name",
				"filter-query": "{ span.status = \"error\" }",
			}),
			expected: expectedResult{
				err:  nil,
				path: "/api/v2/search/tag/service.name/values",
				params: map[string]string{
					"q": "{ span.status = \"error\" }",
				},
			},
		},
		{
			name: "invalid filter",
			request: callToolRequest(map[string]any{
				"name":         "service.name",
				"filter-query": "{ foo bar baz }",
			}),
			expected: expectedResult{
				err: errors.New("filter-query invalid. It can only have one spanset and only &&'ed conditions like { <cond> && <cond> && ... }"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callAndTestResults(t, tt.request, server.handleGetAttributeValues, tt.expected)
		})
	}
}

func TestHandleTraceQLQuery(t *testing.T) {
	server := &MCPServer{
		logger: log.NewNopLogger(),
	}

	ctx := context.Background()
	request := callToolRequest(map[string]any{})

	result, err := server.handleTraceQLQuery(ctx, request)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Content)
}

func TestHandleTraceQLMetrics(t *testing.T) {
	server := &MCPServer{
		logger: log.NewNopLogger(),
	}

	ctx := context.Background()
	request := callToolRequest(map[string]any{})

	result, err := server.handleTraceQLMetrics(ctx, request)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Content)
}

func callToolRequest(args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: struct {
			Name      string    "json:\"name\""
			Arguments any       "json:\"arguments,omitempty\""
			Meta      *mcp.Meta "json:\"_meta,omitempty\""
		}{
			Arguments: args,
		},
	}
}

func testFrontend() (*MCPServer, func(t *testing.T, req mcp.CallToolRequest, handler func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error), expected expectedResult)) {
	var lastRequest *http.Request

	// Mock search handler that returns a successful response and stores the request
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastRequest = r
		w.WriteHeader(http.StatusOK)
	})

	server := &MCPServer{
		frontend: &QueryFrontend{
			SearchHandler:              mockHandler,
			TraceByIDHandlerV2:         mockHandler,
			SearchTagsV2Handler:        mockHandler,
			SearchTagsValuesV2Handler:  mockHandler,
			MetricsQueryInstantHandler: mockHandler,
			MetricsQueryRangeHandler:   mockHandler,
		},
		logger:     log.NewNopLogger(),
		pathPrefix: "",
	}

	callAndTestResults := func(t *testing.T, req mcp.CallToolRequest, handler func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error), expected expectedResult) {
		ctx := context.Background()
		result, err := handler(ctx, req)

		if expected.err != nil {
			require.Error(t, err)
			require.EqualError(t, err, expected.err.Error())
			return
		}

		require.NoError(t, err)
		require.NotNil(t, result)

		// Parse and verify the request URL
		require.Equal(t, expected.path, lastRequest.URL.Path)

		actualParams := lastRequest.URL.Query()
		for key, expectedValue := range expected.params {
			require.Equal(t, expectedValue, actualParams.Get(key), "parameter %s", key)
		}
	}

	return server, callAndTestResults
}
