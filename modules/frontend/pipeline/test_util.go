package pipeline

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/stretchr/testify/require"
)

func DoRequest(t *testing.T, url string, rt AsyncRoundTripper[combiner.PipelineResponse]) int {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	resp, _ := rt.RoundTrip(NewHTTPRequest(req))
	httpResponse, _, err := resp.Next(context.Background())
	require.NoError(t, err)
	return httpResponse.HTTPResponse().StatusCode
}

func GetRoundTripperFunc() AsyncRoundTripperFunc[combiner.PipelineResponse] {
	next := AsyncRoundTripperFunc[combiner.PipelineResponse](func(_ Request) (Responses[combiner.PipelineResponse], error) {
		return NewHTTPToAsyncResponse(&http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader([]byte{})),
		}), nil
	})
	return next
}

func GetRoundTripperFuncWithAsserts(t *testing.T, assert func(t *testing.T, r Request)) AsyncRoundTripperFunc[combiner.PipelineResponse] {
	next := AsyncRoundTripperFunc[combiner.PipelineResponse](func(req Request) (Responses[combiner.PipelineResponse], error) {
		assert(t, req)
		return NewHTTPToAsyncResponse(&http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader([]byte{})),
		}), nil
	})
	return next
}
