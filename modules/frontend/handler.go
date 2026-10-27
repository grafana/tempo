package frontend

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/grafana/dskit/flagext"
	"github.com/grafana/dskit/grpcutil"
	dskitlog "github.com/grafana/dskit/log"
	"github.com/grafana/tempo/v3/modules/frontend/pipeline"
	"github.com/grafana/tempo/v3/pkg/util"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/httpgrpc"
	"github.com/grafana/dskit/user"

	"github.com/grafana/tempo/v3/pkg/util/tracing"
)

const (
	// nil response in ServeHTTP
	NilResponseError = "nil resp in ServeHTTP"
)

var (
	errCanceled              = httpgrpc.Error(util.StatusClientClosedRequest, context.Canceled.Error())
	errDeadlineExceeded      = httpgrpc.Error(http.StatusGatewayTimeout, context.DeadlineExceeded.Error())
	errRequestEntityTooLarge = httpgrpc.Error(http.StatusRequestEntityTooLarge, "http: request body too large")

	// sensitiveHeaderNames is the deny list of HTTP header names whose values
	// carry credentials or session material and must never appear in logs.
	sensitiveHeaderNames = []string{
		"authorization",
		"proxy-authorization",
		"cookie",
		"set-cookie",
		"x-api-key",
		"x-auth-token",
		"x-amz-security-token",
		"authentication-info",
		"www-authenticate",
		"proxy-authenticate",
		"x-csrf-token",
		"x-xsrf-token",
		"x-access-token",
		"x-session-token",
		"x-forwarded-authorization",
		"x-auth-request-access-token",
		"x-id-token",
	}
)

// handler exists to wrap a roundtripper with an HTTP handler. It wraps all
// frontend endpoints and should only contain functionality that is common to all.
type handler struct {
	roundTripper     http.RoundTripper
	logger           log.Logger
	safeHeadersToLog []safeHeader
}

// newHandler creates a handler
func newHandler(LogQueryRequestHeaders flagext.StringSliceCSV, rt http.RoundTripper, logger log.Logger) http.Handler {
	return &handler{
		safeHeadersToLog: safeHeadersToLog(LogQueryRequestHeaders),
		roundTripper:     rt,
		logger:           logger,
	}
}

// validateLogQueryRequestHeaders rejects a log_query_request_headers list that
// contains headers which carry credentials or session material.
func validateLogQueryRequestHeaders(headersToLog []string) error {
	var rejected []string
	for _, h := range headersToLog {
		if isSensitiveHeaderName(h) {
			rejected = append(rejected, h)
		}
	}
	if len(rejected) > 0 {
		return fmt.Errorf("rejected headers: %s", strings.Join(rejected, ", "))
	}
	return nil
}

// ServeHTTP implements http.Handler
func (f *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		_ = r.Body.Close()
	}()

	// Install a query-shape cell so the async-weight middleware can stamp it
	// for the per-query response log lines further down the call chain.
	r = r.WithContext(pipeline.WithQueryShapeCell(r.Context()))
	ctx := r.Context()
	start := time.Now()
	orgID, _ := user.ExtractOrgID(ctx)
	traceID, _ := tracing.ExtractTraceID(ctx)

	// add orgid to existing spans
	span := trace.SpanFromContext(r.Context())
	if span != nil {
		span.SetAttributes(attribute.String("orgID", orgID))
	}

	var headerFields []interface{}
	if len(f.safeHeadersToLog) != 0 {
		headerFields = formatRequestHeaders(&r.Header, f.safeHeadersToLog)
		// Pass the header fields down so the per-query response log lines can include them too.
		r = r.WithContext(withRequestHeaderFields(r.Context(), headerFields))
	}

	resp, err := f.roundTripper.RoundTrip(r)
	elapsed := time.Since(start)

	logMessage := []interface{}{
		"msg", "query stats",
		"tenant", orgID,
		"method", r.Method,
		"traceID", traceID,
		"url", r.URL.RequestURI(),
		"duration", elapsed.String(),
	}
	logMessage = append(logMessage, headerFields...)

	if err != nil {
		statusCode := http.StatusInternalServerError
		err = writeError(w, err)
		logMessage = append(
			logMessage,
			"status", statusCode,
			"error", err.Error(),
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
	span.AddEvent("http.response.write.start")
	var written int64
	if resp.Body != nil {
		var writeErr error
		written, writeErr = io.Copy(w, resp.Body)
		if writeErr != nil {
			span.SetStatus(codes.Error, "response body write failed")
		}
	}
	span.SetAttributes(attribute.Int64("response_bytes", written))
	span.AddEvent("http.response.write.done")

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

// safeHeader is an HTTP request header that has been validated as
// safe to log: it is not in the sensitive-header deny list. Values must be
// constructed via newSafeHeader so the deny-list check cannot be bypassed.
type safeHeader string

func newSafeHeader(name string) (safeHeader, bool) {
	if isSensitiveHeaderName(name) {
		return "", false
	}
	return safeHeader(name), true
}

func (s safeHeader) String() string {
	return string(s)
}

func (s safeHeader) log() string {
	return fmt.Sprintf("header_%s", strings.ReplaceAll(strings.ToLower(string(s)), "-", "_"))
}

// skips headers regardless of the log_query_request_headers configuration.
func isSensitiveHeaderName(name string) bool {
	return slices.Contains(sensitiveHeaderNames, strings.ToLower(strings.TrimSpace(name)))
}

// safeHeadersToLog drops sensitive headers from headersToLog. Config validation
// rejects them already; this keeps them out of the logs if validation is skipped.
func safeHeadersToLog(headersToLog []string) (safeHeaders []safeHeader) {
	for _, h := range headersToLog {
		if s, ok := newSafeHeader(h); ok {
			safeHeaders = append(safeHeaders, s)
		}
	}
	return safeHeaders
}

// formatRequestHeaders returns the log fields for every header in headersToLog
// that is present in the request. Control and formatting characters are dropped
// from the values so a client cannot inject log lines.
func formatRequestHeaders(h *http.Header, headersToLog []safeHeader) (fields []interface{}) {
	for _, s := range headersToLog {
		if v := strings.TrimSpace(h.Get(s.String())); v != "" {
			fields = append(fields, s.log(), dskitlog.DropUnsafeChars(v).String())
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
	if grpcutil.IsCanceled(err) {
		err = errCanceled
	} else if errors.Is(err, context.DeadlineExceeded) {
		err = errDeadlineExceeded
	} else if util.IsRequestBodyTooLarge(err) {
		err = errRequestEntityTooLarge
	}
	httpgrpc.WriteError(w, err)
	return err
}
