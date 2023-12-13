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
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/cache"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb"
)

type streamingSearchHandler func(req *tempopb.SearchRequest, srv tempopb.StreamingQuerier_SearchServer) error

type QueryFrontend struct {
	TraceByIDHandler, SearchHandler, SpanMetricsSummaryHandler, SearchWSHandler                http.Handler
	SearchTagsHandler, SearchTagsV2Handler, SearchTagsValuesHandler, SearchTagsValuesV2Handler http.Handler
	cacheProvider                                                                              cache.Provider
	streamingSearch                                                                            streamingSearchHandler
	logger                                                                                     log.Logger
}

// New returns a new QueryFrontend
func New(cfg Config, next http.RoundTripper, o overrides.Interface, reader tempodb.Reader, cacheProvider cache.Provider, apiPrefix string, logger log.Logger, registerer prometheus.Registerer) (*QueryFrontend, error) {
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

	retryWare := newRetryWare(cfg.MaxRetries, registerer)

	// cache
	searchCache := newFrontendCache(cacheProvider, cache.RoleFrontendSearch, logger)

	// TODO: return error for routes that don't support multi-tenant queries

	// inject multi-tenant middleware in multi-tenant routes
	traceByIDMiddleware := MergeMiddlewares(
		newMultiTenantMiddleware(cfg, combiner.NewTraceByID, logger),
		newTraceByIDMiddleware(cfg, o, logger), retryWare)

	searchMiddleware := MergeMiddlewares(
		newMultiTenantMiddleware(cfg, combiner.NewSearch, logger),
		newSearchMiddleware(cfg, o, reader, searchCache, logger), retryWare)

	searchTagsMiddleware := MergeMiddlewares(
		newMultiTenantMiddleware(cfg, combiner.NewSearchTags, logger),
		newSearchTagsMiddleware(), retryWare)

	searchTagsV2Middleware := MergeMiddlewares(
		newMultiTenantMiddleware(cfg, combiner.NewSearchTagsV2, logger),
		newSearchTagsMiddleware(), retryWare)

	searchTagsValuesMiddleware := MergeMiddlewares(
		newMultiTenantMiddleware(cfg, combiner.NewSearchTagValues, logger),
		newSearchTagsMiddleware(), retryWare)

	searchTagsValuesV2Middleware := MergeMiddlewares(
		newMultiTenantMiddleware(cfg, combiner.NewSearchTagValuesV2, logger),
		newSearchTagsMiddleware(), retryWare)

	spanMetricsMiddleware := MergeMiddlewares(
		newMultiTenantUnsupportedMiddleware(cfg, logger),
		newSpanMetricsMiddleware(), retryWare)

	traces := traceByIDMiddleware.Wrap(next)
	search := searchMiddleware.Wrap(next)
	searchTags := searchTagsMiddleware.Wrap(next)
	searchTagsV2 := searchTagsV2Middleware.Wrap(next)
	searchTagValues := searchTagsValuesMiddleware.Wrap(next)
	searchTagValuesV2 := searchTagsValuesV2Middleware.Wrap(next)

	metrics := spanMetricsMiddleware.Wrap(next)

	streamingMiddleware := MergeMiddlewares(retryWare).Wrap(next)

	return &QueryFrontend{
		TraceByIDHandler:          newHandler(traces, traceByIDSLOPostHook(cfg.TraceByID.SLO), nil, logger),
		SearchHandler:             newHandler(search, searchSLOPostHook(cfg.Search.SLO), searchSLOPreHook, logger),
		SearchTagsHandler:         newHandler(searchTags, nil, nil, logger),
		SearchTagsV2Handler:       newHandler(searchTagsV2, nil, nil, logger),
		SearchTagsValuesHandler:   newHandler(searchTagValues, nil, nil, logger),
		SearchTagsValuesV2Handler: newHandler(searchTagValuesV2, nil, nil, logger),

		SpanMetricsSummaryHandler: newHandler(metrics, nil, nil, logger),
		SearchWSHandler:           newSearchStreamingWSHandler(cfg, o, streamingMiddleware, reader, searchCache, apiPrefix, logger),
		cacheProvider:             cacheProvider,
		streamingSearch:           newSearchStreamingGRPCHandler(cfg, o, streamingMiddleware, reader, searchCache, apiPrefix, logger),
		logger:                    logger,
	}, nil
}

// Search implements StreamingQuerierServer interface for streaming search
func (q *QueryFrontend) Search(req *tempopb.SearchRequest, srv tempopb.StreamingQuerier_SearchServer) error {
	return q.streamingSearch(req, srv)
}

// newTraceByIDMiddleware creates a new frontend middleware responsible for handling get traces requests.
func newTraceByIDMiddleware(cfg Config, o overrides.Interface, logger log.Logger) Middleware {
	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		// We're constructing middleware in this statement, each middleware wraps the next one from left-to-right
		// - the Deduper dedupes Span IDs for Zipkin support
		// - the ShardingWare shards queries by splitting the block ID space
		// - the RetryWare retries requests that have failed (error or http status 500)
		rt := NewRoundTripper(
			next,
			newDeduper(logger),
			newTraceByIDSharder(&cfg.TraceByID, o, logger),
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
					return nil, fmt.Errorf("error reading response body at query frontend: %w", err)
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
func newSearchMiddleware(cfg Config, o overrides.Interface, reader tempodb.Reader, c *frontendCache, logger log.Logger) Middleware {
	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		ss := newSearchSharder(reader, o, cfg.Search.Sharder, newSearchProgress, c, logger)
		searchRT := NewRoundTripper(next, ss)

		return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			// backend search queries require sharding, so we pass through a special roundtripper
			return searchRT.RoundTrip(r)
		})
	})
}

// newSearchTagsMiddleware creates a new frontend middleware to handle search tags requests.
func newSearchTagsMiddleware() Middleware {
	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		ingesterSearchRT := next

		return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			// ingester search tags queries only need to be proxied to a single querier
			orgID, _ := user.ExtractOrgID(r.Context())

			r.Header.Set(user.OrgIDHeaderName, orgID)
			r.RequestURI = buildUpstreamRequestURI(r.RequestURI, nil)

			return ingesterSearchRT.RoundTrip(r)
		})
	})
}

// newSpanMetricsMiddleware creates a new frontend middleware to handle search and search tags requests.
func newSpanMetricsMiddleware() Middleware {
	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		generatorRT := next

		return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			// ingester search queries only need to be proxied to a single querier
			orgID, _ := user.ExtractOrgID(r.Context())

			r.Header.Set(user.OrgIDHeaderName, orgID)
			r.RequestURI = buildUpstreamRequestURI(r.RequestURI, nil)

			return generatorRT.RoundTrip(r)
		})
	})
}

// buildUpstreamRequestURI returns a uri based on the passed parameters
// we do this because dskit/common uses the RequestURI field to translate from http.Request to httpgrpc.Request
// https://github.com/grafana/dskit/blob/740f56bd293423c5147773ce97264519f9fddc58/httpgrpc/server/server.go#L59
func buildUpstreamRequestURI(originalURI string, params url.Values) string {
	const queryDelimiter = "?"

	uri := path.Join(api.PathPrefixQuerier, originalURI)
	if len(params) > 0 {
		uri += queryDelimiter + params.Encode()
	}

	return uri
}
