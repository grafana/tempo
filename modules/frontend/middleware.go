package frontend

import "net/http"

// RoundTripperFunc is like http.HandlerFunc, but for RoundTripper
type RoundTripperFunc func(*http.Request) (*http.Response, error)

func (fn RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

// Basic pipeline building block
type Middleware interface {
	Wrap(http.RoundTripper) http.RoundTripper
}

// MiddlewareFunc is like http.HandlerFunc, but for Middleware.
type MiddlewareFunc func(http.RoundTripper) http.RoundTripper

// Wrap implements Middleware.
func (q MiddlewareFunc) Wrap(h http.RoundTripper) http.RoundTripper {
	return q(h)
}

func MergeMiddlewares(middleware ...Middleware) Middleware {
	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		for i := len(middleware) - 1; i >= 0; i-- {
			next = middleware[i].Wrap(next)
		}
		return next
	})
}

type roundTripper struct {
	next    http.RoundTripper
	handler http.RoundTripper
}

// NewRoundTripper merges a set of middlewares into an handler, then inject it into the `next` roundtripper
func NewRoundTripper(next http.RoundTripper, middlewares ...Middleware) http.RoundTripper {
	transport := roundTripper{
		next: next,
	}
	transport.handler = MergeMiddlewares(middlewares...).Wrap(&transport)
	return transport
}

func (q roundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return q.handler.RoundTrip(r)
}
