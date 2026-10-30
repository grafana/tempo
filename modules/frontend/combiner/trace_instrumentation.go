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
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.Int("span_count", countTraceSpans(result)))
	span.AddEvent("trace.deduplicate.start")
	result = newDeduper().dedupe(result)
	span.AddEvent("trace.deduplicate.done")
	return result
}

func redactTrace(ctx context.Context, result *tempopb.Trace, redactor TraceRedactor) error {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.Int("span_count", countTraceSpans(result)))
	span.AddEvent("trace.redact.start")
	err := redactor.RedactTraceAttributes(result)
	if err != nil && !errors.Is(err, ErrTraceHidden) {
		span.SetStatus(codes.Error, "trace redaction failed")
	}
	span.AddEvent("trace.redact.done")
	return err
}
