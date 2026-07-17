package traceql

import (
	"errors"
	"fmt"
)

var (
	errParseFilter               = errors.New("failed to parse TraceQL spanset filter")
	errUnsupportedQuery          = errors.New("only a single TraceQL filter is supported")
	errUnsupportedIntrinsic      = errors.New("intrinsic is not supported by this filter")
	errUnsupportedAttributeScope = errors.New("attribute scope is not supported by this filter")
)

// CompileSpansetFilter parses a single TraceQL spanset filter (the `{ ... }` of a query) for use
// with MatchSpans. Pipelines, structural operators, and metrics are rejected: MatchSpans is per-span
// and lacks the whole-trace machinery they need.
func CompileSpansetFilter(query string) (*SpansetFilter, error) {
	expr, err := Parse(query)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errParseFilter, err)
	}
	// validate before rewriteNotExists, like a normal search, so bad regexes and intrinsic=nil are a 400.
	if err := Validate(expr); err != nil {
		return nil, fmt.Errorf("%w: %w", errParseFilter, err)
	}
	filter, err := checkIfSupported(expr)
	if err != nil {
		return nil, err
	}
	filter.Expression = rewriteNotExists(filter.Expression)
	return filter, nil
}

// rewriteNotExists replaces `attr = nil` (which the parser lowers to OpNotExists) with `!(attr exists)`
// so it evaluates correctly on this filter's true-nil statics.
// TODO: OpNotExists evaluate pairs with vparquet materializing missing attrs as StaticString("nil") - fix both layers together, then drop this rewrite.
func rewriteNotExists(e FieldExpression) FieldExpression {
	switch x := e.(type) {
	case *BinaryOperation:
		x.LHS = rewriteNotExists(x.LHS)
		x.RHS = rewriteNotExists(x.RHS)
		return x
	case UnaryOperation:
		if x.Op == OpNotExists {
			return UnaryOperation{Op: OpNot, Expression: UnaryOperation{Op: OpExists, Expression: x.Expression}}
		}
		x.Expression = rewriteNotExists(x.Expression)
		return x
	default:
		return e
	}
}

// matchSpansSupportedIntrinsics is the set the proto-span adapter resolves per span/event/link/scope.
// Trace-level intrinsics (trace:id/rootName/rootService/duration) are excluded on purpose: they need a
// whole-trace pass the per-span filter does not do, so they are rejected at compile rather than
// silently matching nothing. NeedsFullTrace gates structural elements, not intrinsics, so this
// per-intrinsic allowlist is still required.
var matchSpansSupportedIntrinsics = map[Intrinsic]struct{}{
	IntrinsicDuration:               {},
	IntrinsicName:                   {},
	IntrinsicStatus:                 {},
	IntrinsicStatusMessage:          {},
	IntrinsicKind:                   {},
	IntrinsicSpanID:                 {},
	IntrinsicParentID:               {},
	IntrinsicEventName:              {},
	IntrinsicEventTimeSinceStart:    {},
	IntrinsicLinkSpanID:             {},
	IntrinsicLinkTraceID:            {},
	IntrinsicInstrumentationName:    {},
	IntrinsicInstrumentationVersion: {},
}

// MatchSpansSupportedIntrinsics returns the intrinsics CompileSpansetFilter accepts, so adapter tests
// can assert they resolve every one of them and catch drift before the adapter's unreachable panic.
func MatchSpansSupportedIntrinsics() []Intrinsic {
	out := make([]Intrinsic, 0, len(matchSpansSupportedIntrinsics))
	for ic := range matchSpansSupportedIntrinsics {
		out = append(out, ic)
	}
	return out
}

// checkIfSupported returns q's sole spanset filter when q is a single { ... } whose intrinsics the
// proto-span adapter can resolve, else errUnsupportedQuery or errUnsupportedIntrinsic.
func checkIfSupported(expr *RootExpr) (*SpansetFilter, error) {
	pipeline, ok := expr.SinglePipeline()
	if !ok {
		return nil, fmt.Errorf("%w: got a multi-stage pipeline or metrics query", errUnsupportedQuery)
	}

	// SinglePipeline allows one series/batch processor (e.g. rate()), so reject those explicitly.
	if len(expr.SeriesProcessor) > 0 || len(expr.BatchSpanProcessor) > 0 {
		return nil, fmt.Errorf("%w: got a metrics query", errUnsupportedQuery)
	}

	// NeedsFullTrace flags structural operators and aggregates ({} >> {}, {} | count()). It does not
	// inspect intrinsics inside the filter, so the allowlist below is still needed.
	if NeedsFullTrace(pipeline) {
		return nil, fmt.Errorf("%w: got a structural or aggregate query", errUnsupportedQuery)
	}

	if len(pipeline.Elements) != 1 {
		return nil, fmt.Errorf("%w: got %d pipeline elements", errUnsupportedQuery, len(pipeline.Elements))
	}

	filter, ok := pipeline.Elements[0].(*SpansetFilter)
	if !ok {
		return nil, fmt.Errorf("%w: got %T", errUnsupportedQuery, pipeline.Elements[0])
	}

	// reject anything the proto-span adapter can't resolve, turning a silent no-match into a visible error:
	// 1. trace-level intrinsics
	// 2. trace-scoped attributes.
	var req FetchSpansRequest
	filter.Expression.extractConditions(&req)
	for _, c := range req.Conditions {
		if c.Attribute.Intrinsic != IntrinsicNone {
			if _, ok := matchSpansSupportedIntrinsics[c.Attribute.Intrinsic]; !ok {
				return nil, fmt.Errorf("%w: %q", errUnsupportedIntrinsic, c.Attribute.Intrinsic)
			}
			continue
		}
		switch c.Attribute.Scope {
		case AttributeScopeNone, AttributeScopeSpan, AttributeScopeResource,
			AttributeScopeEvent, AttributeScopeLink, AttributeScopeInstrumentation:
		default:
			return nil, fmt.Errorf("%w: %q", errUnsupportedAttributeScope, c.Attribute)
		}
	}

	return filter, nil
}

// ReferencesEventOrLink reports whether the filter reads any event: or link: scoped attribute or
// intrinsic. When it doesn't, the caller can skip the per-span event x link expansion.
func (f *SpansetFilter) ReferencesEventOrLink() bool {
	var req FetchSpansRequest
	f.Expression.extractConditions(&req)
	for _, c := range req.Conditions {
		switch c.Attribute.Scope {
		case AttributeScopeEvent, AttributeScopeLink:
			return true
		}
		switch c.Attribute.Intrinsic {
		case IntrinsicEventName, IntrinsicEventTimeSinceStart, IntrinsicLinkTraceID, IntrinsicLinkSpanID:
			return true
		}
	}
	return false
}

// MatchSpans returns the subset of spans matching the filter, reusing the engine's evaluate() so
// semantics match a normal TraceQL search.
func (f *SpansetFilter) MatchSpans(spans []Span) ([]Span, error) {
	if len(spans) == 0 {
		return nil, nil
	}

	output, err := f.evaluate([]*Spanset{{Spans: spans}})
	if err != nil {
		return nil, err
	}

	// a spanset filter never splits one input spanset, so output is 0 or 1.
	if len(output) == 0 {
		return nil, nil
	}
	return output[0].Spans, nil
}
