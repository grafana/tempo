package pipeline

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/dskit/httpgrpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

func TestRetry(t *testing.T) {
	var try atomic.Int32

	for _, tc := range []struct {
		name          string
		handler       http.RoundTripper
		maxRetries    int
		expectedTries int32
		expectedRes   *http.Response
		expectedErr   error
	}{
		{
			name: "retry errors until success",
			handler: RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				if try.Inc() == 5 {
					return &http.Response{StatusCode: 200}, nil
				}
				return nil, errors.New("this request failed")
			}),
			maxRetries:    5,
			expectedTries: 5,
			expectedRes:   &http.Response{StatusCode: 200},
			expectedErr:   nil,
		},
		{
			name: "don't retry 400's",
			handler: RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				try.Inc()
				return &http.Response{StatusCode: 400}, nil
			}),
			maxRetries:    5,
			expectedTries: 1,
			expectedRes:   &http.Response{StatusCode: 400},
			expectedErr:   nil,
		},
		{
			name: "don't retry GRPC request with HTTP 400's",
			handler: RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				try.Inc()
				return nil, httpgrpc.ErrorFromHTTPResponse(&httpgrpc.HTTPResponse{Code: 400})
			}),
			maxRetries:    5,
			expectedTries: 1,
			expectedRes:   nil,
			expectedErr:   httpgrpc.ErrorFromHTTPResponse(&httpgrpc.HTTPResponse{Code: 400}),
		},
		{
			name: "retry 500s",
			handler: RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				try.Inc()
				return &http.Response{StatusCode: 503}, nil
			}),
			maxRetries:    5,
			expectedTries: 5,
			expectedRes:   &http.Response{StatusCode: 503},
			expectedErr:   nil,
		},
		{
			name: "return last error",
			handler: RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				if try.Inc() == 5 {
					return nil, errors.New("request failed")
				}
				return nil, errors.New("not the last request")
			}),
			maxRetries:    5,
			expectedTries: 5,
			expectedRes:   nil,
			expectedErr:   errors.New("request failed"),
		},
		{
			name: "maxRetries=1",
			handler: RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				try.Inc()
				return &http.Response{StatusCode: 500}, nil
			}),
			maxRetries:    1,
			expectedTries: 1,
			expectedRes:   &http.Response{StatusCode: 500},
			expectedErr:   nil,
		},
		{
			name: "maxRetries=0",
			handler: RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				try.Inc()
				return &http.Response{StatusCode: 500}, nil
			}),
			maxRetries:    0,
			expectedTries: 1,
			expectedRes:   &http.Response{StatusCode: 500},
			expectedErr:   nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			try.Store(0)

			retryWare := NewRetryWare(tc.maxRetries, prometheus.NewRegistry())
			handler := retryWare.Wrap(tc.handler)

			req := httptest.NewRequest("GET", "http://example.com", nil)

			res, err := handler.RoundTrip(req)

			require.Equal(t, tc.expectedTries, try.Load())
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedRes, res)
		})
	}
}

func TestRetry_CancelledRequest(t *testing.T) {
	var try atomic.Int32

	// request is cancelled before first call
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://example.com", nil)
	require.NoError(t, err)

	_, err = NewRetryWare(5, prometheus.NewRegistry()).
		Wrap(RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			try.Inc()
			return nil, ctx.Err()
		})).RoundTrip(req)

	require.Equal(t, int32(0), try.Load())
	require.Equal(t, ctx.Err(), err)

	// request is cancelled after first call
	ctx, cancel = context.WithCancel(context.Background())

	req, err = http.NewRequestWithContext(ctx, "GET", "http://example.com", nil)
	require.NoError(t, err)

	_, err = NewRetryWare(5, prometheus.NewRegistry()).
		Wrap(RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			try.Inc()
			cancel()
			return nil, errors.New("this request failed")
		})).RoundTrip(req)

	require.Equal(t, int32(1), try.Load())
	require.Equal(t, ctx.Err(), err)
}
