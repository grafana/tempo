package spanlogger

import (
	"context"
	"fmt"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/grafana/dskit/tenant"
	util_log "github.com/grafana/tempo/pkg/util/log"
)

type loggerCtxMarker struct{}

const (
	TenantIDTagName = "tenant_ids"
)

var (
	tracer          = otel.Tracer("")
	defaultNoopSpan = noop.Span{}
	loggerCtxKey    = &loggerCtxMarker{}
)

// SpanLogger unifies tracing and logging, to reduce repetition.
type SpanLogger struct {
	log.Logger
	trace.Span
}

// New makes a new SpanLogger, where logs will be sent to the global logger.
func New(ctx context.Context, method string, kvps ...interface{}) (*SpanLogger, context.Context) {
	return NewWithLogger(ctx, util_log.Logger, method, kvps...)
}

// NewWithLogger makes a new SpanLogger with a custom log.Logger to send logs
// to. The provided context will have the logger attached to it and can be
// retrieved with FromContext or FromContextWithFallback.
func NewWithLogger(ctx context.Context, l log.Logger, method string, kvps ...interface{}) (*SpanLogger, context.Context) {
	ctx, span := tracer.Start(ctx, method)
	if ids, _ := tenant.TenantIDs(ctx); len(ids) > 0 {
		span.SetAttributes(attribute.StringSlice(TenantIDTagName, ids))
	}
	logger := &SpanLogger{
		Logger: log.With(util_log.WithContext(ctx, l), "method", method),
		Span:   span,
	}
	if len(kvps) > 0 {
		level.Debug(logger).Log(kvps...)
	}

	ctx = context.WithValue(ctx, loggerCtxKey, l)
	return logger, ctx
}

// FromContext returns a span logger using the current parent span. If there
// is no parent span, the SpanLogger will only log to the logger
// in the context. If the context doesn't have a logger, the global logger
// is used.
func FromContext(ctx context.Context) *SpanLogger {
	return FromContextWithFallback(ctx, util_log.Logger)
}

// FromContextWithFallback returns a span logger using the current parent span.
// IF there is no parent span, the SpanLogger will only log to the logger
// within the context. If the context doesn't have a logger, the fallback
// logger is used.
func FromContextWithFallback(ctx context.Context, fallback log.Logger) *SpanLogger {
	logger, ok := ctx.Value(loggerCtxKey).(log.Logger)
	if !ok {
		logger = fallback
	}
	sp := trace.SpanFromContext(ctx)
	if sp == nil {
		sp = defaultNoopSpan
	}
	return &SpanLogger{
		Logger: util_log.WithContext(ctx, logger),
		Span:   sp,
	}
}

// Log implements gokit's Logger interface; sends logs to underlying logger and
// also puts the on the spans.
func (s *SpanLogger) Log(kvps ...interface{}) error {
	s.Logger.Log(kvps...)

	for i := 0; i*2 < len(kvps); i++ {
		key, ok := kvps[i*2].(string)
		if !ok {
			return fmt.Errorf("non-string key (pair #%d): %T", i, kvps[i*2])
		}

		switch t := kvps[i*2+1].(type) {
		case bool:
			s.Span.SetAttributes(attribute.Bool(key, t))
		case string:
			s.Span.SetAttributes(attribute.String(key, t))
		}
	}
	return nil
}

// Error sets error flag and logs the error on the span, if non-nil.  Returns the err passed in.
func (s *SpanLogger) Error(err error) error {
	if err == nil {
		return nil
	}

	s.Span.SetStatus(codes.Error, "")
	s.Span.RecordError(err)
	return err
}
