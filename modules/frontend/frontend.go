package frontend

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/cortexproject/cortex/pkg/querier/frontend"
	"github.com/cortexproject/cortex/pkg/util"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/weaveworks/common/user"
	"net/http"
)

type FrontendConfig struct {
	frontend.Config `yaml:",inline"`
	ShardNum int    `yaml:"shard_num,omitempty"`
}

func (cfg *FrontendConfig) ApplyDefaults() {
	cfg.Config.CompressResponses = false
	cfg.Config.DownstreamURL = ""
	cfg.Config.LogQueriesLongerThan = 0
	cfg.Config.MaxOutstandingPerTenant = 100
	cfg.ShardNum = 4
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
			level.Info(util.Logger).Log("msg", "request received by custom tripperware")
			orgID := r.Header.Get(user.OrgIDHeaderName)
			queriesPerTenant.WithLabelValues(orgID).Inc()

			r = r.WithContext(user.InjectOrgID(r.Context(), orgID))
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
	level.Info(util.Logger).Log("blockStart", r.URL.Query().Get("blockStart"), "blockEnd", r.URL.Query().Get("blockEnd"))
	return q.handler.Do(r)
}

// Do implements Handler.
func (q roundTripper) Do(r *http.Request) (*http.Response, error) {
	level.Info(util.Logger).Log("msg", "roundTripper.Do called")
	return q.next.RoundTrip(r)
}

func ShardingWare(shardNum int, logger log.Logger, registerer prometheus.Registerer) Middleware {
	return MiddlewareFunc(func(next Handler) Handler {
		return shardQuery{
			next:     next,
			shardNum: shardNum,
			logger:   logger,
			splitByCounter: promauto.With(registerer).NewCounterVec(prometheus.CounterOpts{
				Namespace: "tempo",
				Name:      "query_frontend_split_queries_total",
				Help:      "Total number of underlying query requests after sharding",
			}, []string{"user"}),
		}
	})
}

type shardQuery struct {
	next     Handler
	shardNum int
	logger   log.Logger
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
	boundaryBytes := make([][]byte, s.shardNum+1)
	for i := 0; i < s.shardNum+1; i ++ {
		boundaryBytes[i] = make([]byte, 0)
	}
	const MaxUint = ^uint64(0)
	const MaxInt = int64(MaxUint >> 1)
	for i := 0; i < s.shardNum; i++ {
		binary.PutVarint(boundaryBytes[i], MaxInt*(int64(i))/int64(s.shardNum))
		binary.PutVarint(boundaryBytes[i], 0)
	}
	binary.PutVarint(boundaryBytes[s.shardNum], MaxInt)
	binary.PutVarint(boundaryBytes[s.shardNum], MaxInt)

	reqs := make([]*http.Request, s.shardNum)
	for i := 0; i < s.shardNum; i++ {
		reqs[i] = r
		reqs[i].URL.Query().Add("blockStart", string(boundaryBytes[i]))
		reqs[i].URL.Query().Add("blockEnd", string(boundaryBytes[i+1]))
	}
	s.splitByCounter.WithLabelValues(userID).Add(float64(s.shardNum))

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
	// todo: make this configurable using limits
	parallelism := 10
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
