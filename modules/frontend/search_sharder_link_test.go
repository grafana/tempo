package frontend

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildLinkFilterQuery(t *testing.T) {
	tests := []struct {
		name       string
		conditions string
		spanIDs    []string
		maxSpanIDs int
		expected   string
	}{
		{
			name:       "simple condition with few span IDs",
			conditions: `name="backend"`,
			spanIDs:    []string{"0123456789abcdef", "fedcba9876543210", "0011223344556677"},
			maxSpanIDs: 10,
			expected:   `{ link:spanID =~ "(0123456789abcdef|fedcba9876543210|0011223344556677)" && name = ` + "`backend`" + ` }`,
		},
		{
			name:       "empty condition",
			conditions: "",
			spanIDs:    []string{"0123456789abcdef", "fedcba9876543210"},
			maxSpanIDs: 10,
			expected:   `{ link:spanID =~ "(0123456789abcdef|fedcba9876543210)" }`,
		},
		{
			name:       "limit span IDs",
			conditions: `name="backend"`,
			spanIDs:    []string{"0123456789abcdef", "fedcba9876543210", "0011223344556677", "8899aabbccddeeff", "1234567890abcdef"},
			maxSpanIDs: 3,
			expected:   `{ link:spanID =~ "(0123456789abcdef|fedcba9876543210|0011223344556677)" && name = ` + "`backend`" + ` }`,
		},
		{
			name:       "complex condition",
			conditions: `name="backend" && status=error`,
			spanIDs:    []string{"0123456789abcdef", "fedcba9876543210"},
			maxSpanIDs: 10,
			expected:   `{ link:spanID =~ "(0123456789abcdef|fedcba9876543210)" && (name = ` + "`backend`) && (status = error)" + ` }`,
		},
		{
			name:       "filters invalid span IDs",
			conditions: `name="backend"`,
			spanIDs:    []string{"invalid", "0123456789abcdef"},
			maxSpanIDs: 5,
			expected:   `{ link:spanID =~ "(0123456789abcdef)" && name = ` + "`backend`" + ` }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := parseSpansetExpression(t, tt.conditions)
			result, ok := buildLinkFilterQuery(expr, tt.spanIDs, tt.maxSpanIDs)
			assert.True(t, ok)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildLinkFilterQuery_NoValidIDs(t *testing.T) {
	query, ok := buildLinkFilterQuery(nil, []string{"bad", "also-bad"}, 5)
	assert.False(t, ok)
	assert.Equal(t, "", query)
}

func TestIsValidSpanID(t *testing.T) {
	assert.True(t, isValidSpanID("0123456789abcdef"))
	assert.True(t, isValidSpanID("ABCDEF0123456789"))
	assert.False(t, isValidSpanID("short"))
	assert.False(t, isValidSpanID("0123456789abcdeg"))
}

func TestApplyTraceLimit(t *testing.T) {
	traces := []*tempopb.TraceSearchMetadata{
		{TraceID: "trace-1"},
		{TraceID: "trace-2"},
		{TraceID: "trace-3"},
	}

	limited, truncated := applyTraceLimit(traces, 2)
	assert.True(t, truncated)
	assert.Equal(t, 2, len(limited))
	assert.Equal(t, "trace-1", limited[0].TraceID)
	assert.Equal(t, "trace-2", limited[1].TraceID)

	limited, truncated = applyTraceLimit(traces, 0)
	assert.False(t, truncated)
	assert.Equal(t, traces, limited)
}

func parseSpansetExpression(t *testing.T, conditions string) traceql.SpansetExpression {
	t.Helper()
	if conditions == "" {
		return nil
	}

	query := "{" + conditions + "}"
	parsed, err := traceql.Parse(query)
	require.NoError(t, err)
	if parsed == nil || len(parsed.Pipeline.Elements) == 0 {
		return nil
	}

	if expr, ok := parsed.Pipeline.Elements[0].(traceql.SpansetExpression); ok {
		return expr
	}
	return nil
}

func TestBuildQueryFromExpression(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "simple name condition",
			query:    `{name="backend"}`,
			expected: "{ name = `backend` }",
		},
		{
			name:     "multiple conditions",
			query:    `{name="backend" && status=error}`,
			expected: "{ (name = `backend`) && (status = error) }",
		},
		{
			name:     "with attribute",
			query:    `{span.http.status_code=500}`,
			expected: "{ span.http.status_code = 500 }",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := traceql.Parse(tt.query)
			require.NoError(t, err)

			// Extract the spanset expression
			var expr traceql.SpansetExpression
			if len(parsed.Pipeline.Elements) > 0 {
				expr = parsed.Pipeline.Elements[0].(traceql.SpansetExpression)
			}

			result := buildQueryFromExpression(expr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractSpanIDsFromResponse(t *testing.T) {
	tests := []struct {
		name        string
		response    *tempopb.SearchResponse
		contentType string
		expected    []string
	}{
		{
			name: "extract from SpanSets",
			response: &tempopb.SearchResponse{
				Traces: []*tempopb.TraceSearchMetadata{
					{
						TraceID: "trace1",
						SpanSets: []*tempopb.SpanSet{
							{
								Spans: []*tempopb.Span{
									{SpanID: "span1", Name: "test1"},
									{SpanID: "span2", Name: "test2"},
								},
							},
						},
					},
					{
						TraceID: "trace2",
						SpanSets: []*tempopb.SpanSet{
							{
								Spans: []*tempopb.Span{
									{SpanID: "span3", Name: "test3"},
								},
							},
						},
					},
				},
			},
			contentType: "application/json",
			expected:    []string{"span1", "span2", "span3"},
		},
		{
			name: "extract from deprecated SpanSet field",
			response: &tempopb.SearchResponse{
				Traces: []*tempopb.TraceSearchMetadata{
					{
						TraceID: "trace1",
						SpanSet: &tempopb.SpanSet{
							Spans: []*tempopb.Span{
								{SpanID: "span1", Name: "test1"},
								{SpanID: "span2", Name: "test2"},
							},
						},
					},
				},
			},
			contentType: "application/json",
			expected:    []string{"span1", "span2"},
		},
		{
			name: "extract from both SpanSet and SpanSets",
			response: &tempopb.SearchResponse{
				Traces: []*tempopb.TraceSearchMetadata{
					{
						TraceID: "trace1",
						SpanSet: &tempopb.SpanSet{
							Spans: []*tempopb.Span{
								{SpanID: "span1", Name: "test1"},
							},
						},
						SpanSets: []*tempopb.SpanSet{
							{
								Spans: []*tempopb.Span{
									{SpanID: "span2", Name: "test2"},
								},
							},
						},
					},
				},
			},
			contentType: "application/json",
			expected:    []string{"span2", "span1"}, // SpanSets processed first
		},
		{
			name: "empty response",
			response: &tempopb.SearchResponse{
				Traces: []*tempopb.TraceSearchMetadata{},
			},
			contentType: "application/json",
			expected:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal response to JSON
			body, err := json.Marshal(tt.response)
			require.NoError(t, err)

			// Create HTTP response
			httpResp := &http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Content-Type": []string{tt.contentType},
				},
				Body: io.NopCloser(bytes.NewReader(body)),
			}

			// Extract span IDs
			spanIDs, err := extractSpanIDsFromResponse(httpResp)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, spanIDs)
		})
	}
}

func TestExtractSpanIDsFromResponse_Protobuf(t *testing.T) {
	response := &tempopb.SearchResponse{
		Traces: []*tempopb.TraceSearchMetadata{
			{
				TraceID: "trace1",
				SpanSets: []*tempopb.SpanSet{
					{
						Spans: []*tempopb.Span{
							{SpanID: "span1", Name: "test1"},
							{SpanID: "span2", Name: "test2"},
						},
					},
				},
			},
		},
	}

	// Marshal to protobuf
	body, err := response.Marshal()
	require.NoError(t, err)

	// Create HTTP response
	httpResp := &http.Response{
		StatusCode: 200,
		Header: http.Header{
			"Content-Type": []string{"application/protobuf"},
		},
		Body: io.NopCloser(bytes.NewReader(body)),
	}

	// Extract span IDs
	spanIDs, err := extractSpanIDsFromResponse(httpResp)
	require.NoError(t, err)
	assert.Equal(t, []string{"span1", "span2"}, spanIDs)
}

func TestExtractSpanIDsFromResponse_NilResponse(t *testing.T) {
	spanIDs, err := extractSpanIDsFromResponse(nil)
	require.NoError(t, err)
	assert.Nil(t, spanIDs)
}

func TestExtractSpanIDsFromResponse_InvalidJSON(t *testing.T) {
	httpResp := &http.Response{
		StatusCode: 200,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(bytes.NewReader([]byte("invalid json"))),
	}

	spanIDs, err := extractSpanIDsFromResponse(httpResp)
	// Should return error for invalid JSON
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal JSON response")
	assert.Empty(t, spanIDs)
}
