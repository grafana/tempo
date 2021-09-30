package frontend

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/opentracing/opentracing-go"
	ot_log "github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/weaveworks/common/httpgrpc"
	"github.com/weaveworks/common/tracing"
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
	}, []string{"tenant"})

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
	start := time.Now()
	// tracing instrumentation
	span, ctx := opentracing.StartSpanFromContext(req.Context(), "frontend.Middleware")
	defer span.Finish()

	orgID, _ := user.ExtractOrgID(req.Context())
	r.queriesPerTenant.WithLabelValues(orgID).Inc()
	span.SetTag("orgID", orgID)

	// for context propagation with traceID set
	req = req.WithContext(ctx)

	// route the request to the appropriate RoundTripper
	switch op := getOperation(r.apiPrefix, req.URL.Path); op {
	case TracesOp:
		resp, err = r.traces.RoundTrip(req)
	case SearchOp:
		resp, err = r.search.RoundTrip(req)
	default:
		// should never be called
		level.Warn(r.logger).Log("msg", "unknown path called in frontend roundtripper", "path", req.URL.Path)
		resp, err = r.next.RoundTrip(req)
	}

	// jpe move to handler
	traceID, _ := tracing.ExtractTraceID(ctx)
	statusCode := 500
	var contentLength int64 = 0
	if resp != nil {
		statusCode = resp.StatusCode
		contentLength = resp.ContentLength
	} else if httpResp, ok := httpgrpc.HTTPResponseFromError(err); ok {
		statusCode = int(httpResp.Code)
		contentLength = int64(len(httpResp.Body))
	}

	level.Info(r.logger).Log(
		"tenant", orgID,
		"method", req.Method,
		"traceID", traceID,
		"url", req.URL.RequestURI(),
		"duration", time.Since(start).String(),
		"response_size", contentLength,
		"status", statusCode,
	)

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
		rt := NewRoundTripper(next, Deduper(logger), ShardingWare(cfg.QueryShards, logger), RetryWare(cfg.MaxRetries, registerer))

		return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			// don't start a new span, this is already handled by frontendRoundTripper
			span := opentracing.SpanFromContext(r.Context())

			// validate traceID
			_, err := util.ParseTraceID(r)
			if err != nil {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       ioutil.NopCloser(strings.NewReader(err.Error())),
					Header:     http.Header{},
				}, nil
			}
			span.LogFields(ot_log.String("msg", "validated traceID"))

			// check marshalling format
			marshallingFormat := util.JSONTypeHeaderValue
			if r.Header.Get(util.AcceptHeaderKey) == util.ProtobufTypeHeaderValue {
				marshallingFormat = util.ProtobufTypeHeaderValue
			}

			// Enforce all communication internal to Tempo to be in protobuf bytes
			r.Header.Set(util.AcceptHeaderKey, util.ProtobufTypeHeaderValue)

			resp, err := rt.RoundTrip(r)

			if resp != nil && resp.StatusCode == http.StatusOK && marshallingFormat == util.JSONTypeHeaderValue {
				// if request is for application/json, unmarshal into proto object and re-marshal into json bytes
				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					return nil, errors.Wrap(err, "error reading response body at query frontend")
				}
				traceObject := &tempopb.Trace{}
				err = proto.Unmarshal(body, traceObject)
				if err != nil {
					return nil, err
				}

				var jsonTrace bytes.Buffer
				marshaller := &jsonpb.Marshaler{}
				err = marshaller.Marshal(&jsonTrace, traceObject)
				if err != nil {
					return nil, err
				}
				resp.Body = ioutil.NopCloser(bytes.NewReader(jsonTrace.Bytes()))
			}
			span.SetTag("response marshalling format", marshallingFormat)

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
