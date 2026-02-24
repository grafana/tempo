package livestore

import (
	"bytes"
	"time"

	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resource_v1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	trace_v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
)

// protoSpan implements traceql.Span over raw OTLP proto data.
type protoSpan struct {
	span     *trace_v1.Span
	resource *resource_v1.Resource
	scope    *v1.InstrumentationScope

	// Lazily computed nested set model for structural queries.
	// These are set by buildNestedSet before structural queries are used.
	nestedSetParent int32
	nestedSetLeft   int32
	nestedSetRight  int32
}

var _ traceql.Span = (*protoSpan)(nil)

func (s *protoSpan) ID() []byte {
	return s.span.SpanId
}

func (s *protoSpan) StartTimeUnixNanos() uint64 {
	return s.span.StartTimeUnixNano
}

func (s *protoSpan) DurationNanos() uint64 {
	if s.span.EndTimeUnixNano > s.span.StartTimeUnixNano {
		return s.span.EndTimeUnixNano - s.span.StartTimeUnixNano
	}
	return 0
}

func (s *protoSpan) AttributeFor(a traceql.Attribute) (traceql.Static, bool) {
	// Handle intrinsics first
	if a.Intrinsic != traceql.IntrinsicNone {
		return s.intrinsicFor(a.Intrinsic)
	}

	// Scoped attribute lookup
	switch a.Scope {
	case traceql.AttributeScopeSpan:
		return findInKeyValues(s.span.Attributes, a.Name)
	case traceql.AttributeScopeResource:
		if s.resource != nil {
			return findInKeyValues(s.resource.Attributes, a.Name)
		}
		return traceql.NewStaticNil(), false
	case traceql.AttributeScopeEvent:
		return s.findInEvents(a.Name)
	case traceql.AttributeScopeLink:
		return s.findInLinks(a.Name)
	case traceql.AttributeScopeInstrumentation:
		if s.scope != nil {
			return findInKeyValues(s.scope.Attributes, a.Name)
		}
		return traceql.NewStaticNil(), false
	case traceql.AttributeScopeNone:
		// Unscoped: check span first, then resource, then event, link, instrumentation
		if v, ok := findInKeyValues(s.span.Attributes, a.Name); ok {
			return v, true
		}
		if s.resource != nil {
			if v, ok := findInKeyValues(s.resource.Attributes, a.Name); ok {
				return v, true
			}
		}
		if v, ok := s.findInEvents(a.Name); ok {
			return v, true
		}
		if v, ok := s.findInLinks(a.Name); ok {
			return v, true
		}
		if s.scope != nil {
			if v, ok := findInKeyValues(s.scope.Attributes, a.Name); ok {
				return v, true
			}
		}
	}

	return traceql.NewStaticNil(), false
}

func (s *protoSpan) intrinsicFor(i traceql.Intrinsic) (traceql.Static, bool) {
	switch i {
	case traceql.IntrinsicName:
		return traceql.NewStaticString(s.span.Name), true
	case traceql.IntrinsicDuration:
		return traceql.NewStaticDuration(time.Duration(s.DurationNanos())), true
	case traceql.IntrinsicStatus:
		return traceql.NewStaticStatus(protoStatusToTraceQL(s.span.Status)), true
	case traceql.IntrinsicStatusMessage:
		if s.span.Status != nil {
			return traceql.NewStaticString(s.span.Status.Message), true
		}
		return traceql.NewStaticString(""), true
	case traceql.IntrinsicKind:
		return traceql.NewStaticKind(protoKindToTraceQL(s.span.Kind)), true
	case traceql.IntrinsicSpanID:
		return traceql.NewStaticString(string(s.span.SpanId)), true
	case traceql.IntrinsicSpanStartTime:
		return traceql.NewStaticInt(int(s.span.StartTimeUnixNano)), true
	case traceql.IntrinsicNestedSetLeft:
		return traceql.NewStaticInt(int(s.nestedSetLeft)), true
	case traceql.IntrinsicNestedSetRight:
		return traceql.NewStaticInt(int(s.nestedSetRight)), true
	case traceql.IntrinsicNestedSetParent:
		return traceql.NewStaticInt(int(s.nestedSetParent)), true
	}

	return traceql.NewStaticNil(), false
}

func (s *protoSpan) AllAttributes() map[traceql.Attribute]traceql.Static {
	atts := make(map[traceql.Attribute]traceql.Static)
	s.AllAttributesFunc(func(a traceql.Attribute, v traceql.Static) {
		atts[a] = v
	})
	return atts
}

func (s *protoSpan) AllAttributesFunc(cb func(traceql.Attribute, traceql.Static)) {
	// Resource attributes
	if s.resource != nil {
		for _, kv := range s.resource.Attributes {
			if v, ok := anyValueToStatic(kv.Value); ok {
				cb(traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, kv.Key), v)
			}
		}
	}

	// Span attributes
	for _, kv := range s.span.Attributes {
		if v, ok := anyValueToStatic(kv.Value); ok {
			cb(traceql.NewScopedAttribute(traceql.AttributeScopeSpan, false, kv.Key), v)
		}
	}

	// Event attributes
	for _, event := range s.span.Events {
		for _, kv := range event.Attributes {
			if v, ok := anyValueToStatic(kv.Value); ok {
				cb(traceql.NewScopedAttribute(traceql.AttributeScopeEvent, false, kv.Key), v)
			}
		}
	}

	// Link attributes
	for _, link := range s.span.Links {
		for _, kv := range link.Attributes {
			if v, ok := anyValueToStatic(kv.Value); ok {
				cb(traceql.NewScopedAttribute(traceql.AttributeScopeLink, false, kv.Key), v)
			}
		}
	}

	// Instrumentation scope attributes
	if s.scope != nil {
		for _, kv := range s.scope.Attributes {
			if v, ok := anyValueToStatic(kv.Value); ok {
				cb(traceql.NewScopedAttribute(traceql.AttributeScopeInstrumentation, false, kv.Key), v)
			}
		}
	}

	// Intrinsics
	cb(traceql.NewIntrinsic(traceql.IntrinsicName), traceql.NewStaticString(s.span.Name))
	cb(traceql.NewIntrinsic(traceql.IntrinsicDuration), traceql.NewStaticDuration(time.Duration(s.DurationNanos())))
	cb(traceql.NewIntrinsic(traceql.IntrinsicStatus), traceql.NewStaticStatus(protoStatusToTraceQL(s.span.Status)))
	cb(traceql.NewIntrinsic(traceql.IntrinsicKind), traceql.NewStaticKind(protoKindToTraceQL(s.span.Kind)))
}

func (s *protoSpan) findInEvents(name string) (traceql.Static, bool) {
	if name == "name" {
		for _, event := range s.span.Events {
			if event.Name != "" {
				return traceql.NewStaticString(event.Name), true
			}
		}
		return traceql.NewStaticNil(), false
	}
	for _, event := range s.span.Events {
		if v, ok := findInKeyValues(event.Attributes, name); ok {
			return v, true
		}
	}
	return traceql.NewStaticNil(), false
}

func (s *protoSpan) findInLinks(name string) (traceql.Static, bool) {
	for _, link := range s.span.Links {
		if v, ok := findInKeyValues(link.Attributes, name); ok {
			return v, true
		}
	}
	return traceql.NewStaticNil(), false
}

// Structural relationship implementations.
// These use the nested set model computed by buildNestedSet.

func (s *protoSpan) SiblingOf(lhs, rhs []traceql.Span, falseForAll, union bool, buffer []traceql.Span) []traceql.Span {
	buffer = buffer[:0]

	for _, r := range rhs {
		rSpan := r.(*protoSpan)
		found := false
		for _, l := range lhs {
			lSpan := l.(*protoSpan)
			if lSpan.nestedSetParent == rSpan.nestedSetParent && lSpan.nestedSetParent != 0 && lSpan != rSpan {
				found = true
				break
			}
		}
		if found != falseForAll {
			buffer = append(buffer, r)
		}
	}

	if union {
		// Add lhs spans that have siblings in rhs
		for _, l := range lhs {
			lSpan := l.(*protoSpan)
			found := false
			for _, r := range rhs {
				rSpan := r.(*protoSpan)
				if lSpan.nestedSetParent == rSpan.nestedSetParent && lSpan.nestedSetParent != 0 && lSpan != rSpan {
					found = true
					break
				}
			}
			if found != falseForAll {
				buffer = append(buffer, l)
			}
		}
	}

	return buffer
}

func (s *protoSpan) DescendantOf(lhs, rhs []traceql.Span, falseForAll, invert, union bool, buffer []traceql.Span) []traceql.Span {
	buffer = buffer[:0]

	isDescendant := func(child, ancestor *protoSpan) bool {
		return child.nestedSetLeft > ancestor.nestedSetLeft && child.nestedSetRight < ancestor.nestedSetRight
	}

	for _, r := range rhs {
		rSpan := r.(*protoSpan)
		found := false
		for _, l := range lhs {
			lSpan := l.(*protoSpan)
			if invert {
				// AncestorOf: rhs is ancestor of lhs
				found = isDescendant(lSpan, rSpan)
			} else {
				// DescendantOf: rhs is descendant of lhs
				found = isDescendant(rSpan, lSpan)
			}
			if found {
				break
			}
		}
		if found != falseForAll {
			buffer = append(buffer, r)
		}
	}

	if union {
		for _, l := range lhs {
			lSpan := l.(*protoSpan)
			found := false
			for _, r := range rhs {
				rSpan := r.(*protoSpan)
				if invert {
					found = isDescendant(rSpan, lSpan)
				} else {
					found = isDescendant(lSpan, rSpan)
				}
				if found {
					break
				}
			}
			if found != falseForAll {
				buffer = append(buffer, l)
			}
		}
	}

	return buffer
}

func (s *protoSpan) ChildOf(lhs, rhs []traceql.Span, falseForAll, invert, union bool, buffer []traceql.Span) []traceql.Span {
	buffer = buffer[:0]

	isChild := func(child, parent *protoSpan) bool {
		return child.nestedSetParent == parent.nestedSetLeft
	}

	for _, r := range rhs {
		rSpan := r.(*protoSpan)
		found := false
		for _, l := range lhs {
			lSpan := l.(*protoSpan)
			if invert {
				// ParentOf: rhs is parent of lhs
				found = isChild(lSpan, rSpan)
			} else {
				// ChildOf: rhs is child of lhs
				found = isChild(rSpan, lSpan)
			}
			if found {
				break
			}
		}
		if found != falseForAll {
			buffer = append(buffer, r)
		}
	}

	if union {
		for _, l := range lhs {
			lSpan := l.(*protoSpan)
			found := false
			for _, r := range rhs {
				rSpan := r.(*protoSpan)
				if invert {
					found = isChild(rSpan, lSpan)
				} else {
					found = isChild(lSpan, rSpan)
				}
				if found {
					break
				}
			}
			if found != falseForAll {
				buffer = append(buffer, l)
			}
		}
	}

	return buffer
}

// findInKeyValues searches a slice of KeyValue pairs for the given key.
func findInKeyValues(kvs []*v1.KeyValue, key string) (traceql.Static, bool) {
	for _, kv := range kvs {
		if kv.Key == key {
			return anyValueToStatic(kv.Value)
		}
	}
	return traceql.NewStaticNil(), false
}

// anyValueToStatic converts an OTLP AnyValue to a traceql.Static.
func anyValueToStatic(v *v1.AnyValue) (traceql.Static, bool) {
	if v == nil {
		return traceql.NewStaticNil(), false
	}
	switch val := v.Value.(type) {
	case *v1.AnyValue_StringValue:
		return traceql.NewStaticString(val.StringValue), true
	case *v1.AnyValue_IntValue:
		return traceql.NewStaticInt(int(val.IntValue)), true
	case *v1.AnyValue_DoubleValue:
		return traceql.NewStaticFloat(val.DoubleValue), true
	case *v1.AnyValue_BoolValue:
		return traceql.NewStaticBool(val.BoolValue), true
	default:
		return traceql.NewStaticNil(), false
	}
}

func protoStatusToTraceQL(s *trace_v1.Status) traceql.Status {
	if s == nil {
		return traceql.StatusUnset
	}
	switch s.Code {
	case trace_v1.Status_STATUS_CODE_OK:
		return traceql.StatusOk
	case trace_v1.Status_STATUS_CODE_ERROR:
		return traceql.StatusError
	default:
		return traceql.StatusUnset
	}
}

func protoKindToTraceQL(k trace_v1.Span_SpanKind) traceql.Kind {
	switch k {
	case trace_v1.Span_SPAN_KIND_INTERNAL:
		return traceql.KindInternal
	case trace_v1.Span_SPAN_KIND_SERVER:
		return traceql.KindServer
	case trace_v1.Span_SPAN_KIND_CLIENT:
		return traceql.KindClient
	case trace_v1.Span_SPAN_KIND_PRODUCER:
		return traceql.KindProducer
	case trace_v1.Span_SPAN_KIND_CONSUMER:
		return traceql.KindConsumer
	default:
		return traceql.KindUnspecified
	}
}

// buildNestedSet computes the nested set model for a list of protoSpans
// within a single trace, using ParentSpanId to construct the tree.
func buildNestedSet(spans []*protoSpan) {
	if len(spans) == 0 {
		return
	}

	// Build span ID -> index map
	spanByID := make(map[string]int, len(spans))
	children := make(map[int][]int, len(spans))
	var roots []int

	for i, s := range spans {
		spanByID[string(s.span.SpanId)] = i
	}

	for i, s := range spans {
		parentID := s.span.ParentSpanId
		if len(parentID) == 0 {
			roots = append(roots, i)
			continue
		}
		if parentIdx, ok := spanByID[string(parentID)]; ok {
			children[parentIdx] = append(children[parentIdx], i)
		} else {
			// Orphan span (parent not in this trace) - treat as root
			roots = append(roots, i)
		}
	}

	// DFS to assign nested set numbers
	counter := int32(1)
	var dfs func(idx int)
	dfs = func(idx int) {
		spans[idx].nestedSetLeft = counter
		counter++

		for _, childIdx := range children[idx] {
			spans[childIdx].nestedSetParent = spans[idx].nestedSetLeft
			dfs(childIdx)
		}

		spans[idx].nestedSetRight = counter
		counter++
	}

	for _, rootIdx := range roots {
		dfs(rootIdx)
	}
}

// needsNestedSet returns true if any condition in the request requires
// structural relationship data (SiblingOf, DescendantOf, ChildOf).
func needsNestedSet(req traceql.FetchSpansRequest) bool {
	for _, c := range req.Conditions {
		switch c.Attribute.Intrinsic {
		case traceql.IntrinsicNestedSetLeft, traceql.IntrinsicNestedSetRight, traceql.IntrinsicNestedSetParent:
			return true
		}
	}
	for _, c := range req.SecondPassConditions {
		switch c.Attribute.Intrinsic {
		case traceql.IntrinsicNestedSetLeft, traceql.IntrinsicNestedSetRight, traceql.IntrinsicNestedSetParent:
			return true
		}
	}
	return false
}

// extractSpansFromTrace returns all spans in the trace with their resource/scope context.
func extractSpansFromTrace(batches []*trace_v1.ResourceSpans) []*protoSpan {
	var spans []*protoSpan
	for _, rs := range batches {
		for _, ss := range rs.ScopeSpans {
			for _, span := range ss.Spans {
				spans = append(spans, &protoSpan{
					span:     span,
					resource: rs.Resource,
					scope:    ss.Scope,
				})
			}
		}
	}
	return spans
}

// traceIDFromSpans extracts the trace ID from spans. All spans in a trace share the same trace ID.
func traceIDFromSpans(batches []*trace_v1.ResourceSpans) []byte {
	for _, rs := range batches {
		for _, ss := range rs.ScopeSpans {
			for _, span := range ss.Spans {
				if len(span.TraceId) > 0 {
					return span.TraceId
				}
			}
		}
	}
	return nil
}

// traceTimeBounds returns the min start and max end timestamps from all spans in a trace.
func traceTimeBounds(batches []*trace_v1.ResourceSpans) (start, end uint64) {
	for _, rs := range batches {
		for _, ss := range rs.ScopeSpans {
			for _, span := range ss.Spans {
				if start == 0 || span.StartTimeUnixNano < start {
					start = span.StartTimeUnixNano
				}
				if span.EndTimeUnixNano > end {
					end = span.EndTimeUnixNano
				}
			}
		}
	}
	return
}

// traceMatchesID checks if any span in the trace has the given trace ID.
func traceMatchesID(batches []*trace_v1.ResourceSpans, id []byte) bool {
	for _, rs := range batches {
		for _, ss := range rs.ScopeSpans {
			for _, span := range ss.Spans {
				if bytes.Equal(span.TraceId, id) {
					return true
				}
			}
		}
	}
	return false
}
