package pipeline

import (
	"context"
	"io"
	"net/http"
	"strings"
)

type Responses[T any] interface {
	Next(context.Context) (T, bool, error) // bool = done
}

var _ Responses[*http.Response] = syncResponse{}

// syncResponse is a single http.Response that implements the Responses[*http.Response] interface.
type syncResponse struct {
	r *http.Response
}

// NewSyncToAsyncResponse creates a new AsyncResponse that wraps a single http.Response.
func NewSyncToAsyncResponse(r *http.Response) Responses[*http.Response] {
	return syncResponse{
		r: r,
	}
}

// NewBadRequest creates a new AsyncResponse that wraps a single http.Response with a 400 status code and the provided error message.
func NewBadRequest(err error) Responses[*http.Response] {
	return NewSyncToAsyncResponse(&http.Response{
		StatusCode: http.StatusBadRequest,
		Status:     http.StatusText(http.StatusBadRequest),
		Body:       io.NopCloser(strings.NewReader(err.Error())),
	})
}

// NewSuccessfulResponse creates a new AsyncResponse that wraps a single http.Response with a 200 status code and the provided body.
func NewSuccessfulResponse(body string) Responses[*http.Response] {
	return NewSyncToAsyncResponse(&http.Response{
		StatusCode: http.StatusOK,
		Status:     http.StatusText(http.StatusOK),
		Body:       io.NopCloser(strings.NewReader(body)),
	})
}

func (s syncResponse) Next(ctx context.Context) (*http.Response, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, true, err
	}

	return s.r, true, nil
}

var _ Responses[*http.Response] = &asyncResponse{}

// asyncResponse supports a fan in of a variable number of http.Responses.
type asyncResponse struct {
	respChan chan Responses[*http.Response]
	errChan  chan error

	curResponses Responses[*http.Response]
}

func newAsyncResponse() *asyncResponse {
	return &asyncResponse{
		respChan: make(chan Responses[*http.Response]),
		errChan:  make(chan error, 1),
	}
}

func (a *asyncResponse) Send(r Responses[*http.Response]) {
	a.respChan <- r
}

// SendError sends an error to the asyncResponse. This will cause the asyncResponse to return the error on the next call to Next.
func (a *asyncResponse) SendError(err error) {
	select {
	case a.errChan <- err:
	default:
	}
}

func (a *asyncResponse) done() {
	close(a.respChan)
}

// Next returns the next http.Response or an error if one is available. It always prefers an error over a response.
// todo: review performance. There is a lot of channel access here
func (a *asyncResponse) Next(ctx context.Context) (*http.Response, bool, error) {
	for {
		// explicitly check the err channel first. selects are non-deterministic and
		// if there is an error we want to prioritize it over a valid response
		if err := a.error(); err != nil {
			return nil, true, err
		}

		// grab a new AsyncResponse if we don't have one
		if a.curResponses == nil {
			select {
			case err := <-a.errChan:
				return nil, true, err
			case <-ctx.Done():
				return nil, true, ctx.Err()
			case r, ok := <-a.respChan:
				a.curResponses = r
				if r == nil && !ok {
					// this AsyncResponse is completely exhausted
					return nil, true, a.error()
				}
			}
		}

		// get the next response from the current AsyncResponse
		resp, done, err := a.curResponses.Next(ctx)
		if done {
			a.curResponses = nil
		}

		// if the response is nil and there is no error, continue to the next AsyncResponse
		if err == nil && resp == nil {
			continue
		}

		return resp, false, err
	}
}

func (a *asyncResponse) error() error {
	select {
	case err := <-a.errChan:
		return err
	default:
	}

	return nil
}
