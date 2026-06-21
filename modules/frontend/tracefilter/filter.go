// Package tracefilter shapes an assembled trace-by-id response via the `q` spanset filter,
// `keep_hierarchy`, and `match_depth` options.
package tracefilter

import (
	"fmt"
	"net/url"
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
	// MatchDepthParam controls how many levels of descendants to include for each matched span.
	// -1 (default) keeps all descendants. 0 keeps only the matched span. N keeps N levels down.
	MatchDepthParam = "match_depth"
)

// Options holds the parsed filtering options from a request.
type Options struct {
	// Query is a single TraceQL spanset filter; empty means no filtering.
	Query string
	// KeepHierarchy includes each matched span's ancestor path; ignored when Query is empty.
	KeepHierarchy bool
	// MatchDepth controls how many descendant levels to keep for each matched span.
	// -1 = all descendants, 0 = matched span only, N = N levels deep.
	MatchDepth int
}

// OptionsFromValues parses filtering options from URL query parameters; it does not validate the
// TraceQL itself (call Compile for that).
func OptionsFromValues(vals url.Values) (Options, error) {
	// keep_hierarchy defaults to true so a filtered trace stays rooted/renderable unless explicitly disabled.
	// match_depth defaults to -1 (all descendants).
	opts := Options{
		Query:         vals.Get(QueryParam),
		KeepHierarchy: true,
		MatchDepth:    -1,
	}

	if raw := vals.Get(KeepHierarchyParam); raw != "" {
		keep, err := strconv.ParseBool(raw)
		if err != nil {
			return Options{}, fmt.Errorf("invalid value for %s: %w", KeepHierarchyParam, err)
		}
		opts.KeepHierarchy = keep
	}

	if raw := vals.Get(MatchDepthParam); raw != "" {
		depth, err := strconv.Atoi(raw)
		if err != nil {
			return Options{}, fmt.Errorf("invalid value for %s: %w", MatchDepthParam, err)
		}
		if depth < -1 {
			return Options{}, fmt.Errorf("%s must be >= -1 (-1 means all descendants)", MatchDepthParam)
		}
		opts.MatchDepth = depth
	}

	return opts, nil
}

// Filter is a compiled, ready-to-apply trace filter.
type Filter struct {
	spansetFilter *traceql.SpansetFilter
	keepHierarchy bool
	matchDepth    int
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
		matchDepth:    o.MatchDepth,
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

	// Ancestors and descendants are both rooted in the originally matched spans,
	// not in each other, so the order between these two calls does not matter.
	if f.keepHierarchy {
		f.addAncestors(idx, kept)
	}

	if f.matchDepth != 0 {
		f.addDescendants(idx, kept, matched)
	}

	return rebuildTrace(trace, kept), nil
}

// addAncestors adds each matched span's parent chain to kept.
func (f *Filter) addAncestors(idx *spanIndex, kept map[string]struct{}) {
	// snapshot the ids first: we mutate kept while iterating.
	matchedIDs := make([]string, 0, len(kept))
	for id := range kept {
		matchedIDs = append(matchedIDs, id)
	}

	for _, id := range matchedIDs {
		current := id
		for {
			parentID, ok := idx.parentOf[current]
			if !ok || parentID == "" {
				break // reached a root.
			}
			if _, seen := kept[parentID]; seen {
				break // chain already kept; also breaks cycles.
			}
			// a dangling parentID is harmless: rebuildTrace only emits spans that exist.
			kept[parentID] = struct{}{}
			current = parentID
		}
	}
}

// addDescendants adds descendants of the originally matched spans up to f.matchDepth levels.
// -1 means unlimited. Descendants of ancestors added by addAncestors are not included.
func (f *Filter) addDescendants(idx *spanIndex, kept map[string]struct{}, matched []traceql.Span) {
	type entry struct {
		id    string
		depth int // 0 = the matched span itself; 1 = its direct children, etc.
	}

	queue := make([]entry, 0, len(matched))
	for _, s := range matched {
		queue = append(queue, entry{string(s.ID()), 0})
	}

	visited := make(map[string]struct{}, len(matched))

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if _, seen := visited[curr.id]; seen {
			continue
		}
		visited[curr.id] = struct{}{}

		// Stop recursing once we've reached the depth limit.
		// matchDepth == -1 means unlimited, so the condition is never true.
		if f.matchDepth >= 0 && curr.depth >= f.matchDepth {
			continue
		}

		for _, childID := range idx.childrenOf[curr.id] {
			kept[childID] = struct{}{}
			queue = append(queue, entry{childID, curr.depth + 1})
		}
	}
}

// spanIndex is a flattened view of a trace for matching and ancestor/descendant walks.
type spanIndex struct {
	spans      []traceql.Span      // proto-backed spans for the engine.
	parentOf   map[string]string   // span id -> parent span id (both string(bytes)).
	childrenOf map[string][]string // span id -> child span ids.
}

func newSpanIndex(trace *tempopb.Trace) *spanIndex {
	idx := &spanIndex{
		parentOf:   make(map[string]string),
		childrenOf: make(map[string][]string),
	}
	traceAttrs := traceAttributes(trace)
	for _, rs := range trace.ResourceSpans {
		resourceAttrs := resourceAttributes(rs.Resource)
		for _, ss := range rs.ScopeSpans {
			for _, span := range ss.Spans {
				idx.spans = append(idx.spans, newProtoSpan(span, resourceAttrs, traceAttrs, ss.Scope))
				// FIXME: duplicate span ids collapse here (last writer wins); the ancestor walk assumes unique ids.
				spanID := string(span.SpanId)
				parentID := string(span.ParentSpanId)
				idx.parentOf[spanID] = parentID
				if parentID != "" {
					idx.childrenOf[parentID] = append(idx.childrenOf[parentID], spanID)
				}
			}
		}
	}
	return idx
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
				// a span with no parent is the trace root, matching the storage layer's definition.
				if len(span.ParentSpanId) == 0 {
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
	if rootSpan != nil {
		attrs[traceql.IntrinsicTraceRootSpanAttribute] = traceql.NewStaticString(rootSpan.Name)
	}
	attrs[traceql.IntrinsicTraceRootServiceAttribute] = traceql.NewStaticString(rootServiceName(rootResource))
	return attrs
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
