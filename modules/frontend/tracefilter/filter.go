// Plumbing for the trace-by-id q filter: parse options, compile, build the span index, walk ancestors
// for keep_hierarchy, and rebuild the trace. Span matching itself lives in protospan.go.

package tracefilter

import (
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/grafana/tempo/pkg/tempopb"
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
	// Query is a single TraceQL spanset filter. Empty means no filtering.
	Query string
	// KeepHierarchy includes each matched span's ancestor path. Ignored when Query is empty.
	KeepHierarchy bool
}

// OptionsFromValues parses filtering options from URL query parameters. It does not validate the
// TraceQL itself (call Compile for that).
func OptionsFromValues(vals url.Values) (Options, error) {
	// keep_hierarchy defaults to false: return only the matched spans unless the caller opts into ancestors.
	// trim so a blank or whitespace-only q is treated as no filter (full trace), matching the docs.
	opts := Options{
		Query: strings.TrimSpace(vals.Get(QueryParam)),
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

// NewFilterFromValues parses and compiles filtering options in one step. Returns (nil, nil) when no
// filtering is requested. Errors are the caller's to map to a 400.
func NewFilterFromValues(vals url.Values) (*Filter, error) {
	opts, err := OptionsFromValues(vals)
	if err != nil {
		return nil, err
	}
	return opts.Compile()
}

// Compile compiles the options into a Filter. Returns (nil, nil) for an empty Query (passthrough).
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

// Process returns a new trace with only the kept spans. It never mutates the input, which may be cached.
func (f *Filter) Process(trace *tempopb.Trace) (*tempopb.Trace, error) {
	if f == nil || trace == nil {
		return trace, nil
	}

	idx := newSpanIndex(trace, f.keepHierarchy)

	matched, err := f.spansetFilter.MatchSpans(idx.spans)
	if err != nil {
		return nil, err
	}

	kept := make(map[string]struct{}, len(matched))
	for _, s := range matched {
		kept[string(s.ID())] = struct{}{}
	}

	if f.keepHierarchy {
		addAncestors(idx, kept)
	}

	return rebuildTrace(trace, kept), nil
}

// addAncestors adds each matched span's ancestors to kept. A duplicate span id may have multiple
// parents, so all are followed, and kept doubles as the visited set so cycles terminate.
func addAncestors(idx *spanIndex, kept map[string]struct{}) {
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
				continue // already kept, also breaks cycles.
			}
			// a dangling parentID is harmless: rebuildTrace only emits spans that exist.
			kept[parentID] = struct{}{}
			queue = append(queue, parentID)
		}
	}
}

// spanIndex is a flattened view of a trace for matching and ancestor walks.
type spanIndex struct {
	spans []traceql.Span // proto-backed spans for the engine.
	// FIXME: matching and hierarchy are keyed by span id only, so identical span ids in different
	// batches/resources collapse: parentsByID merges their parents and rebuildTrace emits both when one
	// matches. Key by span id + resource identity to disambiguate (rare, bad-instrumentation case).
	parentsByID map[string][]string // span id -> distinct parent span ids of all spans sharing that id.
}

func newSpanIndex(trace *tempopb.Trace, keepHierarchy bool) *spanIndex {
	// pre-size to the span count, a lower bound since events/links expand a span into more (append grows it).
	idx := &spanIndex{spans: make([]traceql.Span, 0, countSpans(trace))}
	// parentsByID is only read by addAncestors (keep_hierarchy), so only allocate it then.
	if keepHierarchy {
		idx.parentsByID = make(map[string][]string)
	}
	for _, rs := range trace.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			for _, span := range ss.Spans {
				idx.spans = expandSpanBindings(idx.spans, span, rs.Resource, ss.Scope)
				if keepHierarchy {
					idx.addParent(string(span.SpanId), string(span.ParentSpanId))
				}
			}
		}
	}
	return idx
}

// addParent records a distinct parent id for a span id. Duplicate span ids accumulate every parent
// so the ancestor walk follows all branches instead of an arbitrary last-writer one.
func (idx *spanIndex) addParent(spanID, parentID string) {
	if slices.Contains(idx.parentsByID[spanID], parentID) {
		return
	}
	idx.parentsByID[spanID] = append(idx.parentsByID[spanID], parentID)
}

// rebuildTrace returns a new trace of only the kept spans, preserving grouping and dropping empties.
// It reuses the input's *Span/*Resource/*Scope pointers (only the slices are new), so the result must
// be treated as read-only - the input trace may be cached.
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
