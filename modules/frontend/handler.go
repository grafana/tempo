package frontend

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/httpgrpc"
	"github.com/grafana/dskit/httpgrpc/server"
	"github.com/grafana/dskit/tracing"
	"github.com/grafana/dskit/user"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// StatusClientClosedRequest is the status code for when a client request cancellation of an http request
	StatusClientClosedRequest = 499
	// nil response in ServeHTTP
	NilResponseError = "nil resp in ServeHTTP"
)

var (
	errCanceled              = httpgrpc.Errorf(StatusClientClosedRequest, context.Canceled.Error())
	errDeadlineExceeded      = httpgrpc.Errorf(http.StatusGatewayTimeout, context.DeadlineExceeded.Error())
	errRequestEntityTooLarge = httpgrpc.Errorf(http.StatusRequestEntityTooLarge, "http: request body too large")
)

// handler exists to wrap a roundtripper with an HTTP handler. It wraps all
// frontend endpoints and should only contain functionality that is common to all.
type handler struct {
	roundTripper     http.RoundTripper
	logger           log.Logger
	queriesPerTenant *prometheus.CounterVec
}

// newHandler creates a handler
func newHandler(rt http.RoundTripper, queriesPerTenant *prometheus.CounterVec, logger log.Logger) http.Handler {
	return &handler{
		roundTripper:     rt,
		logger:           logger,
		queriesPerTenant: queriesPerTenant,
	}
}

// ServeHTTP implements http.Handler
func (f *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		_ = r.Body.Close()
	}()

	ctx := r.Context()
	start := time.Now()
	orgID, _ := user.ExtractOrgID(ctx)
	traceID, _ := tracing.ExtractTraceID(ctx)

	var statusCode int
	defer func(status int) {
		f.queriesPerTenant.WithLabelValues(orgID, strconv.Itoa(status)).Inc()
	}(statusCode)

	// add orgid to existing spans
	span := opentracing.SpanFromContext(r.Context())
	if span != nil {
		span.SetTag("orgID", orgID)
	}

	resp, err := f.roundTripper.RoundTrip(r)
	if err != nil {
		statusCode = http.StatusInternalServerError
		err = writeError(w, err)
		level.Info(f.logger).Log(
			"tenant", orgID,
			"method", r.Method,
			"traceID", traceID,
			"url", r.URL.RequestURI(),
			"duration", time.Since(start).String(),
			"response_size", 0,
			"status", statusCode,
			"err", err.Error(),
		)
		return
	}

	if resp == nil {
		statusCode = http.StatusInternalServerError
		err = writeError(w, errors.New(NilResponseError))
		level.Info(f.logger).Log(
			"tenant", orgID,
			"method", r.Method,
			"traceID", traceID,
			"url", r.URL.RequestURI(),
			"duration", time.Since(start).String(),
			"response_size", 0,
			"status", statusCode,
			"err", err.Error(),
		)
		return
	}

	// write headers, status code and body
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	if resp.Body != nil {
		_, _ = io.Copy(w, resp.Body)
	}

	// request/response logging
	var contentLength int64
	if httpResp, ok := httpgrpc.HTTPResponseFromError(err); ok {
		statusCode = int(httpResp.Code)
		contentLength = int64(len(httpResp.Body))
	} else {
		statusCode = resp.StatusCode
		contentLength = resp.ContentLength
	}

	level.Info(f.logger).Log(
		"tenant", orgID,
		"method", r.Method,
		"traceID", traceID,
		"url", r.URL.RequestURI(),
		"duration", time.Since(start).String(),
		"response_size", contentLength,
		"status", statusCode,
	)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

// writeError handles writing errors to the http.ResponseWriter. It uses weavework common
// server.WriteError() to handle httpgrc errors. The handler handles all incoming HTTP requests
// to the query frontend which then distributes them via httpgrpc to the queriers. As a result
// httpgrpc errors can bubble up to here and should be translated to http errors. It returns
// httpgrpc error.
func writeError(w http.ResponseWriter, err error) error {
	switch err {
	case context.Canceled:
		err = errCanceled
	case context.DeadlineExceeded:
		err = errDeadlineExceeded
	default:
		if isRequestBodyTooLarge(err) {
			err = errRequestEntityTooLarge
		}
	}
	server.WriteError(w, err)
	return err
}

// isRequestBodyTooLarge returns true if the error is "http: request body too large".
func isRequestBodyTooLarge(err error) bool {
	return err != nil && strings.Contains(err.Error(), "http: request body too large")
}
