package frontend

import "net/http"

// middleware.go contains types and code related to building http pipelines

// RoundTripperFunc is like http.HandlerFunc, but for RoundTripper
// chosen for pipeline building over queryrange.Handler b/c of how similar queryrange.Handler is to this existing interface.
type RoundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip implememnts http.RoundTripper
func (fn RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

// Middleware is used to build pipelines of http.Roundtrippers
type Middleware interface {
	Wrap(http.RoundTripper) http.RoundTripper
}

// MiddlewareFunc is like http.HandlerFunc, but for Middleware.
type MiddlewareFunc func(http.RoundTripper) http.RoundTripper

// Wrap implements Middleware.
func (q MiddlewareFunc) Wrap(h http.RoundTripper) http.RoundTripper {
	return q(h)
}

// MergeMiddlewares takes a set of ordered middlewares and merges them into a pipeline
func MergeMiddlewares(middleware ...Middleware) Middleware {
	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		for i := len(middleware) - 1; i >= 0; i-- {
			next = middleware[i].Wrap(next)
		}
		return next
	})
}

type roundTripper struct {
	handler http.RoundTripper
}

// NewRoundTripper takes an ordered set of middlewares and builds a http.RoundTripper
// around them
func NewRoundTripper(next http.RoundTripper, middlewares ...Middleware) http.RoundTripper {
	return roundTripper{
		handler: MergeMiddlewares(middlewares...).Wrap(next),
	}
}

func (q roundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return q.handler.RoundTrip(r)
}
