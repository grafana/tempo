package frontend

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockNextTripperware struct{}

func (s *mockNextTripperware) RoundTrip(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		Body: ioutil.NopCloser(bytes.NewReader([]byte("next"))),
	}, nil
}

type mockTracesTripperware struct{}

func (s *mockTracesTripperware) RoundTrip(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		Body: ioutil.NopCloser(bytes.NewReader([]byte("traces"))),
	}, nil
}

type mockSearchTripperware struct{}

func (s *mockSearchTripperware) RoundTrip(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		Body: ioutil.NopCloser(bytes.NewReader([]byte("search"))),
	}, nil
}

func TestFrontendRoundTripper(t *testing.T) {
	next := &mockNextTripperware{}
	traces := &mockTracesTripperware{}
	search := &mockSearchTripperware{}

	frontendTripper := newFrontendRoundTripper(next, traces, search, log.NewNopLogger(), prometheus.DefaultRegisterer)

	testCases := []struct {
		name     string
		endpoint string
		response string
	}{
		{
			name:     "next tripper",
			endpoint: "/api/foo",
			response: "next",
		},
		{
			name:     "traces tripper",
			endpoint: apiPathTraces + "/X",
			response: "traces",
		},
		{
			name:     "search tripper",
			endpoint: apiPathSearch + "/X",
			response: "search",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				URL: &url.URL{
					Path: tt.endpoint,
				},
			}
			resp, err := frontendTripper.RoundTrip(req)
			require.NoError(t, err)
			require.NotNil(t, resp)

			body, err := ioutil.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, body, []byte(tt.response))
		})
	}
}
