package combiner

import (
	"bytes"
	"context"
	"errors"

	tracev1 "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func deduplicateTrace(ctx context.Context, result *tempopb.Trace) *tempopb.Trace {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.Int("span_count", countTraceSpans(result)))
	span.AddEvent("trace.deduplicate.start")
	// The Zipkin deduper can rewrite shared client/server IDs. Retain the span
	// identity so stored records stay aligned with the returned spans.
	type key struct {
		id   string
		kind int32
	}
	type reference struct {
		span     *tracev1.Span
		parentID string
	}
	var endpoints map[key]reference
	if result != nil && len(result.ServiceGraphEdges) > 0 {
		endpoints = make(map[key]reference)
		for _, rs := range result.ResourceSpans {
			for _, ss := range rs.ScopeSpans {
				for _, span := range ss.Spans {
					k := key{string(span.SpanId), int32(span.Kind)}
					if _, exists := endpoints[k]; !exists {
						endpoints[k] = reference{span, string(span.ParentSpanId)}
					}
				}
			}
		}
	}
	result = newDeduper().dedupe(result)
	if result != nil {
		for _, edge := range result.ServiceGraphEdges {
			if edge == nil {
				continue
			}
			ref, exists := endpoints[key{string(edge.ChildSpanId), edge.ChildKind}]
			if !exists {
				continue
			}
			if !bytes.Equal(edge.ChildSpanId, ref.span.SpanId) {
				edge.ChildSpanId = bytes.Clone(ref.span.SpanId)
			}
			if ref.parentID != string(ref.span.ParentSpanId) {
				edge.ParentSpanId = bytes.Clone(ref.span.ParentSpanId)
			}
		}
	}
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
