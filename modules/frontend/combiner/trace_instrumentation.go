package combiner

import (
	"context"
	"errors"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func deduplicateTrace(ctx context.Context, result *tempopb.Trace) *tempopb.Trace {
	spanCount := attribute.Int("span_count", countTraceSpans(result))

	tracer := trace.SpanFromContext(ctx).TracerProvider().Tracer("modules/frontend/combiner")
	_, span := tracer.Start(ctx, "trace.deduplicate", trace.WithAttributes(spanCount))
	defer span.End()

	return newDeduper().dedupe(result)
}

func redactTrace(ctx context.Context, result *tempopb.Trace, redactor TraceRedactor) error {
	tracer := trace.SpanFromContext(ctx).TracerProvider().Tracer("modules/frontend/combiner")
	_, span := tracer.Start(ctx, "trace.redact", trace.WithAttributes(
		attribute.Int("span_count", countTraceSpans(result)),
	))
	defer span.End()

	err := redactor.RedactTraceAttributes(result)
	if err != nil && !errors.Is(err, ErrTraceHidden) {
		span.SetStatus(codes.Error, "trace redaction failed")
	}
	return err
}
