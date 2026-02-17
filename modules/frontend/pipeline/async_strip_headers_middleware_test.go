package pipeline

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/stretchr/testify/require"
)

func TestStripHeaders(t *testing.T) {
	tcs := []struct {
		name     string
		allow    []string
		headers  map[string][]string
		expected http.Header
	}{
		{
			name:     "empty allow list",
			allow:    []string{},
			headers:  map[string][]string{"header1": {"value1"}, "header2": {"value2"}},
			expected: map[string][]string{},
		},
		{
			name:     "allow list with one header",
			allow:    []string{"header1"},
			headers:  map[string][]string{"header1": {"value1"}, "header2": {"value2"}},
			expected: map[string][]string{"header1": {"value1"}},
		},
		{
			name:     "allow list is case-insensitive",
			allow:    []string{"x-plugin-id", "AUTHORIZATION"},
			headers:  map[string][]string{"X-Plugin-Id": {"grafana-assistant"}, "authorization": {"Bearer abc"}, "X-Drop-Me": {"nope"}},
			expected: map[string][]string{"X-Plugin-Id": {"grafana-assistant"}, "authorization": {"Bearer abc"}},
		},
		{
			name:     "allow list keeps uppercase non-canonical incoming header",
			allow:    []string{"x-plugin-id", "AUTHORIZATION"},
			headers:  map[string][]string{"X-PLUGIN-ID": {"grafana-assistant"}, "authorization": {"Bearer abc"}, "X-Drop-Me": {"nope"}},
			expected: map[string][]string{"X-PLUGIN-ID": {"grafana-assistant"}, "authorization": {"Bearer abc"}},
		},
		{
			name:     "allow list keeps lowercase non-canonical incoming header",
			allow:    []string{"x-plugin-id", "AUTHORIZATION"},
			headers:  map[string][]string{"x-plugin-id": {"grafana-assistant"}, "authorization": {"Bearer abc"}, "X-Drop-Me": {"nope"}},
			expected: map[string][]string{"x-plugin-id": {"grafana-assistant"}, "authorization": {"Bearer abc"}},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			next := AsyncRoundTripperFunc[combiner.PipelineResponse](func(req Request) (Responses[combiner.PipelineResponse], error) {
				actualHeaders := req.HTTPRequest().Header
				require.Equal(t, tc.expected, actualHeaders)

				return NewHTTPToAsyncResponse(&http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewReader([]byte{})),
				}), nil
			})

			stripHeaders := NewStripHeadersWare(tc.allow).Wrap(next)

			req, _ := http.NewRequest(http.MethodGet, "http://localhost:8080", nil)
			req.Header = tc.headers

			_, err := stripHeaders.RoundTrip(NewHTTPRequest(req))
			require.NoError(t, err)
		})
	}
}
