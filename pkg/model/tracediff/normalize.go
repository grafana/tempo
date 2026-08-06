package tracediff

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"

	modeltrace "github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

const serviceNameAttribute = "service.name"

type normalizedTrace struct {
	meta  TraceMeta
	spans []normalizedSpan
}

type normalizedSpan struct {
	spanID         string
	parentIdentity spanLogicalKey
	hasParent      bool
	ref            SpanRef
	snapshot       SpanSnapshot
	startUnixNano  uint64
	endUnixNano    uint64
	durationValid  bool
	spanAttrs      map[string]any
}

type spanLogicalKey struct {
	service string
	name    string
	kind    string
}

type spanWithResource struct {
	span            *tracev1.Span
	resourceService string
}

func normalizeTrace(trace *tempopb.Trace) (normalizedTrace, []Warning) {
	return normalizeTraceForSide(trace, "")
}

func normalizeTraceForSide(trace *tempopb.Trace, side string) (normalizedTrace, []Warning) {
	if trace == nil {
		return normalizedTrace{}, nil
	}

	spans, meta := flattenSpans(trace)
	warnings := highCardinalitySpanNameWarnings(spans)
	warnings = append(warnings, invalidDurationWarnings(spans, side)...)
	if len(spans) == 0 {
		warnings = append(warnings, Warning{
			Code:    WarningZeroSpanTrace,
			Message: fmt.Sprintf("%s contains no spans; comparison has no span matches", sideTraceLabel(side)),
		})
	}
	byID := make(map[string]spanWithResource, len(spans))
	children := map[string][]spanWithResource{}
	for _, span := range spans {
		spanID := spanIDKey(span.span.GetSpanId())
		byID[spanID] = span
	}
	for _, span := range spans {
		parentID := spanIDKey(span.span.GetParentSpanId())
		if _, ok := byID[parentID]; parentID != "" && !ok {
			parentID = ""
		}
		children[parentID] = append(children[parentID], span)
	}
	for parentID := range children {
		sortSpans(children[parentID])
	}

	// out.spans is built in pre-order (a node, then its children).
	out := normalizedTrace{meta: meta, spans: make([]normalizedSpan, 0, len(spans))}
	visited := make(map[string]struct{}, len(spans))
	// First assign paths from normal roots and orphans.
	rootIndex := assignPaths(&out, children, children[""], nil, spanLogicalKey{}, false, 0, visited)
	// Then assign remaining cycle-only/disconnected spans as extra roots.
	remaining := append([]spanWithResource(nil), spans...)
	sortSpans(remaining)
	assignPaths(&out, children, remaining, nil, spanLogicalKey{}, false, rootIndex, visited)
	if dropped := meta.SpanCount - len(out.spans); dropped > 0 {
		warnings = append(warnings, Warning{
			Code:    WarningDuplicateSpanID,
			Message: fmt.Sprintf("%s has %d span(s) with duplicate span IDs; duplicates are ignored by the diff and counts may disagree", sideTraceLabel(side), dropped),
		})
	}
	return out, warnings
}

func sideTraceLabel(side string) string {
	if side == "" {
		return "trace"
	}
	return side + " trace"
}

func flattenSpans(trace *tempopb.Trace) ([]spanWithResource, TraceMeta) {
	var meta TraceMeta
	var spans []spanWithResource
	for _, rs := range trace.ResourceSpans {
		resourceService := attributeString(rs.GetResource().GetAttributes(), serviceNameAttribute)
		for _, ss := range rs.ScopeSpans {
			for _, span := range ss.Spans {
				meta.SpanCount++
				if meta.TraceID == "" && len(span.TraceId) > 0 {
					meta.TraceID = hex.EncodeToString(span.TraceId)
				}
				spans = append(spans, spanWithResource{
					span:            span,
					resourceService: resourceService,
				})
			}
		}
	}
	return spans, meta
}

func assignPaths(out *normalizedTrace, children map[string][]spanWithResource, spans []spanWithResource, parentPath []int, parentIdentity spanLogicalKey, hasParent bool, startIndex int, visited map[string]struct{}) int {
	nextIndex := startIndex
	for _, span := range spans {
		spanID := spanIDKey(span.span.GetSpanId())
		if _, ok := visited[spanID]; ok {
			continue
		}

		path := make([]int, len(parentPath)+1)
		copy(path, parentPath)
		path[len(parentPath)] = nextIndex
		nextIndex++

		logicalKey := logicalKey(span)
		out.spans = append(out.spans, normalizeSpan(span, path, logicalKey, parentIdentity, hasParent))
		visited[spanID] = struct{}{}
		assignPaths(out, children, children[spanID], path, logicalKey, true, 0, visited)
	}
	return nextIndex
}

func normalizeSpan(span spanWithResource, path []int, logicalKey spanLogicalKey, parentIdentity spanLogicalKey, hasParent bool) normalizedSpan {
	ref := SpanRef{
		Path:    path,
		Service: logicalKey.service,
		Name:    logicalKey.name,
		Kind:    logicalKey.kind,
	}
	return normalizedSpan{
		spanID:         spanIDKey(span.span.GetSpanId()),
		parentIdentity: parentIdentity,
		hasParent:      hasParent,
		ref:            ref,
		startUnixNano:  span.span.GetStartTimeUnixNano(),
		endUnixNano:    span.span.GetEndTimeUnixNano(),
		durationValid:  hasValidDuration(span.span),
		spanAttrs:      attributesMap(span.span.GetAttributes()),
		snapshot: SpanSnapshot{
			Path:          path,
			Service:       ref.Service,
			Name:          ref.Name,
			Kind:          ref.Kind,
			DurationNanos: durationNanos(span.span),
			Status:        statusToString(span.span.GetStatus()),
		},
	}
}

func invalidDurationWarnings(spans []spanWithResource, side string) []Warning {
	invalid := make([]spanLogicalKey, 0)
	for _, span := range spans {
		if hasValidDuration(span.span) {
			continue
		}
		invalid = append(invalid, logicalKey(span))
	}
	if len(invalid) == 0 {
		return nil
	}

	examples := invalidDurationExamples(invalid, 3)
	message := fmt.Sprintf(
		"%s has %d span(s) with unset or invalid duration; matched duration changes use null on the invalid side; examples: %s",
		sideTraceLabel(side),
		len(invalid),
		examples,
	)
	return []Warning{{Code: WarningInvalidDuration, Message: message}}
}

func invalidDurationExamples(invalid []spanLogicalKey, limit int) string {
	invalid = append([]spanLogicalKey(nil), invalid...)
	sort.Slice(invalid, func(i, j int) bool {
		if invalid[i].service != invalid[j].service {
			return invalid[i].service < invalid[j].service
		}
		if invalid[i].name != invalid[j].name {
			return invalid[i].name < invalid[j].name
		}
		return invalid[i].kind < invalid[j].kind
	})
	if len(invalid) < limit {
		limit = len(invalid)
	}
	examples := make([]string, 0, limit)
	for _, key := range invalid[:limit] {
		examples = append(examples, fmt.Sprintf("%q service %q kind %q", key.name, key.service, key.kind))
	}
	if len(invalid) > limit {
		examples = append(examples, fmt.Sprintf("and %d more", len(invalid)-limit))
	}
	return fmt.Sprintf("[%s]", strings.Join(examples, "; "))
}

func logicalKey(span spanWithResource) spanLogicalKey {
	service := attributeString(span.span.GetAttributes(), serviceNameAttribute)
	if service == "" {
		service = span.resourceService
	}
	return spanLogicalKey{
		service: service,
		name:    span.span.GetName(),
		kind:    modeltrace.KindToString(span.span.GetKind()),
	}
}

func (s normalizedSpan) identity() spanLogicalKey {
	return spanLogicalKey{service: s.ref.Service, name: s.ref.Name, kind: s.ref.Kind}
}

func highCardinalitySpanNameWarnings(spans []spanWithResource) []Warning {
	for _, span := range spans {
		key := logicalKey(span)
		if !spanNameHasHighCardinalityToken(key.name) {
			continue
		}
		return []Warning{{
			Code: WarningHighCardinalitySpanName,
			Message: fmt.Sprintf(
				"span name %q for service %q kind %q appears to contain a high-cardinality token; matching uses raw span names, so equivalent operations with request-specific IDs may be reported as added/removed. Move request-specific IDs to span attributes.",
				key.name,
				key.service,
				key.kind,
			),
		}}
	}
	return nil
}

// Warn on obvious request-specific IDs without rewriting the span name:
// long numeric tokens or long hex/UUID-like tokens.
var reHighCardinalitySpanNameToken = regexp.MustCompile(`(?i)\b(?:[0-9]{6,}|[0-9a-f][0-9a-f-]{7,})\b`)

func spanNameHasHighCardinalityToken(name string) bool {
	return reHighCardinalitySpanNameToken.MatchString(name)
}

func sortSpans(spans []spanWithResource) {
	sort.Slice(spans, func(i, j int) bool {
		if spans[i].span.GetStartTimeUnixNano() != spans[j].span.GetStartTimeUnixNano() {
			return spans[i].span.GetStartTimeUnixNano() < spans[j].span.GetStartTimeUnixNano()
		}
		return bytes.Compare(spans[i].span.GetSpanId(), spans[j].span.GetSpanId()) < 0
	})
}

func spanIDKey(id []byte) string {
	return string(id)
}

func attributeString(attrs []*commonv1.KeyValue, key string) string {
	for _, attr := range attrs {
		if attr.GetKey() == key {
			return attr.GetValue().GetStringValue()
		}
	}
	return ""
}

func attributesMap(attrs []*commonv1.KeyValue) map[string]any {
	out := make(map[string]any, len(attrs))
	for _, attr := range attrs {
		out[attr.GetKey()] = anyValue(attr.GetValue())
	}
	return out
}

func anyValue(value *commonv1.AnyValue) any {
	if value == nil {
		return nil
	}
	switch v := value.GetValue().(type) {
	case *commonv1.AnyValue_StringValue:
		return v.StringValue
	case *commonv1.AnyValue_BoolValue:
		return v.BoolValue
	case *commonv1.AnyValue_IntValue:
		return v.IntValue
	case *commonv1.AnyValue_DoubleValue:
		return v.DoubleValue
	case *commonv1.AnyValue_ArrayValue:
		values := v.ArrayValue.GetValues()
		out := make([]any, 0, len(values))
		for _, item := range values {
			out = append(out, anyValue(item))
		}
		return out
	case *commonv1.AnyValue_KvlistValue:
		return attributesMap(v.KvlistValue.GetValues())
	case *commonv1.AnyValue_BytesValue:
		return v.BytesValue
	default:
		return nil
	}
}

func durationNanos(span *tracev1.Span) int64 {
	if !hasValidDuration(span) {
		return 0
	}
	return int64(span.GetEndTimeUnixNano() - span.GetStartTimeUnixNano())
}

func hasValidDuration(span *tracev1.Span) bool {
	start := span.GetStartTimeUnixNano()
	end := span.GetEndTimeUnixNano()
	// A start of 0 is unset (OTLP starts are non-zero); treating it as the epoch
	// would report a multi-decade duration against a real end. An end before the
	// start or a duration that cannot fit in int64 is likewise unusable.
	return start > 0 && end >= start && end-start <= math.MaxInt64
}

func statusToString(status *tracev1.Status) string {
	if status == nil {
		return modeltrace.StatusToString(tracev1.Status_STATUS_CODE_UNSET)
	}
	return modeltrace.StatusToString(status.GetCode())
}
