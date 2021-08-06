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
	"github.com/weaveworks/common/httpgrpc"
	"github.com/weaveworks/common/middleware"
	"github.com/weaveworks/common/user"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
)

// NewTripperware returns a Tripperware configured with a middleware to split requests
func NewTripperware(cfg Config, logger log.Logger, registerer prometheus.Registerer) (queryrange.Tripperware, error) {
	level.Info(logger).Log("msg", "creating tripperware in query frontend to shard queries")
	queriesPerTenant := promauto.With(registerer).NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "query_frontend_queries_total",
		Help:      "Total queries received per tenant.",
	}, []string{"tenant"})

	return func(next http.RoundTripper) http.RoundTripper {
		// We're constructing middleware in this statement, each middleware wraps the next one from left-to-right
		// - the Deduper dedupes Span IDs for Zipkin support
		// - the ShardingWare shards queries by splitting the block ID space
		// - the RetryWare retries requests that have failed (error or http status 500)
		rt := NewRoundTripper(next, Deduper(logger), ShardingWare(cfg.QueryShards, logger), RetryWare(cfg.MaxRetries, registerer))

		return queryrange.RoundTripFunc(func(r *http.Request) (*http.Response, error) {
			start := time.Now()
			// tracing instrumentation
			span, ctx := opentracing.StartSpanFromContext(r.Context(), "frontend.Tripperware")
			defer span.Finish()

			orgID, _ := user.ExtractOrgID(r.Context())
			queriesPerTenant.WithLabelValues(orgID).Inc()
			span.SetTag("orgID", orgID)

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

			// for context propagation with traceID set
			r = r.WithContext(ctx)

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

			traceID, _ := middleware.ExtractTraceID(ctx)
			statusCode := 500
			var contentLength int64 = 0
			if resp != nil {
				statusCode = resp.StatusCode
				contentLength = resp.ContentLength
			} else if httpResp, ok := httpgrpc.HTTPResponseFromError(err); ok {
				statusCode = int(httpResp.Code)
				contentLength = int64(len(httpResp.Body))
			}

			level.Info(logger).Log(
				"tenant", orgID,
				"method", r.Method,
				"traceID", traceID,
				"url", r.URL.RequestURI(),
				"duration", time.Since(start).String(),
				"response_size", contentLength,
				"status", statusCode,
			)

			return resp, err
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
