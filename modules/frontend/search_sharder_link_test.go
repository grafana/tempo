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
			spanIDs:    []string{"span1", "span2", "span3"},
			maxSpanIDs: 10,
			expected:   `{ link:spanID =~ "(span1|span2|span3)" && name = ` + "`backend`" + ` }`,
		},
		{
			name:       "empty condition",
			conditions: "",
			spanIDs:    []string{"span1", "span2"},
			maxSpanIDs: 10,
			expected:   `{ link:spanID =~ "(span1|span2)" }`,
		},
		{
			name:       "limit span IDs",
			conditions: `name="backend"`,
			spanIDs:    []string{"span1", "span2", "span3", "span4", "span5"},
			maxSpanIDs: 3,
			expected:   `{ link:spanID =~ "(span1|span2|span3)" && name = ` + "`backend`" + ` }`,
		},
		{
			name:       "complex condition",
			conditions: `name="backend" && status=error`,
			spanIDs:    []string{"abc123", "def456"},
			maxSpanIDs: 10,
			expected:   `{ link:spanID =~ "(abc123|def456)" && (name = ` + "`backend`) && (status = error)" + ` }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the conditions to get a SpansetExpression
			var expr traceql.SpansetExpression
			if tt.conditions != "" {
				query := "{" + tt.conditions + "}"
				parsed, err := traceql.Parse(query)
				require.NoError(t, err)

				// Extract the spanset expression from the pipeline
				if len(parsed.Pipeline.Elements) > 0 {
					if filter, ok := parsed.Pipeline.Elements[0].(*traceql.SpansetFilter); ok {
						// The filter contains a FieldExpression, we need the whole spanset
						// For testing purposes, we'll use the string representation
						condStr := buildQueryFromExpression(filter)
						// Rebuild with the condition string
						query2 := "{" + condStr + "}"
						parsed2, _ := traceql.Parse(query2)
						if len(parsed2.Pipeline.Elements) > 0 {
							expr = parsed2.Pipeline.Elements[0].(traceql.SpansetExpression)
						}
					}
				}
			}

			result := buildLinkFilterQuery(expr, tt.spanIDs, tt.maxSpanIDs)
			assert.Equal(t, tt.expected, result)
		})
	}
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
			expected: "name = `backend`",
		},
		{
			name:     "multiple conditions",
			query:    `{name="backend" && status=error}`,
			expected: "(name = `backend`) && (status = error)",
		},
		{
			name:     "with attribute",
			query:    `{span.http.status_code=500}`,
			expected: "span.http.status_code = 500",
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
