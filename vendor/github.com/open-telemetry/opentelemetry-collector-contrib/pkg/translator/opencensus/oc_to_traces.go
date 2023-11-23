// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package opencensus // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/opencensus"

import (
	"strings"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	octrace "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"go.opencensus.io/trace"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	conventions "go.opentelemetry.io/collector/semconv/v1.6.1"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/occonventions"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/tracetranslator"
)

// OCToTraces may be used only by OpenCensus receiver and exporter implementations.
// Deprecated: use ptrace.Traces instead.
// TODO: move this function to OpenCensus package.
func OCToTraces(node *occommon.Node, resource *ocresource.Resource, spans []*octrace.Span) ptrace.Traces {
	traceData := ptrace.NewTraces()
	if node == nil && resource == nil && len(spans) == 0 {
		return traceData
	}

	if len(spans) == 0 {
		// At least one of the td.Node or td.Resource is not nil. Set the resource and return.
		ocNodeResourceToInternal(node, resource, traceData.ResourceSpans().AppendEmpty().Resource())
		return traceData
	}

	// We may need to split OC spans into several ResourceSpans. OC Spans can have a
	// Resource field inside them set to nil which indicates they use the Resource
	// specified in "td.Resource", or they can have the Resource field inside them set
	// to non-nil which indicates they have overridden Resource field and "td.Resource"
	// does not apply to those spans.
	//
	// Each OC Span that has its own Resource field set to non-nil must be placed in a
	// separate ResourceSpans instance, containing only that span. All other OC Spans
	// that have nil Resource field must be placed in one other ResourceSpans instance,
	// which will gets its Resource field from "td.Resource".
	//
	// We will end up with with one or more ResourceSpans like this:
	//
	// ResourceSpans           ResourceSpans  ResourceSpans
	// +-----+-----+---+-----+ +------------+ +------------+
	// |Span1|Span2|...|SpanM| |Span        | |Span        | ...
	// +-----+-----+---+-----+ +------------+ +------------+

	// Count the number of spans that have nil Resource and need to be combined
	// in one slice.
	combinedSpanCount := 0
	distinctResourceCount := 0
	for _, ocSpan := range spans {
		if ocSpan == nil {
			// Skip nil spans.
			continue
		}
		if ocSpan.Resource == nil {
			combinedSpanCount++
		} else {
			distinctResourceCount++
		}
	}
	// Total number of resources is equal to:
	// 1 (for all spans with nil resource) + numSpansWithResource (distinctResourceCount).

	rss := traceData.ResourceSpans()
	rss.EnsureCapacity(distinctResourceCount + 1)
	rs0 := rss.AppendEmpty()
	ocNodeResourceToInternal(node, resource, rs0.Resource())

	// Allocate a slice for spans that need to be combined into first ResourceSpans.
	ils0 := rs0.ScopeSpans().AppendEmpty()
	combinedSpans := ils0.Spans()
	combinedSpans.EnsureCapacity(combinedSpanCount)

	// Now do the span translation and place them in appropriate ResourceSpans
	// instances.

	for _, ocSpan := range spans {
		if ocSpan == nil {
			// Skip nil spans.
			continue
		}

		if ocSpan.Resource == nil {
			// Add the span to the "combinedSpans". combinedSpans length is equal
			// to combinedSpanCount. The loop above that calculates combinedSpanCount
			// has exact same conditions as we have here in this loop.
			ocSpanToInternal(ocSpan, combinedSpans.AppendEmpty())
		} else {
			// This span has a different Resource and must be placed in a different
			// ResourceSpans instance. Create a separate ResourceSpans item just for this span.
			ocSpanToResourceSpans(ocSpan, node, traceData.ResourceSpans().AppendEmpty())
		}
	}

	return traceData
}

func ocSpanToResourceSpans(ocSpan *octrace.Span, node *occommon.Node, dest ptrace.ResourceSpans) {
	ocNodeResourceToInternal(node, ocSpan.Resource, dest.Resource())
	ilss := dest.ScopeSpans()
	ocSpanToInternal(ocSpan, ilss.AppendEmpty().Spans().AppendEmpty())
}

func ocSpanToInternal(src *octrace.Span, dest ptrace.Span) {
	// Note that ocSpanKindToInternal must be called before initAttributeMapFromOC
	// since it may modify src.Attributes (remove the attribute which represents the
	// span kind).
	dest.SetKind(ocSpanKindToInternal(src.Kind, src.Attributes))

	dest.SetTraceID(traceIDToInternal(src.TraceId))
	dest.SetSpanID(spanIDToInternal(src.SpanId))
	dest.TraceState().FromRaw(ocTraceStateToInternal(src.Tracestate))
	dest.SetParentSpanID(spanIDToInternal(src.ParentSpanId))

	dest.SetName(src.Name.GetValue())
	dest.SetStartTimestamp(pcommon.NewTimestampFromTime(src.StartTime.AsTime()))
	dest.SetEndTimestamp(pcommon.NewTimestampFromTime(src.EndTime.AsTime()))

	ocStatusToInternal(src.Status, src.Attributes, dest.Status())

	initAttributeMapFromOC(src.Attributes, dest.Attributes())
	dest.SetDroppedAttributesCount(ocAttrsToDroppedAttributes(src.Attributes))
	ocEventsToInternal(src.TimeEvents, dest)
	ocLinksToInternal(src.Links, dest)
	ocSameProcessAsParentSpanToInternal(src.SameProcessAsParentSpan, dest)
}

// Transforms the byte slice trace ID into a [16]byte internal pcommon.TraceID.
// If larger input then it is truncated to 16 bytes.
func traceIDToInternal(traceID []byte) pcommon.TraceID {
	tid := [16]byte{}
	copy(tid[:], traceID)
	return pcommon.TraceID(tid)
}

// Transforms the byte slice span ID into a [8]byte internal pcommon.SpanID.
// If larger input then it is truncated to 8 bytes.
func spanIDToInternal(spanID []byte) pcommon.SpanID {
	sid := [8]byte{}
	copy(sid[:], spanID)
	return pcommon.SpanID(sid)
}

func ocStatusToInternal(ocStatus *octrace.Status, ocAttrs *octrace.Span_Attributes, dest ptrace.Status) {
	if ocStatus == nil {
		return
	}

	var code ptrace.StatusCode
	switch ocStatus.Code {
	case trace.StatusCodeOK:
		code = ptrace.StatusCodeUnset
	default:
		// all other OC status codes are errors.
		code = ptrace.StatusCodeError
	}

	if ocAttrs != nil {
		// If conventions.OtelStatusCode is set, it must override the status code value.
		// See the reverse translation in traces_to_oc.go:statusToOC().
		if attr, ok := ocAttrs.AttributeMap[conventions.OtelStatusCode]; ok {
			code = ptrace.StatusCode(attr.GetIntValue())
			delete(ocAttrs.AttributeMap, conventions.OtelStatusCode)
		}
	}

	dest.SetCode(code)
	dest.SetMessage(ocStatus.Message)
}

// Convert tracestate to W3C format. See the https://w3c.github.io/trace-context/
func ocTraceStateToInternal(ocTracestate *octrace.Span_Tracestate) string {
	if ocTracestate == nil {
		return ""
	}
	var sb strings.Builder
	for i, entry := range ocTracestate.Entries {
		sb.Grow(1 + len(entry.Key) + 1 + len(entry.Value))
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(entry.Key)
		sb.WriteString("=")
		sb.WriteString(entry.Value)
	}
	return sb.String()
}

func ocAttrsToDroppedAttributes(ocAttrs *octrace.Span_Attributes) uint32 {
	if ocAttrs == nil {
		return 0
	}
	return uint32(ocAttrs.DroppedAttributesCount)
}

// initAttributeMapFromOC initialize AttributeMap from OC attributes
func initAttributeMapFromOC(ocAttrs *octrace.Span_Attributes, dest pcommon.Map) {
	if ocAttrs == nil || len(ocAttrs.AttributeMap) == 0 {
		return
	}

	dest.EnsureCapacity(len(ocAttrs.AttributeMap))
	for key, ocAttr := range ocAttrs.AttributeMap {
		switch attribValue := ocAttr.Value.(type) {
		case *octrace.AttributeValue_StringValue:
			dest.PutStr(key, attribValue.StringValue.GetValue())
		case *octrace.AttributeValue_IntValue:
			dest.PutInt(key, attribValue.IntValue)
		case *octrace.AttributeValue_BoolValue:
			dest.PutBool(key, attribValue.BoolValue)
		case *octrace.AttributeValue_DoubleValue:
			dest.PutDouble(key, attribValue.DoubleValue)
		default:
			dest.PutStr(key, "<Unknown OpenCensus attribute value type>")
		}
	}
}

func ocSpanKindToInternal(ocKind octrace.Span_SpanKind, ocAttrs *octrace.Span_Attributes) ptrace.SpanKind {
	switch ocKind {
	case octrace.Span_SERVER:
		return ptrace.SpanKindServer

	case octrace.Span_CLIENT:
		return ptrace.SpanKindClient

	case octrace.Span_SPAN_KIND_UNSPECIFIED:
		// Span kind field is unspecified, check if TagSpanKind attribute is set.
		// This can happen if span kind had no equivalent in OC, so we could represent it in
		// the SpanKind. In that case the span kind may be a special attribute TagSpanKind.
		if ocAttrs != nil {
			kindAttr := ocAttrs.AttributeMap[tracetranslator.TagSpanKind]
			if kindAttr != nil {
				strVal, ok := kindAttr.Value.(*octrace.AttributeValue_StringValue)
				if ok && strVal != nil {
					var otlpKind ptrace.SpanKind
					switch tracetranslator.OpenTracingSpanKind(strVal.StringValue.GetValue()) {
					case tracetranslator.OpenTracingSpanKindConsumer:
						otlpKind = ptrace.SpanKindConsumer
					case tracetranslator.OpenTracingSpanKindProducer:
						otlpKind = ptrace.SpanKindProducer
					case tracetranslator.OpenTracingSpanKindInternal:
						otlpKind = ptrace.SpanKindInternal
					default:
						return ptrace.SpanKindUnspecified
					}
					delete(ocAttrs.AttributeMap, tracetranslator.TagSpanKind)
					return otlpKind
				}
			}
		}
		return ptrace.SpanKindUnspecified

	default:
		return ptrace.SpanKindUnspecified
	}
}

func ocEventsToInternal(ocEvents *octrace.Span_TimeEvents, dest ptrace.Span) {
	if ocEvents == nil {
		return
	}

	dest.SetDroppedEventsCount(uint32(ocEvents.DroppedMessageEventsCount + ocEvents.DroppedAnnotationsCount))

	if len(ocEvents.TimeEvent) == 0 {
		return
	}

	events := dest.Events()
	events.EnsureCapacity(len(ocEvents.TimeEvent))

	for _, ocEvent := range ocEvents.TimeEvent {
		if ocEvent == nil {
			// Skip nil source events.
			continue
		}

		event := events.AppendEmpty()
		event.SetTimestamp(pcommon.NewTimestampFromTime(ocEvent.Time.AsTime()))

		switch teValue := ocEvent.Value.(type) {
		case *octrace.Span_TimeEvent_Annotation_:
			if teValue.Annotation != nil {
				event.SetName(teValue.Annotation.Description.GetValue())
				initAttributeMapFromOC(teValue.Annotation.Attributes, event.Attributes())
				event.SetDroppedAttributesCount(ocAttrsToDroppedAttributes(teValue.Annotation.Attributes))
			}

		case *octrace.Span_TimeEvent_MessageEvent_:
			event.SetName("message")
			ocMessageEventToInternalAttrs(teValue.MessageEvent, event.Attributes())
			// No dropped attributes for this case.
			event.SetDroppedAttributesCount(0)

		default:
			event.SetName("An unknown OpenCensus TimeEvent type was detected when translating")
		}
	}
}

func ocLinksToInternal(ocLinks *octrace.Span_Links, dest ptrace.Span) {
	if ocLinks == nil {
		return
	}

	dest.SetDroppedLinksCount(uint32(ocLinks.DroppedLinksCount))

	if len(ocLinks.Link) == 0 {
		return
	}

	links := dest.Links()
	links.EnsureCapacity(len(ocLinks.Link))

	for _, ocLink := range ocLinks.Link {
		if ocLink == nil {
			continue
		}

		link := links.AppendEmpty()
		link.SetTraceID(traceIDToInternal(ocLink.TraceId))
		link.SetSpanID(spanIDToInternal(ocLink.SpanId))
		link.TraceState().FromRaw(ocTraceStateToInternal(ocLink.Tracestate))
		initAttributeMapFromOC(ocLink.Attributes, link.Attributes())
		link.SetDroppedAttributesCount(ocAttrsToDroppedAttributes(ocLink.Attributes))
	}
}

func ocMessageEventToInternalAttrs(msgEvent *octrace.Span_TimeEvent_MessageEvent, dest pcommon.Map) {
	if msgEvent == nil {
		return
	}

	dest.PutStr("message.type", msgEvent.Type.String())
	dest.PutInt(conventions.AttributeMessagingMessageID, int64(msgEvent.Id))
	dest.PutInt(conventions.AttributeMessagingMessagePayloadSizeBytes, int64(msgEvent.UncompressedSize))
	dest.PutInt(conventions.AttributeMessagingMessagePayloadCompressedSizeBytes, int64(msgEvent.CompressedSize))
}

func ocSameProcessAsParentSpanToInternal(spaps *wrapperspb.BoolValue, dest ptrace.Span) {
	if spaps == nil {
		return
	}
	dest.Attributes().PutBool(occonventions.AttributeSameProcessAsParentSpan, spaps.Value)
}
