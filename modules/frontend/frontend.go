package frontend

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/weaveworks/common/user"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
)

const (
	apiPathTraces = "/api/traces"
	apiPathSearch = "/api/search"
)

// NewMiddleware returns a Middleware configured with a middleware to route, split and dedupe requests.
func NewMiddleware(cfg Config, apiPrefix string, logger log.Logger, registerer prometheus.Registerer) (Middleware, error) {
	level.Info(logger).Log("msg", "creating middleware in query frontend")

	// todo: push this farther down? into the actual creation of the shardingware?
	if cfg.QueryShards < minQueryShards || cfg.QueryShards > maxQueryShards {
		return nil, fmt.Errorf("frontend query shards should be between %d and %d (both inclusive)", minQueryShards, maxQueryShards)
	}

	tracesMiddleware := NewTracesMiddleware(cfg, logger, registerer)
	searchMiddleware := NewSearchMiddleware()

	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		traces := tracesMiddleware.Wrap(next)
		search := searchMiddleware.Wrap(next)

		return newFrontendRoundTripper(apiPrefix, next, traces, search, logger, registerer)
	}), nil
}

type frontendRoundTripper struct {
	apiPrefix            string
	next, traces, search http.RoundTripper
	logger               log.Logger
	queriesPerTenant     *prometheus.CounterVec
}

func newFrontendRoundTripper(apiPrefix string, next, traces, search http.RoundTripper, logger log.Logger, registerer prometheus.Registerer) frontendRoundTripper {
	queriesPerTenant := promauto.With(registerer).NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "query_frontend_queries_total",
		Help:      "Total queries received per tenant.",
	}, []string{"tenant", "op"})

	return frontendRoundTripper{
		apiPrefix:        apiPrefix,
		next:             next,
		traces:           traces,
		search:           search,
		logger:           logger,
		queriesPerTenant: queriesPerTenant,
	}
}

func (r frontendRoundTripper) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	op := getOperation(r.apiPrefix, req.URL.Path)
	orgID, _ := user.ExtractOrgID(req.Context())

	r.queriesPerTenant.WithLabelValues(orgID, string(op)).Inc()

	// route the request to the appropriate RoundTripper
	//  todo: use the mux.Router in modules instead of doing custom routing here?
	switch op {
	case TracesOp:
		resp, err = r.traces.RoundTrip(req)
	case SearchOp:
		resp, err = r.search.RoundTrip(req)
	default:
		// should never be called
		level.Warn(r.logger).Log("msg", "unknown path called in frontend roundtripper", "path", req.URL.Path)
		resp, err = r.next.RoundTrip(req)
	}

	return
}

type RequestOp string

const (
	TracesOp RequestOp = "traces"
	SearchOp RequestOp = "search"
)

func getOperation(prefix, path string) RequestOp {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}

	// remove prefix from path
	path = path[len(prefix):]

	switch {
	case strings.HasPrefix(path, apiPathTraces):
		return TracesOp
	case strings.HasPrefix(path, apiPathSearch):
		return SearchOp
	default:
		return ""
	}
}

// NewTracesMiddleware creates a new frontend middleware responsible for handling get traces requests.
func NewTracesMiddleware(cfg Config, logger log.Logger, registerer prometheus.Registerer) Middleware {
	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		// We're constructing middleware in this statement, each middleware wraps the next one from left-to-right
		// - the Deduper dedupes Span IDs for Zipkin support
		// - the ShardingWare shards queries by splitting the block ID space
		// - the RetryWare retries requests that have failed (error or http status 500)
		rt := NewRoundTripper(next, Deduper(logger), ShardingWare(cfg.QueryShards, cfg.TolerateFailedBlocks, logger), RetryWare(cfg.MaxRetries, registerer))

		return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			// validate traceID
			_, err := util.ParseTraceID(r)
			if err != nil {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(err.Error())),
					Header:     http.Header{},
				}, nil
			}

			// check marshalling format
			marshallingFormat := util.JSONTypeHeaderValue
			if r.Header.Get(util.AcceptHeaderKey) == util.ProtobufTypeHeaderValue {
				marshallingFormat = util.ProtobufTypeHeaderValue
			}

			// enforce all communication internal to Tempo to be in protobuf bytes
			r.Header.Set(util.AcceptHeaderKey, util.ProtobufTypeHeaderValue)

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

				if marshallingFormat == util.JSONTypeHeaderValue {
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

// NewSearchMiddleware creates a new frontend middleware to handle search and search tags requests.
func NewSearchMiddleware() Middleware {
	return MiddlewareFunc(func(rt http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			orgID, _ := user.ExtractOrgID(r.Context())

			r.Header.Set(user.OrgIDHeaderName, orgID)
			r.RequestURI = querierPrefix + r.RequestURI

			resp, err := rt.RoundTrip(r)

			return resp, err
		})
	})
}
