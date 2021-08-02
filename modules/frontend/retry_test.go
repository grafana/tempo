package frontend

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

type HandlerFunc func(req *http.Request) (*http.Response, error)

// Wrap implements Handler.
func (q HandlerFunc) Do(req *http.Request) (*http.Response, error) {
	return q(req)
}

func TestRetry(t *testing.T) {
	var try atomic.Int32

	for _, tc := range []struct {
		name          string
		handler       Handler
		maxRetries    int
		expectedTries int32
		expectedRes   *http.Response
		expectedErr   error
	}{
		{
			name: "retry until success",
			handler: HandlerFunc(func(req *http.Request) (*http.Response, error) {
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
			handler: HandlerFunc(func(req *http.Request) (*http.Response, error) {
				try.Inc()
				return &http.Response{StatusCode: 400}, nil
			}),
			maxRetries:    5,
			expectedTries: 1,
			expectedRes:   &http.Response{StatusCode: 400},
			expectedErr:   nil,
		},
		{
			name: "retry 500s",
			handler: HandlerFunc(func(req *http.Request) (*http.Response, error) {
				try.Inc()
				return &http.Response{StatusCode: 500}, nil
			}),
			maxRetries:    5,
			expectedTries: 5,
			expectedRes:   &http.Response{StatusCode: 500},
			expectedErr:   nil,
		},
		{
			name: "return last error",
			handler: HandlerFunc(func(req *http.Request) (*http.Response, error) {
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
			handler: HandlerFunc(func(req *http.Request) (*http.Response, error) {
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
			handler: HandlerFunc(func(req *http.Request) (*http.Response, error) {
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

			retryWare := RetryWare(tc.maxRetries, log.NewNopLogger())
			handler := retryWare.Wrap(tc.handler)

			req := httptest.NewRequest("GET", "http://example.com", nil)

			res, err := handler.Do(req)

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

	_, err = RetryWare(5, log.NewNopLogger()).
		Wrap(HandlerFunc(func(req *http.Request) (*http.Response, error) {
			try.Inc()
			return nil, ctx.Err()
		})).
		Do(req)

	require.Equal(t, int32(0), try.Load())
	require.Equal(t, ctx.Err(), err)

	// request is cancelled after first call
	ctx, cancel = context.WithCancel(context.Background())

	req, err = http.NewRequestWithContext(ctx, "GET", "http://example.com", nil)
	require.NoError(t, err)

	_, err = RetryWare(5, log.NewNopLogger()).
		Wrap(HandlerFunc(func(req *http.Request) (*http.Response, error) {
			try.Inc()
			cancel()
			return nil, errors.New("this request failed")
		})).
		Do(req)

	require.Equal(t, int32(1), try.Load())
	require.Equal(t, ctx.Err(), err)
}
