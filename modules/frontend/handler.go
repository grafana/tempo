package frontend

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/grafana/dskit/flagext"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/httpgrpc"
	"github.com/grafana/dskit/user"
	"github.com/opentracing/opentracing-go"

	"github.com/grafana/tempo/pkg/util/tracing"
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
	roundTripper           http.RoundTripper
	logger                 log.Logger
	logQueryRequestHeaders flagext.StringSliceCSV
}

// newHandler creates a handler
func newHandler(LogQueryRequestHeaders flagext.StringSliceCSV, rt http.RoundTripper, logger log.Logger) http.Handler {
	return &handler{
		logQueryRequestHeaders: LogQueryRequestHeaders,
		roundTripper:           rt,
		logger:                 logger,
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

	// add orgid to existing spans
	span := opentracing.SpanFromContext(r.Context())
	if span != nil {
		span.SetTag("orgID", orgID)
	}

	resp, err := f.roundTripper.RoundTrip(r)
	elapsed := time.Since(start)

	logMessage := []interface{}{
		"tenant", orgID,
		"method", r.Method,
		"traceID", traceID,
		"url", r.URL.RequestURI(),
		"duration", elapsed.String(),
	}
	if len(f.logQueryRequestHeaders) != 0 {
		logMessage = append(logMessage, formatRequestHeaders(&r.Header, f.logQueryRequestHeaders)...)
	}

	if err != nil {
		statusCode := http.StatusInternalServerError
		err = writeError(w, err)
		logMessage = append(
			logMessage,
			"status", statusCode,
			"err", err.Error(),
			"response_size", 0,
		)
		level.Info(f.logger).Log(logMessage...)
		return
	}

	if resp == nil {
		statusCode := http.StatusInternalServerError
		err = writeError(w, errors.New(NilResponseError))
		logMessage = append(
			logMessage,
			"status", statusCode,
			"err", err.Error(),
			"response_size", 0,
		)
		level.Info(f.logger).Log(logMessage...)
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
	var statusCode int
	if httpResp, ok := httpgrpc.HTTPResponseFromError(err); ok {
		statusCode = int(httpResp.Code)
		contentLength = int64(len(httpResp.Body))
	} else {
		statusCode = resp.StatusCode
		contentLength = resp.ContentLength
	}

	logMessage = append(
		logMessage,
		"response_size", contentLength,
		"status", statusCode,
	)
	level.Info(f.logger).Log(logMessage...)
}

func formatRequestHeaders(h *http.Header, headersToLog []string) (fields []interface{}) {
	for _, s := range headersToLog {
		if v := h.Get(s); v != "" {
			fields = append(fields, fmt.Sprintf("header_%s", strings.ReplaceAll(strings.ToLower(s), "-", "_")), v)
		}
	}
	return fields
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

// writeError handles writing errors to the http.ResponseWriter. It uses dskit's
// httpgrpc.WriteError() to handle httpgrc errors. The handler handles all incoming HTTP requests
// to the query frontend which then distributes them via httpgrpc to the queriers. As a result
// httpgrpc errors can bubble up to here and should be translated to http errors. It returns
// httpgrpc error.
func writeError(w http.ResponseWriter, err error) error {
	if errors.Is(err, context.Canceled) {
		err = errCanceled
	} else if errors.Is(err, context.DeadlineExceeded) {
		err = errDeadlineExceeded
	} else if isRequestBodyTooLarge(err) {
		err = errRequestEntityTooLarge
	}
	httpgrpc.WriteError(w, err)
	return err
}

// isRequestBodyTooLarge returns true if the error is "http: request body too large".
func isRequestBodyTooLarge(err error) bool {
	return err != nil && strings.Contains(err.Error(), "http: request body too large")
}
