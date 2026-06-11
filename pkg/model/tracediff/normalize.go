package tracediff

import (
	"bytes"
	"encoding/hex"
	"sort"
	"strconv"
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
	spanID    string
	matchKey  spanMatchKey
	ref       SpanRef
	snapshot  SpanSnapshot
	spanAttrs map[string]any
}

type spanMatchKey struct {
	logicalPath string
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

func normalizeTrace(trace *tempopb.Trace) normalizedTrace {
	if trace == nil {
		return normalizedTrace{}
	}

	spans, meta := flattenSpans(trace)
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

	out := normalizedTrace{meta: meta, spans: make([]normalizedSpan, 0, len(spans))}
	visited := make(map[string]struct{}, len(spans))
	rootOrdinals := map[spanLogicalKey]int{}
	// First assign paths from normal roots and orphans.
	rootIndex := assignPaths(&out, children, children[""], nil, "", rootOrdinals, 0, visited)
	// Then assign remaining cycle-only/disconnected spans as extra roots.
	assignPaths(&out, children, spans, nil, "", rootOrdinals, rootIndex, visited)
	return out
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

func assignPaths(out *normalizedTrace, children map[string][]spanWithResource, spans []spanWithResource, parentPath []int, parentMatchPath string, ordinals map[spanLogicalKey]int, startIndex int, visited map[string]struct{}) int {
	nextIndex := startIndex
	if ordinals == nil {
		ordinals = map[spanLogicalKey]int{}
	}
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
		ordinal := ordinals[logicalKey]
		ordinals[logicalKey]++
		matchPath := appendLogicalPath(parentMatchPath, logicalKey, ordinal)

		out.spans = append(out.spans, normalizeSpan(span, path, logicalKey, matchPath))
		visited[spanID] = struct{}{}
		assignPaths(out, children, children[spanID], path, matchPath, nil, 0, visited)
	}
	return nextIndex
}

func normalizeSpan(span spanWithResource, path []int, logicalKey spanLogicalKey, matchPath string) normalizedSpan {
	ref := SpanRef{
		Path:    path,
		Service: logicalKey.service,
		Name:    logicalKey.name,
		Kind:    logicalKey.kind,
	}
	return normalizedSpan{
		spanID:    spanIDKey(span.span.GetSpanId()),
		matchKey:  spanMatchKey{logicalPath: matchPath},
		ref:       ref,
		spanAttrs: attributesMap(span.span.GetAttributes()),
		snapshot: SpanSnapshot{
			Path:       path,
			Service:    ref.Service,
			Name:       ref.Name,
			Kind:       ref.Kind,
			DurationMs: durationMs(span.span),
			Status:     statusToString(span.span.GetStatus()),
		},
	}
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

func appendLogicalPath(parent string, key spanLogicalKey, ordinal int) string {
	var b strings.Builder
	if parent != "" {
		b.WriteString(parent)
		b.WriteByte('/')
	}
	writeLogicalPathPart(&b, key.service)
	writeLogicalPathPart(&b, key.name)
	writeLogicalPathPart(&b, key.kind)
	b.WriteByte('#')
	b.WriteString(strconv.Itoa(ordinal))
	return b.String()
}

func writeLogicalPathPart(b *strings.Builder, value string) {
	b.WriteString(strconv.Itoa(len(value)))
	b.WriteByte(':')
	b.WriteString(value)
	b.WriteByte('|')
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

func pathKey(path []int) string {
	if len(path) == 0 {
		return ""
	}
	var b strings.Builder
	for i, p := range path {
		if i > 0 {
			b.WriteByte('/')
		}
		b.WriteString(strconv.Itoa(p))
	}
	return b.String()
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

func durationMs(span *tracev1.Span) int64 {
	if span.GetEndTimeUnixNano() < span.GetStartTimeUnixNano() {
		return 0
	}
	return int64((span.GetEndTimeUnixNano() - span.GetStartTimeUnixNano()) / 1_000_000)
}

func statusToString(status *tracev1.Status) string {
	if status == nil {
		return modeltrace.StatusToString(tracev1.Status_STATUS_CODE_UNSET)
	}
	return modeltrace.StatusToString(status.GetCode())
}
