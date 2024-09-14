package pipeline

import (
	"context"
	"net/http"

	"github.com/grafana/tempo/modules/frontend/combiner"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("modules/frontend/pipeline")

type Request interface {
	HTTPRequest() *http.Request
	Context() context.Context
	WithContext(context.Context)

	SetCacheKey(string)
	CacheKey() string

	SetResponseData(any) // add data that will be sent back with this requests response
	ResponseData() any
}

type HTTPRequest struct {
	req *http.Request

	cacheKey     string
	responseData any
}

func NewHTTPRequest(req *http.Request) *HTTPRequest {
	return &HTTPRequest{req: req}
}

func (r HTTPRequest) HTTPRequest() *http.Request {
	return r.req
}

func (r HTTPRequest) Context() context.Context {
	if r.req == nil {
		return nil
	}

	return r.req.Context()
}

func (r *HTTPRequest) WithContext(ctx context.Context) {
	r.req = r.req.WithContext(ctx)
}

func (r *HTTPRequest) SetCacheKey(s string) {
	r.cacheKey = s
}

func (r *HTTPRequest) CacheKey() string {
	return r.cacheKey
}

func (r *HTTPRequest) SetResponseData(data any) {
	r.responseData = data
}

func (r *HTTPRequest) ResponseData() any {
	return r.responseData
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

type RoundTripperFunc func(Request) (*http.Response, error)

// RoundTrip implememnts RoundTripper
func (fn RoundTripperFunc) RoundTrip(req Request) (*http.Response, error) {
	return fn(req)
}

type RoundTripper interface {
	RoundTrip(Request) (*http.Response, error)
}

// Middleware is used to build pipelines of pipeline.Roundtrippers
type Middleware interface {
	Wrap(RoundTripper) RoundTripper
}

// MiddlewareFunc is like http.HandlerFunc, but for Middleware.
type MiddlewareFunc func(RoundTripper) RoundTripper

// Wrap implements Middleware.
func (f MiddlewareFunc) Wrap(w RoundTripper) RoundTripper {
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

	syncPipeline := MiddlewareFunc(func(next RoundTripper) RoundTripper {
		for i := len(mw) - 1; i >= 0; i-- {
			next = mw[i].Wrap(next)
		}
		return next
	})

	// bridge the two pipelines
	bridge := &pipelineBridge{
		next: syncPipeline.Wrap(RoundTripperFunc(func(req Request) (*http.Response, error) {
			return next.RoundTrip(req.HTTPRequest())
		})),
		convert: NewHTTPToAsyncResponse,
	}
	return asyncPipeline.Wrap(bridge)
}

var _ AsyncRoundTripper[combiner.PipelineResponse] = (*pipelineBridge)(nil)

type pipelineBridge struct {
	next    RoundTripper
	convert func(*http.Response) Responses[combiner.PipelineResponse]
}

func (b *pipelineBridge) RoundTrip(req Request) (Responses[combiner.PipelineResponse], error) {
	r, err := b.next.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// check for request data in the context and echo it back if it exists
	if val := req.ResponseData(); val != nil {
		return NewHTTPToAsyncResponseWithRequestData(r, val), nil
	}

	return NewHTTPToAsyncResponse(r), nil
}
