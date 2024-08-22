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

var nextFunc = AsyncRoundTripperFunc[combiner.PipelineResponse](func(_ Request) (Responses[combiner.PipelineResponse], error) {
	return NewHTTPToAsyncResponse(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader([]byte{})),
	}), nil
})

func TestQueryValidator(t *testing.T) {
	roundTrip := NewQueryValidatorWare().Wrap(nextFunc)
	statusCode := doRequest(t, "http://localhost:8080/api/search", roundTrip)
	assert.Equal(t, 200, statusCode)
}

func TestQueryValidatorForAValidQuery(t *testing.T) {
	roundTrip := NewQueryValidatorWare().Wrap(nextFunc)
	statusCode := doRequest(t, "http://localhost:8080/api/search&q={}", roundTrip)
	assert.Equal(t, 200, statusCode)
}

func TestQueryValidatorForAnInvalidTraceQLQuery(t *testing.T) {
	roundTrip := NewQueryValidatorWare().Wrap(nextFunc)
	statusCode := doRequest(t, "http://localhost:8080/api/search?q={. hi}", roundTrip)
	assert.Equal(t, 400, statusCode)
}

func TestQueryValidatorForAnInvalidTraceQlQueryRegex(t *testing.T) {
	roundTrip := NewQueryValidatorWare().Wrap(nextFunc)
	statusCode := doRequest(t, "http://localhost:8080/api/search?query={span.a =~ \".*((?<!(-test))(?<!(-uat)))$\"}", roundTrip)
	assert.Equal(t, 400, statusCode)
}

func doRequest(t *testing.T, url string, rt AsyncRoundTripper[combiner.PipelineResponse]) int {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	resp, _ := rt.RoundTrip(NewHTTPRequest(req))
	httpResponse, _, err := resp.Next(context.Background())
	require.NoError(t, err)
	return httpResponse.HTTPResponse().StatusCode
}
