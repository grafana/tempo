package frontend

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/weaveworks/common/user"

	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb"
)

const (
	traceByIDOp = "traces"
	searchOp    = "search"
)

type QueryFrontend struct {
	TraceByID, Search, BackendSearch http.Handler
	logger                           log.Logger
	queriesPerTenant                 *prometheus.CounterVec
	store                            storage.Store
}

// New returns a new QueryFrontend
func New(cfg Config, next http.RoundTripper, store storage.Store, logger log.Logger, registerer prometheus.Registerer) (*QueryFrontend, error) {
	level.Info(logger).Log("msg", "creating middleware in query frontend")

	if cfg.QueryShards < minQueryShards || cfg.QueryShards > maxQueryShards {
		return nil, fmt.Errorf("frontend query shards should be between %d and %d (both inclusive)", minQueryShards, maxQueryShards)
	}

	queriesPerTenant := promauto.With(registerer).NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "query_frontend_queries_total",
		Help:      "Total queries received per tenant.",
	}, []string{"tenant", "op"})

	retryWare := newRetryWare(cfg.MaxRetries, registerer)

	traceByIDMiddleware := MergeMiddlewares(newTraceByIDMiddleware(cfg, logger), retryWare)
	searchMiddleware := MergeMiddlewares(newSearchMiddleware(), retryWare)
	backendMiddleware := MergeMiddlewares(newBackendSearchMiddleware(store, logger), retryWare)

	traceByIDCounter := queriesPerTenant.MustCurryWith(prometheus.Labels{
		"op": traceByIDOp,
	})
	searchCounter := queriesPerTenant.MustCurryWith(prometheus.Labels{
		"op": searchOp,
	})

	traces := traceByIDMiddleware.Wrap(next)
	search := searchMiddleware.Wrap(next)
	backend := backendMiddleware.Wrap(next)
	return &QueryFrontend{
		TraceByID:        newHandler(traces, traceByIDCounter, logger),
		Search:           newHandler(search, searchCounter, logger),
		BackendSearch:    newHandler(backend, searchCounter, logger),
		logger:           logger,
		queriesPerTenant: queriesPerTenant,
		store:            store,
	}, nil
}

// newTraceByIDMiddleware creates a new frontend middleware responsible for handling get traces requests.
func newTraceByIDMiddleware(cfg Config, logger log.Logger) Middleware {
	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		// We're constructing middleware in this statement, each middleware wraps the next one from left-to-right
		// - the Deduper dedupes Span IDs for Zipkin support
		// - the ShardingWare shards queries by splitting the block ID space
		// - the RetryWare retries requests that have failed (error or http status 500)
		rt := NewRoundTripper(next, newDeduper(logger), newTraceByIDSharder(cfg.QueryShards, cfg.TolerateFailedBlocks, logger))

		return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			// validate traceID
			_, err := api.ParseTraceID(r)
			if err != nil {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(err.Error())),
					Header:     http.Header{},
				}, nil
			}

			// check marshalling format
			marshallingFormat := api.HeaderAcceptJSON
			if r.Header.Get(api.HeaderAccept) == api.HeaderAcceptProtobuf {
				marshallingFormat = api.HeaderAcceptProtobuf
			}

			// enforce all communication internal to Tempo to be in protobuf bytes
			r.Header.Set(api.HeaderAccept, api.HeaderAcceptProtobuf)

			resp, err := rt.RoundTrip(r)

			// todo : should all of this request/response content type be up a level and be used for all query types?
			if resp != nil && resp.StatusCode == http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					return nil, errors.Wrap(err, "error reading response body at query frontend")
				}
				responseObject := &tempopb.TraceByIDResponse{}
				err = proto.Unmarshal(body, responseObject)
				if err != nil {
					return nil, err
				}

				if responseObject.Metrics.FailedBlocks > 0 {
					resp.StatusCode = http.StatusPartialContent
				}

				if marshallingFormat == api.HeaderAcceptJSON {
					var jsonTrace bytes.Buffer
					marshaller := &jsonpb.Marshaler{}
					err = marshaller.Marshal(&jsonTrace, responseObject.Trace)
					if err != nil {
						return nil, err
					}
					resp.Body = io.NopCloser(bytes.NewReader(jsonTrace.Bytes()))
				} else {
					traceBuffer, err := proto.Marshal(responseObject.Trace)
					if err != nil {
						return nil, err
					}
					resp.Body = io.NopCloser(bytes.NewReader(traceBuffer))
				}
			}
			span := opentracing.SpanFromContext(r.Context())
			if span != nil {
				span.SetTag("contentType", marshallingFormat)
			}

			return resp, err
		})
	})
}

// newSearchMiddleware creates a new frontend middleware to handle search and search tags requests.
func newSearchMiddleware() Middleware {
	return MiddlewareFunc(func(rt http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			orgID, _ := user.ExtractOrgID(r.Context())

			r.Header.Set(user.OrgIDHeaderName, orgID)
			r.RequestURI = api.PathPrefixQuerier + r.RequestURI

			resp, err := rt.RoundTrip(r)

			return resp, err
		})
	})
}

// newBackendSearchMiddleware creates a new frontend middleware to handle backend search.
// todo(search): integrate with real search
func newBackendSearchMiddleware(reader tempodb.Reader, logger log.Logger) Middleware {
	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		rt := NewRoundTripper(next, newSearchSharder(reader, defaultConcurrentRequests, logger))

		return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			orgID, _ := user.ExtractOrgID(r.Context())

			r.Header.Set(user.OrgIDHeaderName, orgID)

			resp, err := rt.RoundTrip(r)

			return resp, err
		})
	})
}
