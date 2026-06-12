package traceql

import "fmt"

// CompileSpansetFilter parses a single TraceQL spanset filter (the `{ ... }` of a query) for use
// with MatchSpans. Pipelines, structural operators, and metrics are rejected: MatchSpans is per-span
// and lacks the whole-trace machinery they need.
func CompileSpansetFilter(query string) (*SpansetFilter, error) {
	expr, err := Parse(query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TraceQL spanset filter: %w", err)
	}

	pipeline, ok := expr.SinglePipeline()
	if !ok {
		return nil, fmt.Errorf("only a single TraceQL spanset filter is supported, not a full pipeline or metrics query")
	}

	// SinglePipeline allows one series/batch processor (e.g. rate()), so reject those explicitly.
	if len(expr.SeriesProcessor) > 0 || len(expr.BatchSpanProcessor) > 0 {
		return nil, fmt.Errorf("only a single TraceQL spanset filter is supported, not a metrics query")
	}

	if len(pipeline.Elements) != 1 {
		return nil, fmt.Errorf("only a single TraceQL spanset filter is supported, got %d pipeline elements", len(pipeline.Elements))
	}

	filter, ok := pipeline.Elements[0].(*SpansetFilter)
	if !ok {
		// e.g. structural ops like `{} >> {}` parse to a single non-SpansetFilter element.
		return nil, fmt.Errorf("only a single TraceQL spanset filter is supported, got %T", pipeline.Elements[0])
	}

	if err := rejectUnsupportedIntrinsics(filter); err != nil {
		return nil, err
	}

	return filter, nil
}

// matchSpansSupportedIntrinsics is the set MatchSpans can resolve from a single proto span plus its
// whole-trace context; anything else would silently match nothing, so it is rejected at compile time.
var matchSpansSupportedIntrinsics = map[Intrinsic]struct{}{
	IntrinsicDuration:               {},
	IntrinsicName:                   {},
	IntrinsicStatus:                 {},
	IntrinsicStatusMessage:          {},
	IntrinsicKind:                   {},
	IntrinsicSpanID:                 {},
	IntrinsicParentID:               {},
	IntrinsicTraceID:                {},
	IntrinsicTraceRootService:       {},
	IntrinsicTraceRootSpan:          {},
	IntrinsicTraceDuration:          {},
	IntrinsicEventName:              {},
	IntrinsicEventTimeSinceStart:    {},
	IntrinsicLinkSpanID:             {},
	IntrinsicLinkTraceID:            {},
	IntrinsicInstrumentationName:    {},
	IntrinsicInstrumentationVersion: {},
}

// rejectUnsupportedIntrinsics fails filters that reference intrinsics MatchSpans cannot resolve
// (e.g. span:childCount, nestedSetLeft), turning a silent no-match into a caller-visible error.
func rejectUnsupportedIntrinsics(filter *SpansetFilter) error {
	var req FetchSpansRequest
	filter.Expression.extractConditions(&req)
	for _, c := range req.Conditions {
		if c.Attribute.Intrinsic == IntrinsicNone {
			continue
		}
		if _, ok := matchSpansSupportedIntrinsics[c.Attribute.Intrinsic]; !ok {
			return fmt.Errorf("intrinsic %q is not supported by this filter", c.Attribute.Intrinsic)
		}
	}
	return nil
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
