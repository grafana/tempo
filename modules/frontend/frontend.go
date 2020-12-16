package frontend

import (
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/weaveworks/common/user"

	"github.com/cortexproject/cortex/pkg/querier/frontend"
)

// NewTripperware returns a Tripperware configured with a middleware to split requests
func NewTripperware(cfg Config, logger log.Logger, registerer prometheus.Registerer) (frontend.Tripperware, error) {
	level.Info(logger).Log("msg", "creating tripperware in query frontend to shard queries")
	queriesPerTenant := promauto.With(registerer).NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "query_frontend_queries_total",
		Help:      "Total queries received per tenant.",
	}, []string{"tenant"})

	return func(next http.RoundTripper) http.RoundTripper {
		// Get the http request, add custom parameters to it, split it, and call downstream roundtripper
		rt := NewRoundTripper(next, ShardingWare(cfg.QueryShards, logger))
		return frontend.RoundTripFunc(func(r *http.Request) (*http.Response, error) {
			orgID, _ := user.ExtractOrgID(r.Context())
			queriesPerTenant.WithLabelValues(orgID).Inc()

			// tracing instrumentation
			span, ctx := opentracing.StartSpanFromContext(r.Context(), "frontend.RoundTrip")
			defer span.Finish()

			r = r.WithContext(ctx)
			return rt.RoundTrip(r)
		})
	}, nil
}

type Handler interface {
	Do(*http.Request) (*http.Response, error)
}

type Middleware interface {
	Wrap(Handler) Handler
}

// MiddlewareFunc is like http.HandlerFunc, but for Middleware.
type MiddlewareFunc func(Handler) Handler

// Wrap implements Middleware.
func (q MiddlewareFunc) Wrap(h Handler) Handler {
	return q(h)
}

func MergeMiddlewares(middleware ...Middleware) Middleware {
	return MiddlewareFunc(func(next Handler) Handler {
		for i := len(middleware) - 1; i >= 0; i-- {
			next = middleware[i].Wrap(next)
		}
		return next
	})
}

type roundTripper struct {
	next    http.RoundTripper
	handler Handler
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
	return q.handler.Do(r)
}

// Do implements Handler.
func (q roundTripper) Do(r *http.Request) (*http.Response, error) {
	return q.next.RoundTrip(r)
}
