package frontend

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/go-kit/log"
	"github.com/gorilla/mux"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

type expectedResult struct {
	err    string
	path   string
	params map[string]string
	meta   map[string]any
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
				path: "/api/search",
				params: map[string]string{
					"q": "{ span.foo = \"bar\" }",
				},
				meta: map[string]any{
					"type":     "search-results",
					"encoding": "json",
					"version":  "1",
				},
			},
		},
		{
			name:    "no query!",
			request: callToolRequest(map[string]any{}),
			expected: expectedResult{
				err: "required argument \"query\" not found",
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
				path: "/api/search",
				params: map[string]string{
					"q":     "{ span.foo = \"bar\" }",
					"start": "1640995200",
					"end":   "1641081600",
				},
				meta: map[string]any{
					"type":     "search-results",
					"encoding": "json",
					"version":  "1",
				},
			},
		},
		{
			name: "bad query",
			request: callToolRequest(map[string]any{
				"query": "{ foo bar baz }",
			}),
			expected: expectedResult{
				err: "query parse error. Consult TraceQL docs tools: parse error at line 1, col 3: syntax error: unexpected IDENTIFIER",
			},
		},
		{
			name: "metrics query",
			request: callToolRequest(map[string]any{
				"query": "{} | rate()",
			}),
			expected: expectedResult{
				err: "TraceQL metrics query received on traceql-search tool. Use the traceql-metrics-instant or traceql-metrics-range tool instead",
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
				path: "/api/metrics/query",
				params: map[string]string{
					"q": "{} | rate()",
				},
				meta: map[string]any{
					"type":     "metrics-instant",
					"encoding": "json",
					"version":  "1",
				},
			},
		},
		{
			name:    "no query",
			request: callToolRequest(map[string]any{}),
			expected: expectedResult{
				err: "required argument \"query\" not found",
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
				path: "/api/metrics/query",
				params: map[string]string{
					"q":     "{} | rate()",
					"start": "1640995200000000000",
					"end":   "1641081600000000000",
				},
				meta: map[string]any{
					"type":     "metrics-instant",
					"encoding": "json",
					"version":  "1",
				},
			},
		},
		{
			name: "bad query",
			request: callToolRequest(map[string]any{
				"query": "{ foo bar baz }",
			}),
			expected: expectedResult{
				err: "query parse error. Consult TraceQL docs tools: parse error at line 1, col 3: syntax error: unexpected IDENTIFIER",
			},
		},
		{
			name: "no metrics query",
			request: callToolRequest(map[string]any{
				"query": "{}",
			}),
			expected: expectedResult{
				err: "TraceQL search query received on instant query tool. Use the traceql-search tool instead",
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
				path: "/api/metrics/query_range",
				params: map[string]string{
					"q": "{} | rate()",
				},
				meta: map[string]any{
					"type":     "metrics-range",
					"encoding": "json",
					"version":  "1",
				},
			},
		},
		{
			name:    "no query",
			request: callToolRequest(map[string]any{}),
			expected: expectedResult{
				err: "required argument \"query\" not found",
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
				path: "/api/metrics/query_range",
				params: map[string]string{
					"q":     "{} | rate()",
					"start": "1640995200000000000", // query range uses nanos
					"end":   "1641081600000000000",
				},
				meta: map[string]any{
					"type":     "metrics-range",
					"encoding": "json",
					"version":  "1",
				},
			},
		},
		{
			name: "bad query",
			request: callToolRequest(map[string]any{
				"query": "{ foo bar baz }",
			}),
			expected: expectedResult{
				err: "query parse error. Consult TraceQL docs tools: parse error at line 1, col 3: syntax error: unexpected IDENTIFIER",
			},
		},
		{
			name: "no metrics query",
			request: callToolRequest(map[string]any{
				"query": "{}",
			}),
			expected: expectedResult{
				err: "TraceQL search query received on range query tool. Use the traceql-search tool instead",
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
				path:   "/api/v2/traces/12345678abcdef90",
				params: map[string]string{},
				meta: map[string]any{
					"type":     "trace",
					"encoding": "json",
					"version":  "2",
				},
			},
		},
		{
			name:    "no trace ID",
			request: callToolRequest(map[string]any{}),
			expected: expectedResult{
				err: "required argument \"trace_id\" not found",
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
				path:   "/api/v2/search/tags",
				params: map[string]string{},
				meta: map[string]any{
					"type":     "attribute-names",
					"encoding": "json",
					"version":  "2",
				},
			},
		},
		{
			name: "with scope",
			request: callToolRequest(map[string]any{
				"scope": "span",
			}),
			expected: expectedResult{
				path: "/api/v2/search/tags",
				params: map[string]string{
					"scope": "span",
				},
				meta: map[string]any{
					"type":     "attribute-names",
					"encoding": "json",
					"version":  "2",
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
				path:   "/api/v2/search/tag/service.name/values",
				params: map[string]string{},
				meta: map[string]any{
					"type":     "attribute-values",
					"encoding": "json",
					"version":  "2",
				},
			},
		},
		{
			name:    "no name",
			request: callToolRequest(map[string]any{}),
			expected: expectedResult{
				err: "required argument \"name\" not found",
			},
		},
		{
			name: "name + filter query",
			request: callToolRequest(map[string]any{
				"name":         "service.name",
				"filter-query": "{ span.status = \"error\" }",
			}),
			expected: expectedResult{
				path: "/api/v2/search/tag/service.name/values",
				params: map[string]string{
					"q": "{ span.status = \"error\" }",
				},
				meta: map[string]any{
					"type":     "attribute-values",
					"encoding": "json",
					"version":  "2",
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
				err: "filter-query invalid. It can only have one spanset and only &&'ed conditions like { <cond> && <cond> && ... }",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callAndTestResults(t, tt.request, server.handleGetAttributeValues, tt.expected)
		})
	}
}

func TestHandleTraceQLDocs(t *testing.T) {
	server := &MCPServer{
		logger: log.NewNopLogger(),
	}

	ctx := context.Background()
	request := callToolRequest(map[string]any{})

	result, err := server.handleTraceQLDocs(ctx, request)

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

		require.NoError(t, err)
		require.NotNil(t, result)

		if expected.meta != nil {
			require.Equal(t, expected.meta, result.Meta.AdditionalFields)
		}

		if expected.err != "" {
			// Check if the result contains an error
			require.True(t, result.IsError)
			require.Len(t, result.Content, 1)
			textContent, ok := result.Content[0].(mcp.TextContent)
			require.True(t, ok)
			require.Equal(t, expected.err, textContent.Text)
			return
		}

		// For successful cases, verify we have text content
		require.False(t, result.IsError)
		require.NotEmpty(t, result.Content)

		// Parse and verify the request URL
		require.Equal(t, expected.path, lastRequest.URL.Path)

		actualParams := lastRequest.URL.Query()
		for key, expectedValue := range expected.params {
			require.Equal(t, expectedValue, actualParams.Get(key), "parameter %s", key)
		}
	}

	return server, callAndTestResults
}
