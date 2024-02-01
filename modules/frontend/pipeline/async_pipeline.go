package pipeline

import (
	"context"
	"net/http"
)

// sync call with one response just returns a Response interface above. maybe a small wrapper to fulfill Responses
// async call with multiple responses returns something that satisfies Responses. likely with an internal channel to pass back response

type Responses[T any] interface {
	Next(context.Context) (T, error, bool) // bool = done
}

type AsyncRoundTripper[T any] interface {
	RoundTrip(*http.Request) (Responses[T], error)
}

type AsyncRoundTripperFunc[T any] func(*http.Request) (Responses[T], error)

func (fn AsyncRoundTripperFunc[T]) RoundTrip(req *http.Request) (Responses[T], error) {
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

// what does the transition point between pipeline handlers that care about actual types and items that only care about http look like?

// helpers:
//  - sharder
//  - single response wrapper for Responses
//  - async response wrapper for Responses (channel)
//  - pipeline builder
//  - middleware merger
//    - pipeline to http.Handler

// error handling:
//  - treat everything coming back as http and combine needs standard http status code combining
//  - independently return error
//  - add simple http -> grpc status code mapping
