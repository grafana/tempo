package util

import (
	"context"

	"github.com/grafana/dskit/tracing"
	"go.opentelemetry.io/otel/trace"
)

// ExtractTraceID extracts the trace id, if any from the context.
func ExtractTraceID(ctx context.Context) (string, bool) {
	// Extract from OpenTracing Jaeger exporter
	traceID, ok := tracing.ExtractTraceID(ctx)
	if ok {
		return traceID, true
	}

	// Extract from OpenTelemetry
	otelSpan := trace.SpanFromContext(ctx)
	if otelSpan.SpanContext().HasTraceID() {
		return otelSpan.SpanContext().TraceID().String(), true
	}

	return "", false
}
