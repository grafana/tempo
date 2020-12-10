package frontend

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/cortexproject/cortex/pkg/querier/frontend"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/grafana/tempo/modules/querier"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/weaveworks/common/user"
	"net/http"
)

// NewTripperware returns a Tripperware configured with a middleware to split requests
func NewTripperware(cfg FrontendConfig, logger log.Logger, registerer prometheus.Registerer) (frontend.Tripperware, error) {
	level.Info(logger).Log("msg", "creating tripperware in query frontend to shard queries")
	queriesPerTenant := promauto.With(registerer).NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "query_frontend_queries_total",
		Help:      "Total queries received per tenant.",
	}, []string{"tenant"})

	return func(next http.RoundTripper) http.RoundTripper {
		// get the http request, add some custom parameters to it, split it, and call downstream roundtripper
		rt := NewRoundTripper(next, ShardingWare(cfg.QueryShards, logger, registerer))
		return frontend.RoundTripFunc(func(r *http.Request) (*http.Response, error) {
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
	return q.handler.Do(r)
}

// Do implements Handler.
func (q roundTripper) Do(r *http.Request) (*http.Response, error) {
	return q.next.RoundTrip(r)
}

func ShardingWare(queryShards int, logger log.Logger, registerer prometheus.Registerer) Middleware {
	return MiddlewareFunc(func(next Handler) Handler {
		return shardQuery{
			next:        next,
			queryShards: queryShards,
			logger:      logger,
			splitByCounter: promauto.With(registerer).NewCounterVec(prometheus.CounterOpts{
				Namespace: "tempo",
				Name:      "query_frontend_split_queries_total",
				Help:      "Total number of underlying query requests after sharding.",
			}, []string{"tenant"}),
		}
	})
}

type shardQuery struct {
	next            Handler
	queryShards     int
	logger          log.Logger
	blockBoundaries [][]byte

	// Metrics.
	splitByCounter *prometheus.CounterVec
}

// Do implements Handler
func (s shardQuery) Do(r *http.Request) (*http.Response, error) {
	userID, err := user.ExtractOrgID(r.Context())
	if err != nil {
		return nil, err
	}

	// only need to initialise boundaries once
	if len(s.blockBoundaries) == 0 {
		s.blockBoundaries = createBlockShards(s.queryShards)
	}

	reqs := make([]*http.Request, s.queryShards)
	for i := 0; i < s.queryShards; i++ {
		reqs[i] = r.Clone(r.Context())
		q := reqs[i].URL.Query()
		q.Add(querier.BlockStartKey, hex.EncodeToString(s.blockBoundaries[i]))
		q.Add(querier.BlockEndKey, hex.EncodeToString(s.blockBoundaries[i+1]))

		if i==0 {
			q.Add(querier.QueryIngestersKey, "true")
		} else {
			q.Add(querier.QueryIngestersKey, "false")
		}

		// adding to RequestURI ONLY because weaveworks/common uses the RequestURI field to translate from
		reqs[i].RequestURI = reqs[i].URL.RequestURI() + "?" + q.Encode()
	}
	s.splitByCounter.WithLabelValues(userID).Add(float64(s.queryShards))

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

func createBlockShards(queryShards int) [][]byte {
	if queryShards == 0 {
		return nil
	}

	// create sharded queries
	blockBoundaries := make([][]byte, queryShards+1)
	for i := 0; i < queryShards+1; i ++ {
		blockBoundaries[i] = make([]byte, 16)
	}
	const MaxUint = uint64(^uint8(0))
	for i := 0; i < queryShards; i++ {
		binary.LittleEndian.PutUint64(blockBoundaries[i][:8], (MaxUint/uint64(queryShards))*uint64(i))
		binary.LittleEndian.PutUint64(blockBoundaries[i][8:], 0)
	}
	const MaxUint64 = ^uint64(0)
	binary.LittleEndian.PutUint64(blockBoundaries[queryShards][:8], MaxUint64)
	binary.LittleEndian.PutUint64(blockBoundaries[queryShards][8:], MaxUint64)

	return blockBoundaries
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
