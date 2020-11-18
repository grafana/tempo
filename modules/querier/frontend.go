package querier

import (
	"context"
	"fmt"
	"github.com/cortexproject/cortex/pkg/querier/frontend"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/weaveworks/common/user"
	"net/http"
)

type FrontendConfig struct {
	ShardNum int `yaml:"shard_num,omitempty"`
}

// NewTripperware returns a Tripperware configured with a middleware to split requests
func NewTripperware(cfg FrontendConfig, logger log.Logger, registerer prometheus.Registerer) (frontend.Tripperware, error) {
	level.Info(logger).Log("msg", "creating tripperware in query frontend to shard queries")
	queriesPerTenant := promauto.With(registerer).NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "query_frontend_queries_total",
		Help:      "Total queries sent per tenant.",
	}, []string{"user"})

	return func(next http.RoundTripper) http.RoundTripper {
		// get the http request, add some custom parameters to it, split it, and call downstream roundtripper
		rt := NewRoundTripper(next, ShardingWare(cfg.ShardNum, logger, registerer))
		return frontend.RoundTripFunc(func(r *http.Request) (*http.Response, error) {
			userID, err := user.ExtractOrgID(r.Context())
			if err != nil {
				return nil, err
			}
			queriesPerTenant.WithLabelValues(userID).Inc()
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
		next:  next,
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

func ShardingWare(shardNum int, logger log.Logger, registerer prometheus.Registerer) Middleware {
	return MiddlewareFunc(func(next Handler) Handler {
		return shardQuery{
			next:     next,
			shardNum: shardNum,
			splitByCounter: promauto.With(registerer).NewCounterVec(prometheus.CounterOpts{
				Namespace: "tempo",
				Name:      "frontend_split_queries_total",
				Help:      "Total number of underlying query requests after the split by interval is applied",
			}, []string{"user"}),
		}
	})
}

type shardQuery struct {
	next     Handler
	shardNum int
	logger log.Logger
	// Metrics.
	splitByCounter *prometheus.CounterVec
}

// Do implements Handler
func (s shardQuery) Do(r *http.Request) (*http.Response, error) {
	level.Info(s.logger).Log("msg", "shardQuery called")

	userID, err := user.ExtractOrgID(r.Context())
	if err != nil {
		return nil, err
	}

	// create sharded queries
	// fixme: using a constant 4 for now, change to "shardNum" partitions
	uuidBoundary0 := string(0)
	uuidBoundary1 := string(1 << 14)
	uuidBoundary2 := string(1 << 15)
	uuidBoundary3 := string((1 << 15) + (1 << 14))
	uuidBoundary4 := string((1 << 16) - 1)

	reqs := make([]*http.Request, 4)
	reqs[0] = r
	reqs[0].URL.Query().Add("blockStart", uuidBoundary0)
	reqs[0].URL.Query().Add("blockEnd", uuidBoundary1)
	reqs[1] = r
	reqs[1].URL.Query().Add("blockStart", uuidBoundary1)
	reqs[1].URL.Query().Add("blockEnd", uuidBoundary2)
	reqs[2] = r
	reqs[2].URL.Query().Add("blockStart", uuidBoundary2)
	reqs[2].URL.Query().Add("blockEnd", uuidBoundary3)
	reqs[3] = r
	reqs[3].URL.Query().Add("blockStart", uuidBoundary3)
	reqs[3].URL.Query().Add("blockEnd", uuidBoundary4)

	// fixme: change to "shardNum"
	s.splitByCounter.WithLabelValues(userID).Add(4)

	rrs, err := DoRequests(r.Context(), s.next, reqs)
	if err != nil {
		return nil, err
	}

	// todo: add merging logic here if there are more than one results
	for _, rr := range rrs {
		if rr.Response.StatusCode == http.StatusOK {
			return rr.Response, nil
		}
	}

	return nil, fmt.Errorf("trace not found")
}

// RequestResponse contains a request response and the respective request that was used.
type RequestResponse struct {
	Request  *http.Request
	Response *http.Response
}

// DoRequests executes a list of requests in parallel. The limits parameters is used to limit parallelism per single request.
func DoRequests(ctx context.Context, downstream Handler, reqs []*http.Request) ([]RequestResponse, error) {
	// If one of the requests fail, we want to be able to cancel the rest of them.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Feed all requests to a bounded intermediate channel to limit parallelism.
	intermediate := make(chan *http.Request)
	go func() {
		for _, req := range reqs {
			intermediate <- req
		}
		close(intermediate)
	}()

	respChan, errChan := make(chan RequestResponse), make(chan error)
	parallelism := 10 // todo: make this configurable using limits
	if parallelism > len(reqs) {
		parallelism = len(reqs)
	}
	for i := 0; i < parallelism; i++ {
		go func() {
			for req := range intermediate {
				resp, err := downstream.Do(req)
				if err != nil {
					errChan <- err
				} else {
					respChan <- RequestResponse{req, resp}
				}
			}
		}()
	}

	resps := make([]RequestResponse, 0, len(reqs))
	var firstErr error
	for range reqs {
		select {
		case resp := <-respChan:
			resps = append(resps, resp)
		case err := <-errChan:
			if firstErr == nil {
				cancel()
				firstErr = err
			}
		}
	}

	return resps, firstErr
}
