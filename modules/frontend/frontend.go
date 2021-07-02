package frontend

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/cortexproject/cortex/pkg/querier/queryrange"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/opentracing/opentracing-go"
	ot_log "github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/weaveworks/common/middleware"
	"github.com/weaveworks/common/user"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
)

// NewTripperware returns a Tripperware configured with a middleware to route, split and dedupe requests.
func NewTripperware(cfg Config, logger log.Logger, registerer prometheus.Registerer) (queryrange.Tripperware, error) {
	level.Info(logger).Log("msg", "creating tripperware in query frontend")

	tracesTripperware := NewTracesTripperware(cfg, logger)
	searchTripperware := NewSearchTripperware(logger)

	return func(next http.RoundTripper) http.RoundTripper {
		traces := tracesTripperware(next)
		search := searchTripperware(next)

		return newFrontendRoundTripper(next, traces, search, logger, registerer)
	}, nil
}

type frontendRoundTripper struct {
	next, traces, search http.RoundTripper
	logger               log.Logger
	queriesPerTenant     *prometheus.CounterVec
}

func newFrontendRoundTripper(next, traces, search http.RoundTripper, logger log.Logger, registerer prometheus.Registerer) frontendRoundTripper {
	queriesPerTenant := promauto.With(registerer).NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "query_frontend_queries_total",
		Help:      "Total queries received per tenant.",
	}, []string{"tenant"})

	return frontendRoundTripper{
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
	span, ctx := opentracing.StartSpanFromContext(req.Context(), "frontend.Tripperware")
	defer span.Finish()

	orgID, _ := user.ExtractOrgID(req.Context())
	r.queriesPerTenant.WithLabelValues(orgID).Inc()
	span.SetTag("orgID", orgID)

	// for context propagation with traceID set
	req = req.WithContext(ctx)

	// route the request to the appropriate RoundTripper
	switch op := getOperation(req.URL.Path); op {
	case TracesOp:
		resp, err = r.traces.RoundTrip(req)
	case SearchOp:
		resp, err = r.search.RoundTrip(req)
	default:
		resp, err = r.next.RoundTrip(req)
	}

	traceID, _ := middleware.ExtractTraceID(ctx)
	statusCode := 500
	var contentLength int64 = 0
	if resp != nil {
		statusCode = resp.StatusCode
		contentLength = resp.ContentLength
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
	SearchOp           = "search"
)

func getOperation(path string) RequestOp {
	switch {
	case strings.HasPrefix(path, "/api/traces"):
		return TracesOp
	case strings.HasPrefix(path, "/api/search"):
		return SearchOp
	default:
		return ""
	}
}

// NewTracesTripperware creates a new frontend tripperware responsible for handling get traces requests.
func NewTracesTripperware(cfg Config, logger log.Logger) func(next http.RoundTripper) http.RoundTripper {
	return func(next http.RoundTripper) http.RoundTripper {
		// We're constructing middleware in this statement. There are two at the moment -
		// - the rightmost one (executed first) is ShardingWare which helps to shard queries by splitting the block ID space
		// - the leftmost one (executed last) is Deduper which dedupe Span IDs for Zipkin support
		rt := NewRoundTripper(next, Deduper(logger), ShardingWare(cfg.QueryShards, logger))

		return queryrange.RoundTripFunc(func(r *http.Request) (*http.Response, error) {
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
	}
}

// NewSearchTripperware creates a new frontend tripperware to handle search and search tags requests.
func NewSearchTripperware(logger log.Logger) queryrange.Tripperware {
	return func(rt http.RoundTripper) http.RoundTripper {
		return queryrange.RoundTripFunc(func(r *http.Request) (*http.Response, error) {
			// don't start a new span, this is already handled by frontendRoundTripper
			span := opentracing.SpanFromContext(r.Context())

			// check marshalling format
			marshallingFormat := util.JSONTypeHeaderValue
			if r.Header.Get(util.AcceptHeaderKey) == util.ProtobufTypeHeaderValue {
				marshallingFormat = util.ProtobufTypeHeaderValue
			}

			// TODO GRPC requests aren't handled properply yet

			//// Enforce all communication internal to Tempo to be in protobuf bytes
			//r.Header.Set(util.AcceptHeaderKey, util.ProtobufTypeHeaderValue)

			orgID, _ := user.ExtractOrgID(r.Context())

			r.Header.Set(user.OrgIDHeaderName, orgID)
			r.RequestURI = querierPrefix + r.RequestURI

			resp, err := rt.RoundTrip(r)

			//if resp != nil && resp.StatusCode == http.StatusOK && marshallingFormat == util.JSONTypeHeaderValue {
			//	// if request is for application/json, unmarshal into proto object and re-marshal into json bytes
			//	body, err := io.ReadAll(resp.Body)
			//	resp.Body.Close()
			//	if err != nil {
			//		return nil, errors.Wrap(err, "error reading response body at query frontend")
			//	}
			//	traceObject := &tempopb.Trace{}
			//	err = proto.Unmarshal(body, traceObject)
			//	if err != nil {
			//		return nil, err
			//	}
			//
			//	var jsonTrace bytes.Buffer
			//	marshaller := &jsonpb.Marshaler{}
			//	err = marshaller.Marshal(&jsonTrace, traceObject)
			//	if err != nil {
			//		return nil, err
			//	}
			//	resp.Body = ioutil.NopCloser(bytes.NewReader(jsonTrace.Bytes()))
			//}
			span.SetTag("response marshalling format", marshallingFormat)

			return resp, err
		})
	}
}
