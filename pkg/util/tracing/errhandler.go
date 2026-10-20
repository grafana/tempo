package tracing

import (
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// RecordErr marks span as errored (RecordError + SetStatus) if err is
// non-nil, or explicitly Ok otherwise. RecordError alone does not flip a
// span's status; SetStatus is what `{ status = error }` queries key off.
func RecordErr(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}
	span.SetStatus(codes.Ok, "")
}

// EndSpan marks span via RecordErr using *err at call time, then ends it.
// Defer it right after the span is created, passing a pointer to a named
// (or otherwise addressable) err so the deferred call sees its final value.
func EndSpan(span trace.Span, err *error) {
	RecordErr(span, *err)
	span.End()
}
