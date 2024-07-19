package pipeline

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdjustsResponseCode(t *testing.T) {
	nextFn := func(status int) RoundTripper {
		return RoundTripperFunc(func(req Request) (*http.Response, error) {
			return &http.Response{StatusCode: status}, nil
		})
	}

	tcs := []struct {
		actualCode   int
		expectedCode int
	}{
		{actualCode: 400, expectedCode: 500},
		{actualCode: 429, expectedCode: 429},
		{actualCode: 500, expectedCode: 500},
		{actualCode: 200, expectedCode: 200},
		{actualCode: 418, expectedCode: 500},
	}

	for _, tc := range tcs {
		retryWare := NewStatusCodeAdjustWare()
		handler := retryWare.Wrap(nextFn(tc.actualCode))
		req := httptest.NewRequest("GET", "http://example.com", nil)
		res, err := handler.RoundTrip(NewHTTPRequest(req))

		require.NoError(t, err)
		require.Equal(t, res.StatusCode, tc.expectedCode)
	}
}

func TestAdjustsResponseCodeTeapotAllowed(t *testing.T) {
	nextFn := func(status int) RoundTripper {
		return RoundTripperFunc(func(req Request) (*http.Response, error) {
			return &http.Response{StatusCode: status}, nil
		})
	}

	tcs := []struct {
		actualCode   int
		expectedCode int
	}{
		{actualCode: 400, expectedCode: 500},
		{actualCode: 429, expectedCode: 429},
		{actualCode: 500, expectedCode: 500},
		{actualCode: 200, expectedCode: 200},
		{actualCode: 418, expectedCode: 418},
	}

	for _, tc := range tcs {
		retryWare := NewStatusCodeAdjustWareWithAllowedCode(418)
		handler := retryWare.Wrap(nextFn(tc.actualCode))
		req := httptest.NewRequest("GET", "http://example.com", nil)
		res, err := handler.RoundTrip(NewHTTPRequest(req))

		require.NoError(t, err)
		require.Equal(t, res.StatusCode, tc.expectedCode)
	}
}
