package pipeline

import (
	"context"
	"net/http"
)

// sync call with one response just returns a Response interface above. maybe a small wrapper to fulfill Responses
// async call with multiple responses returns something that satisfies Responses. likely with an internal channel to pass back response

type Responses interface {
	Next(context.Context) (*http.Response, error, bool) // bool = done
}

type AsyncRoundTripper interface {
	RoundTrip(*http.Request) (Responses, error)
}

type AsyncRoundTripperFunc func(*http.Request) (Responses, error)

func (fn AsyncRoundTripperFunc) RoundTrip(req *http.Request) (Responses, error) {
	return fn(req)
}

// AsyncMiddleware is used to build pipelines of pipeline.Roundtrippers
type AsyncMiddleware interface {
	Wrap(AsyncRoundTripper) AsyncRoundTripper
}

// AsyncMiddlewareFunc is like http.HandlerFunc, but for Middleware.
type AsyncMiddlewareFunc func(AsyncRoundTripper) AsyncRoundTripper

// Wrap implements Middleware.
func (f AsyncMiddlewareFunc) Wrap(w AsyncRoundTripper) AsyncRoundTripper {
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
