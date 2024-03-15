package pipeline

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"go.uber.org/goleak"
)

func TestNewSyncToAsyncResponse(t *testing.T) {
	expected := &http.Response{
		Header: http.Header{
			"foo": []string{"bar"},
		},
		StatusCode: http.StatusAlreadyReported,
		Status:     http.StatusText(http.StatusEarlyHints),
		Body:       nil,
	}

	asyncR := NewSyncToAsyncResponse(expected)

	// confirm we get back what we put in
	actual, done, err := asyncR.Next(context.Background())
	require.True(t, done)
	require.NoError(t, err)
	require.Equal(t, expected, actual)

	// confirm errored context is honored
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	actual, done, err = asyncR.Next(ctx)
	require.True(t, done)
	require.Error(t, err)
	require.Nil(t, actual)

	// confirm bad request is expected
	asyncR = NewBadRequest(errors.New("foo"))
	expected = &http.Response{
		StatusCode: http.StatusBadRequest,
		Status:     http.StatusText(http.StatusBadRequest),
		Body:       io.NopCloser(strings.NewReader("foo")),
	}
	actual, done, err = asyncR.Next(context.Background())
	require.True(t, done)
	require.NoError(t, err)
	require.Equal(t, expected, actual)

	// confirm successful response is expected
	asyncR = NewSuccessfulResponse("foo")
	expected = &http.Response{
		StatusCode: http.StatusOK,
		Status:     http.StatusText(http.StatusOK),
		Body:       io.NopCloser(strings.NewReader("foo")),
	}
	actual, done, err = asyncR.Next(context.Background())
	require.True(t, done)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestAsyncResponseReturnsResponsesInOrder(t *testing.T) {
	// create a slice of responses and send them through
	//  an async response
	expected := []*http.Response{
		{
			StatusCode: http.StatusAccepted,
			Status:     http.StatusText(http.StatusAccepted),
			Body:       io.NopCloser(strings.NewReader("foo")),
		},
		{
			StatusCode: http.StatusAlreadyReported,
			Status:     http.StatusText(http.StatusAlreadyReported),
			Body:       io.NopCloser(strings.NewReader("bar")),
		},
		{
			StatusCode: http.StatusContinue,
			Status:     http.StatusText(http.StatusContinue),
			Body:       io.NopCloser(strings.NewReader("baz")),
		},
	}

	asyncR := newAsyncResponse()
	go func() {
		for _, r := range expected {
			asyncR.Send(NewSyncToAsyncResponse(r))
		}
		asyncR.SendComplete()
	}()

	// confirm we get back what we put in
	for _, e := range expected {
		actual, done, err := asyncR.Next(context.Background())
		require.False(t, done)
		require.NoError(t, err)
		require.Equal(t, e, actual)
	}

	// next call should be done
	actual, done, err := asyncR.Next(context.Background())
	require.True(t, done)
	require.NoError(t, err)
	require.Nil(t, actual)
}

func TestAsyncResponseHonorsContextFailure(t *testing.T) {
	asyncR := newAsyncResponse()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	actual, done, err := asyncR.Next(ctx)
	require.True(t, done)
	require.Error(t, err)
	require.Nil(t, actual)
}

func TestAsyncResponseReturnsSentErrors(t *testing.T) {
	asyncR := newAsyncResponse()
	expectedErr := errors.New("foo")
	// send a real response and an error and confirm errors are preferred
	go func() {
		asyncR.SendError(expectedErr)
	}()
	go func() {
		asyncR.Send(NewSuccessfulResponse("foo"))
	}()
	time.Sleep(100 * time.Millisecond)
	actual, done, actualErr := asyncR.Next(context.Background())
	require.True(t, done)
	require.Equal(t, expectedErr, actualErr)
	require.Nil(t, actual)

	// make sure that responses continues to return the error
	actual, done, actualErr = asyncR.Next(context.Background())
	require.True(t, done)
	require.Equal(t, expectedErr, actualErr)
	require.Nil(t, actual)
}

func TestAsyncResponseFansIn(t *testing.T) {
	leakOpts := goleak.IgnoreCurrent()

	// create a random hierarchy of async responses and add a bunch of responses.
	// count the added responses and confirm the number we pull is the same.
	wg := sync.WaitGroup{}
	rootResp := newAsyncResponse()

	expected := 0
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer rootResp.SendComplete()

		expected = addResponses(rootResp)
	}()

	actual := 0
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer rootResp.NextComplete()

		for {
			resp, done, err := rootResp.Next(context.Background())
			if done {
				return
			}
			actual++
			require.NoError(t, err)
			require.NotNil(t, resp)
		}
	}()

	wg.Wait()
	require.Equal(t, expected, actual)

	goleak.VerifyNone(t, leakOpts)
}

func addResponses(r *asyncResponse) int {
	responsesToAdd := rand.Intn(5)
	childResponse := newAsyncResponse()
	defer childResponse.SendComplete()

	r.Send(childResponse)
	for i := 0; i < responsesToAdd; i++ {
		childResponse.Send(NewSyncToAsyncResponse(&http.Response{}))
	}

	recurse := rand.Intn(2)%2 == 0
	if recurse {
		return responsesToAdd + addResponses(childResponse)
	}

	return responsesToAdd
}

func TestAsyncResponsesDoesNotLeak(t *testing.T) {
	tcs := []struct {
		name    string
		finalRT func(requestCancel context.CancelFunc) RoundTripperFunc
		cleanup func()
	}{
		{
			name: "happy path",
			finalRT: func(_ context.CancelFunc) RoundTripperFunc {
				return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
					return &http.Response{
						Body: io.NopCloser(strings.NewReader("foo")),
					}, nil
				})
			},
		},
		{
			name: "error path",
			finalRT: func(_ context.CancelFunc) RoundTripperFunc {
				return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
					return nil, errors.New("foo")
				})
			},
		},
		{
			name: "combiner bails early",
			finalRT: func(_ context.CancelFunc) RoundTripperFunc {
				responseCounter := atomic.Int32{}

				return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
					counter := responseCounter.Add(1)
					if counter == 2 {
						return &http.Response{
							StatusCode: http.StatusNotFound, // force the combiner to bail on the second response
							Body:       io.NopCloser(strings.NewReader("foo")),
						}, nil
					}

					return &http.Response{
						Body: io.NopCloser(strings.NewReader("foo")),
					}, nil
				})
			},
		},
		{
			name: "context cancelled before returned responses",
			finalRT: func(cancel context.CancelFunc) RoundTripperFunc {
				go func() {
					time.Sleep(1 * time.Second)
					cancel()
				}()

				return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
					time.Sleep(3 * time.Second)

					return &http.Response{
						Body: io.NopCloser(strings.NewReader("foo")),
					}, nil
				})
			},
			cleanup: func() {
				time.Sleep(5 * time.Second) // allow all responses to come through
			},
		},
		{
			name: "context cancelled in between returned responses",
			finalRT: func(cancel context.CancelFunc) RoundTripperFunc {
				responseCounter := atomic.Int32{}

				return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
					counter := responseCounter.Add(1)
					if counter == 2 {
						cancel()
					}

					return &http.Response{
						Body: io.NopCloser(strings.NewReader("foo")),
					}, nil
				})
			},
			cleanup: func() {
				time.Sleep(2 * time.Second) // allow all responses to come through
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// http
			t.Run("http", func(t *testing.T) {
				leakOpts := goleak.IgnoreCurrent()
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				req, err := http.NewRequestWithContext(ctx, "GET", "http://foo.com", nil)
				require.NoError(t, err)

				bridge := &pipelineBridge{
					next: tc.finalRT(cancel),
				}
				httpCollector := NewHTTPCollector(sharder{next: bridge}, combiner.NewNoOp())

				_, _ = httpCollector.RoundTrip(req)

				if tc.cleanup != nil {
					tc.cleanup()
				}

				goleak.VerifyNone(t, leakOpts)
			})

			//grpc
			t.Run("grpc", func(t *testing.T) {
				leakOpts := goleak.IgnoreCurrent()
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				req, err := http.NewRequestWithContext(ctx, "GET", "http://foo.com", nil)
				require.NoError(t, err)

				bridge := &pipelineBridge{
					next: tc.finalRT(cancel),
				}
				grpcCollector := NewGRPCCollector[*tempopb.SearchResponse](sharder{next: bridge}, combiner.NewNoOp().(combiner.GRPCCombiner[*tempopb.SearchResponse]), func(sr *tempopb.SearchResponse) error { return nil })

				_ = grpcCollector.RoundTrip(req)

				if tc.cleanup != nil {
					tc.cleanup()
				}

				goleak.VerifyNone(t, leakOpts)
			})

			//multiple sharder tiers
			t.Run("multiple sharder tiers", func(t *testing.T) {
				leakOpts := goleak.IgnoreCurrent()
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				req, err := http.NewRequestWithContext(ctx, "GET", "http://foo.com", nil)
				require.NoError(t, err)

				bridge := &pipelineBridge{
					next: tc.finalRT(cancel),
				}

				s := sharder{next: sharder{next: bridge}, funcSharder: true}
				//s := sharder{next: sharder{next: bridge, funcSharder: true}}
				grpcCollector := NewGRPCCollector[*tempopb.SearchResponse](s, combiner.NewNoOp().(combiner.GRPCCombiner[*tempopb.SearchResponse]), func(sr *tempopb.SearchResponse) error { return nil })

				_ = grpcCollector.RoundTrip(req)

				if tc.cleanup != nil {
					tc.cleanup()
				}

				goleak.VerifyNone(t, leakOpts)
			})
		})
	}
}

type sharder struct {
	next        AsyncRoundTripper[*http.Response]
	funcSharder bool
}

func (s sharder) RoundTrip(r *http.Request) (Responses[*http.Response], error) {
	total := 4
	concurrent := 2

	// execute requests
	if s.funcSharder {
		return NewAsyncSharderFunc(concurrent, total, func(i int) *http.Request {
			return r
		}, s.next), nil
	}

	reqCh := make(chan *http.Request)
	go func() {
		for i := 0; i < total; i++ {
			reqCh <- r
		}
		close(reqCh)
	}()
	return NewAsyncSharderChan(concurrent, reqCh, nil, s.next), nil
}

func BenchmarkNewSyncToAsyncResponse(b *testing.B) {
	r := &http.Response{}
	for i := 0; i < b.N; i++ {
		foo := NewSyncToAsyncResponse(r)
		_ = foo
	}
}
