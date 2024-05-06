package pipeline

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/grafana/tempo/modules/frontend/combiner"
	"go.uber.org/atomic"
)

type Responses[T any] interface {
	// Next returns the next response or an error if one is available. It always prefers an error over a response.
	Next(context.Context) (T, bool, error) // bool = done
}

var _ Responses[combiner.PipelineResponse] = syncResponse{}

type pipelineResponse struct {
	r              *http.Response
	additionalData any
}

func (p pipelineResponse) HTTPResponse() *http.Response {
	return p.r
}

func (p pipelineResponse) AdditionalData() any {
	return p.additionalData
}

// syncResponse is a single http.Response that implements the Responses[*http.Response] interface.
type syncResponse struct {
	r combiner.PipelineResponse
}

// NewHTTPToAsyncResponse creates a new AsyncResponse that wraps a single http.Response.
func NewHTTPToAsyncResponse(r *http.Response) Responses[combiner.PipelineResponse] {
	return syncResponse{
		r: pipelineResponse{
			r:              r,
			additionalData: nil,
		},
	}
}

func NewHTTPToAsyncResponseWithAdditionalData(r *http.Response, additionalData any) Responses[combiner.PipelineResponse] {
	return syncResponse{
		r: pipelineResponse{
			r:              r,
			additionalData: additionalData,
		},
	}
}

// NewBadRequest creates a new AsyncResponse that wraps a single http.Response with a 400 status code and the provided error message.
func NewBadRequest(err error) Responses[combiner.PipelineResponse] {
	return NewHTTPToAsyncResponse(&http.Response{
		StatusCode: http.StatusBadRequest,
		Status:     http.StatusText(http.StatusBadRequest),
		Body:       io.NopCloser(strings.NewReader(err.Error())),
	})
}

// NewSuccessfulResponse creates a new AsyncResponse that wraps a single http.Response with a 200 status code and the provided body.
func NewSuccessfulResponse(body string) Responses[combiner.PipelineResponse] {
	return NewHTTPToAsyncResponse(&http.Response{
		StatusCode: http.StatusOK,
		Status:     http.StatusText(http.StatusOK),
		Body:       io.NopCloser(strings.NewReader(body)),
	})
}

func (s syncResponse) Next(ctx context.Context) (combiner.PipelineResponse, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, true, err
	}

	return s.r, true, nil
}

var _ Responses[combiner.PipelineResponse] = &asyncResponse{}

// asyncResponse supports a fan in of a variable number of http.Responses.
type asyncResponse struct {
	respChan chan Responses[combiner.PipelineResponse]
	errChan  chan error
	err      *atomic.Error

	curResponses Responses[combiner.PipelineResponse]
}

func newAsyncResponse() *asyncResponse {
	return &asyncResponse{
		respChan: make(chan Responses[combiner.PipelineResponse]),
		errChan:  make(chan error, 1),
		err:      atomic.NewError(nil),
	}
}

func (a *asyncResponse) Send(ctx context.Context, r Responses[combiner.PipelineResponse]) {
	select {
	case <-ctx.Done():
	case a.respChan <- r:
	}
}

// SendError sends an error to the asyncResponse. This will cause the asyncResponse to return the error on the next call to Next.
// we send on a channel to give errors the chance to unblock the select below. we also store in an atomic error so that
// a Responses in error will always remain in error
func (a *asyncResponse) SendError(err error) {
	select {
	case a.errChan <- err:
		a.err.Store(err)
	default:
	}
}

// SendComplete indicates the sender is done. We close the channel to give a clear signal to the consumer
func (a *asyncResponse) SendComplete() {
	close(a.respChan)
}

// Next returns the next http.Response or an error if one is available. It always prefers an error over a response.
// todo: review performance. There is a lot of channel access here
func (a *asyncResponse) Next(ctx context.Context) (combiner.PipelineResponse, bool, error) {
	for {
		// explicitly check the err channel first. selects are non-deterministic and
		// if there is an error we want to prioritize it over a valid response
		if err := a.err.Load(); err != nil {
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
					return nil, true, a.err.Load()
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
