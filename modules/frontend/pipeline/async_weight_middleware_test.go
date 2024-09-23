package pipeline

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var nextRequest = AsyncRoundTripperFunc[combiner.PipelineResponse](func(_ Request) (Responses[combiner.PipelineResponse], error) {
	return NewHTTPToAsyncResponse(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader([]byte{})),
	}), nil
})

func TestWeightMiddlewareForTraceByIDRequest(t *testing.T) {
	roundTrip := NewWeightRequestWare(TraceByID).Wrap(nextRequest)
	req := DoWeightedRequest(t, "http://localhost:8080/api/v2/traces/123345", roundTrip)

	assert.Equal(t, TraceByIDWeight, req.Weight())
}

func TestWeightMiddlewareForDefaultRequest(t *testing.T) {
	roundTrip := NewWeightRequestWare(Default).Wrap(nextRequest)
	req := DoWeightedRequest(t, "http://localhost:8080/api/v2/search/tags", roundTrip)

	assert.Equal(t, DefaultWeight, req.Weight())
}

func TestWeightMiddlewareForTraceQLRequest(t *testing.T) {
	roundTrip := NewWeightRequestWare(TraceQLSearch).Wrap(nextRequest)
	cases := []struct {
		req      string
		expected int
	}{
		{
			req:      "http://localhost:3200/api/search?q={ span.http.status_code >= 200 }",
			expected: TraceQLSearchWeight,
		},
		{
			req:      "http://localhost:3200/api/search?q={ span.http.status_code >= 200 || span.http.status_code < 300 }",
			expected: TraceQLSearchWeight,
		},
		{
			req:      "http://localhost:8080/api/search?query={span.a =~ \"postgresql|mysql\"}",
			expected: TraceQLSearchWeight + 1,
		},
		{
			req:      "http://localhost:8080/api/search?query={span.a !~ \"postgresql|mysql\"}",
			expected: TraceQLSearchWeight + 1,
		},
		{
			req:      "http://localhost:8080/api/search?query={span.http.method = \"DELETE\" || status != ok || span.http.status_code >= 200 || span.http.status_code < 300 }",
			expected: TraceQLSearchWeight + 1,
		},
	}
	for _, c := range cases {
		actual := DoWeightedRequest(t, c.req, roundTrip)
		if actual.Weight() != c.expected {
			t.Errorf("expected %d, got %d", c.expected, actual.Weight())
		}
	}
}

func DoWeightedRequest(t *testing.T, url string, rt AsyncRoundTripper[combiner.PipelineResponse]) *HTTPRequest {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	request := NewHTTPRequest(req)
	resp, _ := rt.RoundTrip(request)
	_, _, err := resp.Next(context.Background())
	require.NoError(t, err)
	return request
}
