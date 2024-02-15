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

	"github.com/stretchr/testify/require"
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
		asyncR.done()
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
}

func TestAsyncResponseFansIn(t *testing.T) {
	// create a random hierarchy of async responses and add a bunch of responses.
	// count the added responses and confirm the number we pull is the same.
	wg := sync.WaitGroup{}
	rootResp := newAsyncResponse()

	expected := 0
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer rootResp.done()

		expected = addResponses(rootResp)
	}()

	actual := 0
	wg.Add(1)
	go func() {
		defer wg.Done()

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
}

func addResponses(r *asyncResponse) int {
	responsesToAdd := rand.Intn(5)
	childResponse := newAsyncResponse()
	defer childResponse.done()

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

func BenchmarkNewSyncToAsyncResponse(b *testing.B) {
	r := &http.Response{}
	for i := 0; i < b.N; i++ {
		foo := NewSyncToAsyncResponse(r)
		_ = foo
	}
}
