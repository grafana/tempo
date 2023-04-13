package frontend

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/golang/protobuf/jsonpb" //nolint:all //deprecated
	"github.com/golang/protobuf/proto"  //nolint:all //deprecated
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/weaveworks/common/user"

	"github.com/grafana/tempo/modules/overrides"
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
	TraceByIDHandler, SearchHandler http.Handler
	logger                          log.Logger
	store                           storage.Store
}

// New returns a new QueryFrontend
func New(cfg Config, next http.RoundTripper, o *overrides.Overrides, store storage.Store, logger log.Logger, registerer prometheus.Registerer) (*QueryFrontend, error) {
	level.Info(logger).Log("msg", "creating middleware in query frontend")

	if cfg.TraceByID.QueryShards < minQueryShards || cfg.TraceByID.QueryShards > maxQueryShards {
		return nil, fmt.Errorf("frontend query shards should be between %d and %d (both inclusive)", minQueryShards, maxQueryShards)
	}

	if cfg.Search.Sharder.ConcurrentRequests <= 0 {
		return nil, fmt.Errorf("frontend search concurrent requests should be greater than 0")
	}

	if cfg.Search.Sharder.TargetBytesPerRequest <= 0 {
		return nil, fmt.Errorf("frontend search target bytes per request should be greater than 0")
	}

	if cfg.Search.Sharder.QueryIngestersUntil < cfg.Search.Sharder.QueryBackendAfter {
		return nil, fmt.Errorf("query backend after should be less than or equal to query ingester until")
	}

	queriesPerTenant := promauto.With(registerer).NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "query_frontend_queries_total",
		Help:      "Total queries received per tenant.",
	}, []string{"tenant", "op", "status"})

	retryWare := newRetryWare(cfg.MaxRetries, registerer)

	// tracebyid middleware
	traceByIDMiddleware := MergeMiddlewares(newTraceByIDMiddleware(cfg, logger), retryWare)
	searchMiddleware := MergeMiddlewares(newSearchMiddleware(cfg, o, store, logger), retryWare)

	traceByIDCounter := queriesPerTenant.MustCurryWith(prometheus.Labels{"op": traceByIDOp})
	searchCounter := queriesPerTenant.MustCurryWith(prometheus.Labels{"op": searchOp})

	traces := traceByIDMiddleware.Wrap(next)
	search := searchMiddleware.Wrap(next)
	return &QueryFrontend{
		TraceByIDHandler: newHandler(traces, traceByIDCounter, logger),
		SearchHandler:    newHandler(search, searchCounter, logger),
		logger:           logger,
		store:            store,
	}, nil
}

func (q *QueryFrontend) Search(req *tempopb.SearchRequest, srv tempopb.StreamingQuerier_SearchServer) error {
	return nil
}

// newTraceByIDMiddleware creates a new frontend middleware responsible for handling get traces requests.
func newTraceByIDMiddleware(cfg Config, logger log.Logger) Middleware {
	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		// We're constructing middleware in this statement, each middleware wraps the next one from left-to-right
		// - the Deduper dedupes Span IDs for Zipkin support
		// - the ShardingWare shards queries by splitting the block ID space
		// - the RetryWare retries requests that have failed (error or http status 500)
		rt := NewRoundTripper(
			next,
			newDeduper(logger),
			newTraceByIDSharder(cfg.TraceByID.QueryShards, cfg.TolerateFailedBlocks, cfg.TraceByID.SLO, logger),
			newHedgedRequestWare(cfg.TraceByID.Hedging),
		)

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

			// validate start and end parameter
			_, _, _, _, _, reqErr := api.ValidateAndSanitizeRequest(r)
			if reqErr != nil {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(reqErr.Error())),
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

				if resp.Header != nil {
					resp.Header.Set(api.HeaderContentType, marshallingFormat)
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
func newSearchMiddleware(cfg Config, o *overrides.Overrides, reader tempodb.Reader, logger log.Logger) Middleware {
	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		ingesterSearchRT := next
		backendSearchRT := NewRoundTripper(next, newSearchSharder(reader, o, cfg.Search.Sharder, cfg.Search.SLO, logger))

		return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			// backend search queries require sharding so we pass through a special roundtripper
			if api.IsBackendSearch(r) {
				return backendSearchRT.RoundTrip(r)
			}

			// ingester search queries only need to be proxied to a single querier
			orgID, _ := user.ExtractOrgID(r.Context())

			r.Header.Set(user.OrgIDHeaderName, orgID)
			r.RequestURI = buildUpstreamRequestURI(r.RequestURI, nil)

			return ingesterSearchRT.RoundTrip(r)
		})
	})
}

// buildUpstreamRequestURI returns a uri based on the passed parameters
// we do this because weaveworks/common uses the RequestURI field to translate from http.Request to httpgrpc.Request
// https://github.com/weaveworks/common/blob/47e357f4e1badb7da17ad74bae63e228bdd76e8f/httpgrpc/server/server.go#L48
func buildUpstreamRequestURI(originalURI string, params url.Values) string {
	const queryDelimiter = "?"

	uri := path.Join(api.PathPrefixQuerier, originalURI)
	if len(params) > 0 {
		uri += queryDelimiter + params.Encode()
	}

	return uri
}
