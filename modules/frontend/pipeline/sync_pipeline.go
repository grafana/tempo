package pipeline

import "net/http"

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
