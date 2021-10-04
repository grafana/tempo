package frontend

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/cortexproject/cortex/pkg/util"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/opentracing/opentracing-go"
	"github.com/weaveworks/common/httpgrpc"
	"github.com/weaveworks/common/httpgrpc/server"
	"github.com/weaveworks/common/tracing"
	"github.com/weaveworks/common/user"
)

const (
	// StatusClientClosedRequest is the status code for when a client request cancellation of an http request
	StatusClientClosedRequest = 499
)

var (
	errCanceled              = httpgrpc.Errorf(StatusClientClosedRequest, context.Canceled.Error())
	errDeadlineExceeded      = httpgrpc.Errorf(http.StatusGatewayTimeout, context.DeadlineExceeded.Error())
	errRequestEntityTooLarge = httpgrpc.Errorf(http.StatusRequestEntityTooLarge, "http: request body too large")
)

// Handler exists to wrap a roundtripper with an HTTP handler. It wraps all
// frontend endpoints and should only contain functionality that is common to all.
type Handler struct {
	roundTripper http.RoundTripper
	logger       log.Logger
}

// NewHandler creates a handler
func NewHandler(rt http.RoundTripper, logger log.Logger) http.Handler {
	return &Handler{
		roundTripper: rt,
		logger:       logger,
	}
}

// ServeHTTP implements http.Handler
func (f *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		_ = r.Body.Close()
	}()

	ctx := r.Context()
	start := time.Now()
	orgID, _ := user.ExtractOrgID(ctx)

	// add orgid to existing spans
	span := opentracing.SpanFromContext(r.Context())
	if span != nil {
		span.SetTag("orgID", orgID)
	}

	resp, err := f.roundTripper.RoundTrip(r)
	if err != nil {
		writeError(w, err)
		return
	}

	if resp == nil {
		writeError(w, errors.New("nil resp in ServerHTTP"))
		return
	}

	// write header and body
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)

	// request/response logging
	traceID, _ := tracing.ExtractTraceID(ctx)
	var statusCode int
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

// writeError handles writing errors to the http.ResponseWriter. It uses weavework common
// server.WriteError() to handle httpgrc errors. The handler handles all incoming HTTP requests
// to the query frontend which then distributes them via httpgrpc to the queriers. As a result
// httpgrpc errors can bubble up to here and should be translated to http errors.
func writeError(w http.ResponseWriter, err error) {
	switch err {
	case context.Canceled:
		err = errCanceled
	case context.DeadlineExceeded:
		err = errDeadlineExceeded
	default:
		if util.IsRequestBodyTooLarge(err) {
			err = errRequestEntityTooLarge
		}
	}
	server.WriteError(w, err)
}
