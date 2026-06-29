package tracefilter

import (
	"bytes"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	resourcev1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
)

const (
	// QueryParam carries the single TraceQL spanset filter.
	QueryParam = "q"
	// KeepHierarchyParam also returns each matched span's ancestor path to the root.
	KeepHierarchyParam = "keep_hierarchy"
)

// Options holds the parsed filtering options from a request.
type Options struct {
	// Query is a single TraceQL spanset filter; empty means no filtering.
	Query string
	// KeepHierarchy includes each matched span's ancestor path; ignored when Query is empty.
	KeepHierarchy bool
}

// OptionsFromValues parses filtering options from URL query parameters; it does not validate the
// TraceQL itself (call Compile for that).
func OptionsFromValues(vals url.Values) (Options, error) {
	// keep_hierarchy defaults to true so a filtered trace stays rooted/renderable unless explicitly disabled.
	opts := Options{
		Query:         vals.Get(QueryParam),
		KeepHierarchy: true,
	}

	// keep_hierarchy is ignored without a query, so don't parse/reject it then.
	if opts.Query == "" {
		return opts, nil
	}

	if raw := vals.Get(KeepHierarchyParam); raw != "" {
		keep, err := strconv.ParseBool(raw)
		if err != nil {
			return Options{}, fmt.Errorf("invalid value for %s: %w", KeepHierarchyParam, err)
		}
		opts.KeepHierarchy = keep
	}

	return opts, nil
}

// Filter is a compiled, ready-to-apply trace filter.
type Filter struct {
	spansetFilter *traceql.SpansetFilter
	keepHierarchy bool
}

// NewFilterFromValues parses and compiles filtering options in one step; returns (nil, nil) when no
// filtering is requested. Errors are caller's to map to a 400.
func NewFilterFromValues(vals url.Values) (*Filter, error) {
	opts, err := OptionsFromValues(vals)
	if err != nil {
		return nil, err
	}
	return opts.Compile()
}

// Compile compiles the options into a Filter; returns (nil, nil) for an empty Query (passthrough).
func (o Options) Compile() (*Filter, error) {
	if o.Query == "" {
		return nil, nil
	}

	sf, err := traceql.CompileSpansetFilter(o.Query)
	if err != nil {
		return nil, fmt.Errorf("invalid %s filter: %w", QueryParam, err)
	}

	return &Filter{
		spansetFilter: sf,
		keepHierarchy: o.KeepHierarchy,
	}, nil
}

// Process returns a new trace with only the kept spans; it never mutates the input, which may be cached.
func (f *Filter) Process(trace *tempopb.Trace) (*tempopb.Trace, error) {
	if f == nil || trace == nil {
		return trace, nil
	}

	idx := newSpanIndex(trace)

	matched, err := f.spansetFilter.MatchSpans(idx.spans)
	if err != nil {
		return nil, err
	}

	kept := make(map[string]struct{}, len(matched))
	for _, s := range matched {
		kept[string(s.ID())] = struct{}{}
	}

	if f.keepHierarchy {
		f.addAncestors(idx, kept)
	}

	return rebuildTrace(trace, kept), nil
}

// addAncestors adds every matched span's ancestor path to kept. Walking is a BFS because duplicate
// span ids may have multiple parents; kept doubles as the visited set so cycles terminate.
func (f *Filter) addAncestors(idx *spanIndex, kept map[string]struct{}) {
	// snapshot the starting ids: we mutate kept while walking.
	queue := make([]string, 0, len(kept))
	for id := range kept {
		queue = append(queue, id)
	}

	for len(queue) > 0 {
		current := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		for _, parentID := range idx.parentsByID[current] {
			if parentID == "" {
				continue // reached a root.
			}
			if _, seen := kept[parentID]; seen {
				continue // already kept; also breaks cycles.
			}
			// a dangling parentID is harmless: rebuildTrace only emits spans that exist.
			kept[parentID] = struct{}{}
			queue = append(queue, parentID)
		}
	}
}

// spanIndex is a flattened view of a trace for matching and ancestor walks.
type spanIndex struct {
	spans       []traceql.Span      // proto-backed spans for the engine.
	parentsByID map[string][]string // span id -> distinct parent span ids of all spans sharing that id.
}

func newSpanIndex(trace *tempopb.Trace) *spanIndex {
	idx := &spanIndex{
		parentsByID: make(map[string][]string),
	}
	traceAttrs := traceAttributes(trace)
	for _, rs := range trace.ResourceSpans {
		resourceAttrs := resourceAttributes(rs.Resource)
		for _, ss := range rs.ScopeSpans {
			for _, span := range ss.Spans {
				idx.spans = append(idx.spans, newProtoSpan(span, resourceAttrs, traceAttrs, ss.Scope))
				idx.addParent(string(span.SpanId), string(span.ParentSpanId))
			}
		}
	}
	return idx
}

// addParent records a distinct parent id for a span id; duplicate span ids accumulate every parent
// so the ancestor walk follows all branches instead of an arbitrary last-writer one.
func (idx *spanIndex) addParent(spanID, parentID string) {
	if slices.Contains(idx.parentsByID[spanID], parentID) {
		return
	}
	idx.parentsByID[spanID] = append(idx.parentsByID[spanID], parentID)
}

// traceAttributes computes the trace-level intrinsics (trace:rootName, trace:rootService,
// trace:duration) once so every span in the trace resolves them identically, matching TraceQL.
func traceAttributes(trace *tempopb.Trace) map[traceql.Attribute]traceql.Static {
	var (
		traceStart, traceEnd uint64
		rootSpan             *tracev1.Span
		rootResource         *resourcev1.Resource
	)
	for _, rs := range trace.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			for _, span := range ss.Spans {
				if traceStart == 0 || span.StartTimeUnixNano < traceStart {
					traceStart = span.StartTimeUnixNano
				}
				if span.EndTimeUnixNano > traceEnd {
					traceEnd = span.EndTimeUnixNano
				}
				// mirror storage's root definition: parentless and no intra-trace child_of link.
				if len(span.ParentSpanId) == 0 && !hasChildOfLink(span) {
					rootSpan = span
					rootResource = rs.Resource
				}
			}
		}
	}

	attrs := make(map[traceql.Attribute]traceql.Static, 3)
	duration := uint64(0)
	if traceEnd > traceStart {
		duration = traceEnd - traceStart
	}
	attrs[traceql.IntrinsicTraceDurationAttribute] = traceql.NewStaticDuration(time.Duration(duration))
	// set both root intrinsics together so a rootless trace resolves them identically, as storage does.
	if rootSpan != nil {
		attrs[traceql.IntrinsicTraceRootSpanAttribute] = traceql.NewStaticString(rootSpan.Name)
		attrs[traceql.IntrinsicTraceRootServiceAttribute] = traceql.NewStaticString(rootServiceName(rootResource))
	} else {
		attrs[traceql.IntrinsicTraceRootSpanAttribute] = traceql.NewStaticString("")
		attrs[traceql.IntrinsicTraceRootServiceAttribute] = traceql.NewStaticString("")
	}
	return attrs
}

// hasChildOfLink reports whether the span has an intra-trace OpenTracing child_of link, which
// storage treats as disqualifying a parentless span from being the trace root.
func hasChildOfLink(span *tracev1.Span) bool {
	for _, link := range span.Links {
		if !bytes.Equal(span.TraceId, link.TraceId) {
			continue
		}
		for _, attr := range link.GetAttributes() {
			if attr.Key == "opentracing.ref_type" && attr.GetValue().GetStringValue() == "child_of" {
				return true
			}
		}
	}
	return false
}

// rootServiceName extracts service.name from the root span's resource; empty when absent.
func rootServiceName(resource *resourcev1.Resource) string {
	if resource == nil {
		return ""
	}
	for _, kv := range resource.Attributes {
		if kv.Key == "service.name" {
			return kv.Value.GetStringValue()
		}
	}
	return ""
}

// rebuildTrace returns a new trace of only the kept spans, preserving grouping and dropping empties.
func rebuildTrace(trace *tempopb.Trace, kept map[string]struct{}) *tempopb.Trace {
	out := &tempopb.Trace{}

	for _, rs := range trace.ResourceSpans {
		var keptScopes []*tracev1.ScopeSpans
		for _, ss := range rs.ScopeSpans {
			var keptSpans []*tracev1.Span
			for _, span := range ss.Spans {
				if _, ok := kept[string(span.SpanId)]; ok {
					keptSpans = append(keptSpans, span)
				}
			}
			if len(keptSpans) == 0 {
				continue
			}
			keptScopes = append(keptScopes, &tracev1.ScopeSpans{
				Scope:     ss.Scope,
				SchemaUrl: ss.SchemaUrl,
				Spans:     keptSpans,
			})
		}
		if len(keptScopes) == 0 {
			continue
		}
		out.ResourceSpans = append(out.ResourceSpans, &tracev1.ResourceSpans{
			Resource:   rs.Resource,
			SchemaUrl:  rs.SchemaUrl,
			ScopeSpans: keptScopes,
		})
	}

	return out
}
