package frontend

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockNextTripperware struct{}

func (s *mockNextTripperware) RoundTrip(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		Body: io.NopCloser(bytes.NewReader([]byte("next"))),
	}, nil
}

type mockTracesTripperware struct{}

func (s *mockTracesTripperware) RoundTrip(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		Body: io.NopCloser(bytes.NewReader([]byte("traces"))),
	}, nil
}

type mockSearchTripperware struct{}

func (s *mockSearchTripperware) RoundTrip(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		Body: io.NopCloser(bytes.NewReader([]byte("search"))),
	}, nil
}

func TestFrontendRoundTripper(t *testing.T) {
	next := &mockNextTripperware{}
	traces := &mockTracesTripperware{}
	search := &mockSearchTripperware{}

	testCases := []struct {
		name      string
		apiPrefix string
		endpoint  string
		response  string
	}{
		{
			name:      "next tripper",
			apiPrefix: "",
			endpoint:  "/api/foo",
			response:  "next",
		},
		{
			name:      "traces tripper",
			apiPrefix: "",
			endpoint:  apiPathTraces + "/X",
			response:  "traces",
		},
		{
			name:      "search tripper",
			apiPrefix: "",
			endpoint:  apiPathSearch + "/X",
			response:  "search",
		},
		{
			name:      "traces tripper with prefix",
			apiPrefix: "/tempo",
			endpoint:  "/tempo" + apiPathTraces + "/X",
			response:  "traces",
		},
		{
			name:      "next tripper with a misleading prefix",
			apiPrefix: "/api/traces",
			endpoint:  "/api/traces" + apiPathSearch + "/api/traces",
			response:  "search",
		},
	}

	queriesPerTenant := promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "query_frontend_queries_total",
		Help:      "Total queries received per tenant.",
	}, []string{"tenant", "op"})

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			frontendTripper := frontendRoundTripper{
				apiPrefix:        tt.apiPrefix,
				next:             next,
				traces:           traces,
				search:           search,
				logger:           log.NewNopLogger(),
				queriesPerTenant: queriesPerTenant,
			}

			req := &http.Request{
				URL: &url.URL{
					Path: tt.endpoint,
				},
			}
			resp, err := frontendTripper.RoundTrip(req)
			require.NoError(t, err)
			require.NotNil(t, resp)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, body, []byte(tt.response))
		})
	}
}
