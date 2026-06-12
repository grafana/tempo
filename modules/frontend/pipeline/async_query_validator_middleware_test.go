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

func TestValidateTraceQLQueryParamsSize(t *testing.T) {
	oversizedQuery := "{ malformed query that is too large"
	tests := []struct {
		name    string
		params  url.Values
		wantErr bool
	}{
		{
			name:    "no query params",
			params:  url.Values{},
			wantErr: false,
		},
		{
			name:    "q within limit",
			params:  url.Values{"q": []string{"{}"}},
			wantErr: false,
		},
		{
			name:    "q too large",
			params:  url.Values{"q": []string{oversizedQuery}},
			wantErr: true,
		},
		{
			name:    "q too large with safe query alias",
			params:  url.Values{"query": []string{"{}"}, "q": []string{oversizedQuery}},
			wantErr: true,
		},
		{
			name:    "query too large with safe q alias",
			params:  url.Values{"q": []string{"{}"}, "query": []string{oversizedQuery}},
			wantErr: true,
		},
		{
			name:    "repeated q with one oversized value",
			params:  url.Values{"q": []string{"{}", oversizedQuery}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTraceQLQueryParamsSize(tt.params, 10)
			if tt.wantErr {
				require.ErrorContains(t, err, "TraceQL expression exceeds the configured maximum size")
				return
			}
			require.NoError(t, err)
		})
	}
}
