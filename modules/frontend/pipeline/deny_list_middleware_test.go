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

var next = AsyncRoundTripperFunc[combiner.PipelineResponse](func(_ Request) (Responses[combiner.PipelineResponse], error) {
	return NewHTTPToAsyncResponse(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader([]byte{})),
	}), nil
})

func TestURLBlackListMiddlewareForEmptyBlackList(t *testing.T) {
	regexes := []string{}
	roundTrip := NewURLDenyListWare(regexes).Wrap(next)
	statusCode := DoRequest(t, "http://localhost:8080/api/v2/traces/123345", roundTrip)
	assert.Equal(t, 200, statusCode)
}

func TestURLBlackListMiddlewarePanicsOnSyntacticallyIncorrectRegex(t *testing.T) {
	regexes := []string{"qr/^(.*\\.traces\\/[a-f0-9]{32}$/"}
	assert.Panics(t, func() {
		NewURLDenyListWare(regexes).Wrap(next)
	})
}

func TestURLBlackListMiddleware(t *testing.T) {
	regexes := []string{
		"^.*v2.*",
	}
	roundTrip := NewURLDenyListWare(regexes).Wrap(next)
	statusCode := DoRequest(t, "http://localhost:9000", roundTrip)
	assert.Equal(t, 200, statusCode)

	// Blacklisted url
	statusCode = DoRequest(t, "http://localhost:8080/api/v2/traces/123345", roundTrip)
	assert.Equal(t, 400, statusCode)
}

func DoRequest(t *testing.T, url string, rt AsyncRoundTripper[combiner.PipelineResponse]) int {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	resp, _ := rt.RoundTrip(NewHTTPRequest(req))
	httpResponse, _, err := resp.Next(context.Background())
	require.NoError(t, err)
	return httpResponse.HTTPResponse().StatusCode
}
