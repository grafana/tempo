package frontend

import (
	"net/http"

	"github.com/grafana/tempo/modules/frontend/pipeline"
)

// MergeMiddlewares takes a set of ordered middlewares and merges them into a pipeline
func MergeMiddlewares(middleware ...pipeline.Middleware) pipeline.Middleware {
	return pipeline.MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		for i := len(middleware) - 1; i >= 0; i-- {
			next = middleware[i].Wrap(next)
		}
		return next
	})
}

type roundTripper struct {
	handler http.RoundTripper
}

func (q roundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return q.handler.RoundTrip(r)
}

func NewRoundTripper(next http.RoundTripper, middlewares ...pipeline.Middleware) http.RoundTripper {
	return roundTripper{
		handler: MergeMiddlewares(middlewares...).Wrap(next),
	}
}
