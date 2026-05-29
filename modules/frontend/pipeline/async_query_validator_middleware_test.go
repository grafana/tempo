package pipeline

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/stretchr/testify/require"
)

var nextFunc = AsyncRoundTripperFunc[combiner.PipelineResponse](func(_ Request) (Responses[combiner.PipelineResponse], error) {
	return NewHTTPToAsyncResponse(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader([]byte("foo"))),
	}), nil
})

func TestQueryValidator(t *testing.T) {
	tests := []struct {
		name              string
		url               string
		statusCode        int
		maxQuerySizeBytes int
	}{
		{
			name:              "No Query",
			url:               "http://localhost:8080/api/search",
			statusCode:        200,
			maxQuerySizeBytes: 1000,
		},
		{
			name:              "Empty query value",
			url:               "http://localhost:8080/api/search&q=",
			statusCode:        200,
			maxQuerySizeBytes: 1000,
		},
		{
			name:              "Valid query",
			url:               "http://localhost:8080/api/search&q={}",
			statusCode:        200,
			maxQuerySizeBytes: 1000,
		},
		{
			name:              "Invalid TraceQL query",
			url:               "http://localhost:8080/api/search?q={. hi}",
			statusCode:        400,
			maxQuerySizeBytes: 1000,
		},
		{
			name:              "Invalid TraceQL query regex",
			url:               "http://localhost:8080/api/search?query={span.a =~ \"[\"}",
			statusCode:        400,
			maxQuerySizeBytes: 1000,
		},
		{
			name:              "TraceQL query smaller then max size",
			url:               "http://localhost:8080/api/search?q={ resource.service.name=\"service\" }",
			statusCode:        200,
			maxQuerySizeBytes: 1000,
		},
		{
			name:              "TraceQL query bigger then max size",
			url:               "http://localhost:8080/api/search?q={ resource.service.name=\"service\" }",
			statusCode:        400,
			maxQuerySizeBytes: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := NewQueryValidatorWare(tt.maxQuerySizeBytes).Wrap(nextFunc)
			req, _ := http.NewRequest(http.MethodGet, tt.url, nil)
			resp, _ := rt.RoundTrip(NewHTTPRequest(req))

			httpResponse, _, err := resp.Next(context.Background())
			require.NoError(t, err)
			body, err := io.ReadAll(httpResponse.HTTPResponse().Body)
			require.NoError(t, err)
			require.NotEmpty(t, body)

			require.Equal(t, tt.statusCode, httpResponse.HTTPResponse().StatusCode)
		})
	}
}

func TestQuerySizeValidator(t *testing.T) {
	oversizedQuery := url.QueryEscape("{ malformed query that is too large")
	tests := []struct {
		name string
		url  string
	}{
		{
			name: "q too large",
			url:  "http://localhost:8080/api/search?q=" + oversizedQuery,
		},
		{
			name: "q too large with safe query alias",
			url:  "http://localhost:8080/api/search?query={}&q=" + oversizedQuery,
		},
		{
			name: "query too large with safe q alias",
			url:  "http://localhost:8080/api/search?q={}&query=" + oversizedQuery,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := NewQuerySizeValidatorWare(10).Wrap(nextFunc)
			req, err := http.NewRequest(http.MethodGet, tt.url, nil)
			require.NoError(t, err)

			resp, err := rt.RoundTrip(NewHTTPRequest(req))
			require.NoError(t, err)

			httpResponse, _, err := resp.Next(context.Background())
			require.NoError(t, err)
			body, err := io.ReadAll(httpResponse.HTTPResponse().Body)
			require.NoError(t, err)

			require.Equal(t, http.StatusBadRequest, httpResponse.HTTPResponse().StatusCode)
			require.Contains(t, string(body), "TraceQL expression exceeds the configured maximum size")
		})
	}
}
