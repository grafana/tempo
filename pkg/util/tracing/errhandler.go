package tracing

import (
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// RecordErr marks span as errored (RecordError + SetStatus) when err is
// non-nil, or explicitly Ok when it is nil. Call from a defer, right after
// the span is created, closing over a named or otherwise reachable err:
//
//	func doThing(ctx context.Context) (err error) {
//		ctx, span := tracer.Start(ctx, "doThing")
//		defer func() { tracing.RecordErr(span, err); span.End() }()
//		...
//		return err
//	}
//
// Both calls matter: RecordError attaches the error as an exception event;
// SetStatus flips the span's status field, which is what `{ status = error }`
// TraceQL queries and error-rate dashboards actually key off. RecordError
// alone does not mark a span as errored.
func RecordErr(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}
	span.SetStatus(codes.Ok, "")
}
