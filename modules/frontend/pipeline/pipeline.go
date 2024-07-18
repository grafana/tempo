package pipeline

import (
	"context"
	"net/http"

	"github.com/grafana/tempo/modules/frontend/combiner"
)

type Request interface {
	HTTPRequest() *http.Request
	Context() context.Context
}

type HTTPRequest struct {
	req *http.Request
}

func NewHTTPRequest(req *http.Request) HTTPRequest {
	return HTTPRequest{req: req}
}

func (r HTTPRequest) HTTPRequest() *http.Request {
	return r.req // jpe ?
}

func (r HTTPRequest) Context() context.Context {
	if r.req == nil {
		return nil
	}

	return r.req.Context()
}

//
// Async Pipeline
//

type AsyncRoundTripper[T any] interface {
	RoundTrip(Request) (Responses[T], error)
}

type AsyncRoundTripperFunc[T any] func(Request) (Responses[T], error)

func (fn AsyncRoundTripperFunc[T]) RoundTrip(req Request) (Responses[T], error) {
	return fn(req)
}

// AsyncMiddleware is used to build pipelines of pipeline.Roundtrippers
type AsyncMiddleware[T any] interface {
	Wrap(AsyncRoundTripper[T]) AsyncRoundTripper[T]
}

// AsyncMiddlewareFunc is like http.HandlerFunc, but for Middleware.
type AsyncMiddlewareFunc[T any] func(AsyncRoundTripper[T]) AsyncRoundTripper[T]

// Wrap implements Middleware.
func (f AsyncMiddlewareFunc[T]) Wrap(w AsyncRoundTripper[T]) AsyncRoundTripper[T] {
	return f(w)
}

//
// Sync Pipeline
//

type RoundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip implememnts http.RoundTripper
func (fn RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

// Middleware is used to build pipelines of pipeline.Roundtrippers
type Middleware interface {
	Wrap(http.RoundTripper) http.RoundTripper
}

// MiddlewareFunc is like http.HandlerFunc, but for Middleware.
type MiddlewareFunc func(http.RoundTripper) http.RoundTripper

// Wrap implements Middleware.
func (f MiddlewareFunc) Wrap(w http.RoundTripper) http.RoundTripper {
	return f(w)
}

//
// Builder and Bridge
//

// Build takes a slice of async, sync middleware and a http.RoundTripper and builds a request pipeline
func Build(asyncMW []AsyncMiddleware[combiner.PipelineResponse], mw []Middleware, next http.RoundTripper) AsyncRoundTripper[combiner.PipelineResponse] {
	asyncPipeline := AsyncMiddlewareFunc[combiner.PipelineResponse](func(next AsyncRoundTripper[combiner.PipelineResponse]) AsyncRoundTripper[combiner.PipelineResponse] {
		for i := len(asyncMW) - 1; i >= 0; i-- {
			next = asyncMW[i].Wrap(next)
		}
		return next
	})

	syncPipeline := MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		for i := len(mw) - 1; i >= 0; i-- {
			next = mw[i].Wrap(next)
		}
		return next
	})

	// bridge the two pipelines
	bridge := &pipelineBridge{
		next:    syncPipeline.Wrap(next),
		convert: NewHTTPToAsyncResponse,
	}
	return asyncPipeline.Wrap(bridge)
}

var _ AsyncRoundTripper[combiner.PipelineResponse] = (*pipelineBridge)(nil)

type pipelineBridge struct {
	next    http.RoundTripper
	convert func(*http.Response) Responses[combiner.PipelineResponse]
}

func (b *pipelineBridge) RoundTrip(req Request) (Responses[combiner.PipelineResponse], error) {
	httpReq := req.HTTPRequest()

	r, err := b.next.RoundTrip(httpReq)
	if err != nil {
		return nil, err
	}

	// check for request data in the context and echo it back if it exists
	if val := httpReq.Context().Value(contextRequestDataForResponse); val != nil {
		return NewHTTPToAsyncResponseWithRequestData(r, val), nil
	}

	return NewHTTPToAsyncResponse(r), nil
}
