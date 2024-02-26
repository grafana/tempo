package pipeline

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAsyncSharders(t *testing.T) {
	expectedRequestCount := 10000

	tcs := []struct {
		name       string
		responseFn func(next AsyncRoundTripper[*http.Response]) *asyncResponse
	}{
		{
			name: "AsyncSharder",
			responseFn: func(next AsyncRoundTripper[*http.Response]) *asyncResponse {
				return NewAsyncSharderFunc(10, expectedRequestCount, func(i int) *http.Request {
					if i >= expectedRequestCount {
						return nil
					}
					return &http.Request{}
				}, next).(*asyncResponse)
			},
		},
		{
			name: "AsyncSharder - no limit",
			responseFn: func(next AsyncRoundTripper[*http.Response]) *asyncResponse {
				return NewAsyncSharderFunc(0, expectedRequestCount, func(i int) *http.Request {
					if i >= expectedRequestCount {
						return nil
					}
					return &http.Request{}
				}, next).(*asyncResponse)
			},
		}, {
			name: "AsyncSharderLimitedGoroutines",
			responseFn: func(next AsyncRoundTripper[*http.Response]) *asyncResponse {
				reqChan := make(chan *http.Request)
				go func() {
					for i := 0; i < expectedRequestCount; i++ {
						reqChan <- &http.Request{}
					}
					close(reqChan)
				}()

				return NewAsyncSharderChan(10, reqChan, nil, next).(*asyncResponse)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			next := AsyncRoundTripperFunc[*http.Response](func(r *http.Request) (Responses[*http.Response], error) {
				// return a generic 200
				return NewSyncToAsyncResponse(&http.Response{
					Body:       io.NopCloser(strings.NewReader("")),
					StatusCode: 200,
				}), nil
			})

			sharderResp := tc.responseFn(next)

			// drain and count the requests
			wg := &sync.WaitGroup{}
			wg.Add(1)
			actualRequestCount := 0
			go func() {
				defer wg.Done()
				for {
					resp, done, err := sharderResp.Next(context.Background())
					if resp != nil {
						actualRequestCount++
					}
					require.NoError(t, err)
					if done {
						break
					}

				}
			}()

			wg.Wait()
			require.Equal(t, expectedRequestCount, actualRequestCount)
		})
	}
}
