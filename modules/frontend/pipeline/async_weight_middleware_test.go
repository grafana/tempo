package pipeline

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

const (
	DefaultWeight       int = 1
	TraceQLSearchWeight int = 1
	TraceByIDWeight     int = 2
)

func TestWeightMiddlewareForTraceByIDRequest(t *testing.T) {
	config := WeightsConfig{
		RequestWithWeights: true,
	}
	roundTrip := NewWeightRequestWare(TraceByID, config).Wrap(nextRequest)
	req := DoWeightedRequest(t, "http://localhost:8080/api/v2/traces/123345", roundTrip)

	assert.Equal(t, TraceByIDWeight, req.Weight())
}

func TestDisabledWeightMiddlewareForTraceByIDRequest(t *testing.T) {
	config := WeightsConfig{
		RequestWithWeights: false,
	}
	roundTrip := NewWeightRequestWare(TraceByID, config).Wrap(nextRequest)
	req := DoWeightedRequest(t, "http://localhost:8080/api/v2/traces/123345", roundTrip)

	assert.Equal(t, DefaultWeight, req.Weight())
}

func TestWeightMiddlewareForDefaultRequest(t *testing.T) {
	config := WeightsConfig{
		RequestWithWeights: true,
	}
	roundTrip := NewWeightRequestWare(Default, config).Wrap(nextRequest)
	req := DoWeightedRequest(t, "http://localhost:8080/api/v2/search/tags", roundTrip)

	assert.Equal(t, DefaultWeight, req.Weight())
}

func TestWeightMiddlewareForTraceQLRequest(t *testing.T) {
	config := WeightsConfig{
		RequestWithWeights:   true,
		MaxTraceQLConditions: 4,
		MaxRegexConditions:   1,
	}
	roundTrip := NewWeightRequestWare(TraceQLSearch, config).Wrap(nextRequest)
	cases := []struct {
		query    string
		expected int
	}{
		{
			// Wrong query, this will be catched by the validator middlware
			query:    "{ span.http.status_code }",
			expected: TraceQLSearchWeight,
		},
		{
			// All traces
			query:    "{}",
			expected: TraceQLSearchWeight,
		},
		{
			// Simple query
			query:    "{ span.http.status_code >= 200 }",
			expected: TraceQLSearchWeight,
		},
		{
			// Simple query with OR operation
			query:    "{ span.http.status_code >= 200 || span.http.status_code < 300 }",
			expected: TraceQLSearchWeight + 1, // +1 for OR operation
		},
		{
			// Regex, complex query
			query:    "{span.a =~ \"postgresql|mysql\"}",
			expected: TraceQLSearchWeight + 1,
		},
		{
			// Regex, complex query
			query:    "{span.a !~ \"postgresql|mysql\"}",
			expected: TraceQLSearchWeight + 1,
		},

		{
			// 4 conditions
			query:    "{span.http.method = \"DELETE\" && status != ok && span.http.status_code >= 200 && span.attr < 300 }",
			expected: TraceQLSearchWeight + 1, // complex query
		},
		{
			// 4 conditions (+1) and OR operation (+1) = +2
			query:    "{span.http.method = \"DELETE\" || status != ok || span.http.status_code >= 200 || span.http.status_code < 300 }",
			expected: TraceQLSearchWeight + 2,
		},
		{
			// OR operation (AllConditions=false)
			query:    "{span.foo = \"bar\" || span.baz = \"qux\"}",
			expected: TraceQLSearchWeight + 1,
		},
		// Structural operators
		{
			query:    "{span.foo = \"bar\"} >> {span.baz = \"qux\"}",
			expected: TraceQLSearchWeight + 2, // +1 for AllConditions is false, +1 for NeedsFullTrace
		},
		{
			query:    "{span.foo = \"bar\"} > {span.baz = \"qux\"}",
			expected: TraceQLSearchWeight + 2,
		},
		{
			query:    "{span.foo = \"bar\"} << {span.baz = \"qux\"}",
			expected: TraceQLSearchWeight + 2,
		},
		{
			query:    "{} < {}",
			expected: TraceQLSearchWeight + 2,
		},
		{
			query:    "{span.a = \"1\"} ~ {span.b = \"2\"}",
			expected: TraceQLSearchWeight + 2,
		},

		{
			// Aggregate requiring full trace
			query:    "{} | count() > 5",
			expected: TraceQLSearchWeight + 2, // +1 for AllConditions is false, +1 for NeedsFullTrace
		},
		{
			// Spanset AND operation
			query:    "{} && {}",
			expected: TraceQLSearchWeight + 2, // +1 for AllConditions is false, +1 for NeedsFullTrace
		},
		{
			// Complex: regex + OR operation
			query:    "{span.a =~ \"foo.*\" || span.b = \"bar\"}",
			expected: TraceQLSearchWeight + 2, // +1 for AllConditions is false, +1 for complex query
		},
		{
			// Complex: structural operator with OR inside spanset
			// The OR inside the spanset makes AllConditions=false
			query:    "{span.a = \"1\" || span.b = \"2\"} >> {span.c = \"3\"}",
			expected: TraceQLSearchWeight + 2, // +1 for AllConditions is false, +1 for NeedsFullTrace
		},
		{
			// Complex: regex + structural operator
			query:    "{span.a =~ \"foo.*\"} >> {span.b = \"bar\"}",
			expected: TraceQLSearchWeight + 3, // +1 for regex, +1 for AllConditions is false, +1 for NeedsFullTrace
		},

		// TraceQL Metrics queries
		{
			query:    "{} && {} | rate()",
			expected: TraceQLSearchWeight + 2, // +1 for AllConditions is false, +1 for NeedsFullTrace
		},
		{
			query:    "{} | compare({status=error})",
			expected: TraceQLSearchWeight + 1, // +1 for SecondPassSelectAll
		},
	}
	for _, c := range cases {
		req := fmt.Sprintf("http://localhost:8080/api/search?query=%s", url.QueryEscape(c.query))
		actual := DoWeightedRequest(t, req, roundTrip)
		if actual.Weight() != c.expected {
			t.Errorf("query: %s, expected %d, got %d", c.query, c.expected, actual.Weight())
		}
	}
}

func TestWeightMiddlewareForTraceQLRequestURL(t *testing.T) {
	config := WeightsConfig{
		RequestWithWeights:   true,
		MaxTraceQLConditions: 4,
		MaxRegexConditions:   1,
	}
	roundTrip := NewWeightRequestWare(TraceQLSearch, config).Wrap(nextRequest)

	query := "{} && {}"
	expected := TraceQLSearchWeight + 2

	req := fmt.Sprintf("http://localhost:8080/api/search?query=%s", url.QueryEscape(query))
	actual := DoWeightedRequest(t, req, roundTrip)
	assert.Equal(t, expected, actual.Weight())

	req = fmt.Sprintf("http://example.com:3000/api/metrics/query_range?q=%s", url.QueryEscape(query))
	actual = DoWeightedRequest(t, req, roundTrip)
	assert.Equal(t, expected, actual.Weight())
}

func DoWeightedRequest(t *testing.T, url string, rt AsyncRoundTripper[combiner.PipelineResponse]) *HTTPRequest {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	request := NewHTTPRequest(req)
	resp, _ := rt.RoundTrip(request)
	_, _, err := resp.Next(context.Background())
	require.NoError(t, err)
	return request
}
