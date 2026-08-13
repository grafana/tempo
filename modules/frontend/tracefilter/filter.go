// Plumbing for the trace-by-id q filter: parse options, compile, build the span index, walk ancestors
// and descendants of matched spans with depthBoundedWalk, and rebuild the trace. Span matching itself
// lives in protospan.go.

package tracefilter

import (
	"fmt"
	"slices"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level" //nolint:all //deprecated

	"github.com/grafana/tempo/pkg/tempopb"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
)

// Options holds a request's filtering options, parsed by api.ParseTraceByIDFilterParams.
type Options struct {
	// Query is a single TraceQL spanset filter. Empty means no filtering.
	Query string
	// KeepHierarchy includes each matched span's ancestor path. Ignored when Query is empty.
	KeepHierarchy bool
	// MatchDepth bounds how many hops of descendants of each matched span are kept, cumulatively.
	// -1 means unlimited and 0 means no descendants are kept; -1 is the only negative value accepted,
	// and Compile rejects anything below it. Ignored when Query is empty.
	MatchDepth int
	// AncestorDepth bounds how many hops of ancestors of each matched span are kept, cumulatively.
	// -1 means unlimited and 0 means no ancestors are kept. Ignored when Query is empty or
	// KeepHierarchy is false, and only checked when it is read: -1 is then the only negative value
	// accepted, and Compile rejects anything below it.
	AncestorDepth int
}

// Filter is a compiled, ready-to-apply trace filter.
type Filter struct {
	spansetFilter *traceql.SpansetFilter
	keepHierarchy bool
	matchDepth    int
	ancestorDepth int
	// expandElements is used to expand event/link elements.
	expandElements bool
	// logger warns when a span's event/link fan-out is truncated. Defaults to a nop logger.
	logger log.Logger
}

// NewFilter compiles the options into a Filter that logs with logger. Returns (nil, nil) when no
// filtering is requested. Errors are the caller's to map to a 400.
func NewFilter(opts Options, logger log.Logger) (*Filter, error) {
	f, err := opts.Compile()
	if f != nil && logger != nil {
		f.logger = logger
	}
	return f, err
}

// Compile compiles the options into a Filter. Returns (nil, nil) for an empty Query (passthrough).
// -1 is the only negative depth with a meaning, so anything below it is rejected rather than quietly
// treated as unlimited: a caller that passes -5 is confused about the contract, not asking for the
// whole subtree. A depth is only checked where it is actually read, so AncestorDepth is validated
// only under KeepHierarchy — the same rule api.ParseTraceByIDFilterParams applies to the request
// params. Errors are the caller's to map to a 400.
func (o Options) Compile() (*Filter, error) {
	if o.Query == "" {
		return nil, nil
	}

	if o.KeepHierarchy {
		if o.AncestorDepth < -1 {
			return nil, fmt.Errorf("invalid ancestor depth %d: must be >= -1", o.AncestorDepth)
		}
	}

	if o.MatchDepth < -1 {
		return nil, fmt.Errorf("invalid match depth %d: must be >= -1", o.MatchDepth)
	}

	sf, err := traceql.CompileSpansetFilter(o.Query)
	if err != nil {
		return nil, fmt.Errorf("invalid TraceQL filter: %w", err)
	}

	return &Filter{
		spansetFilter:  sf,
		keepHierarchy:  o.KeepHierarchy,
		matchDepth:     o.MatchDepth,
		ancestorDepth:  o.AncestorDepth,
		expandElements: sf.ReferencesEventOrLink(),
		logger:         log.NewNopLogger(),
	}, nil
}

// Process returns a new trace with only the kept spans. It never mutates the input, which may be cached.
func (f *Filter) Process(trace *tempopb.Trace) (*tempopb.Trace, error) {
	if f == nil || trace == nil {
		return trace, nil
	}

	idx := newSpanIndex(trace, f.expandElements, f.keepHierarchy, f.matchDepth != 0)
	if idx.truncatedSpans > 0 {
		level.Warn(f.logger).Log("msg", "trace by id q filter: span event/link fan-out hit the cap, some spans may under-match", "cap", maxBindingsPerSpan, "truncated_spans", idx.truncatedSpans)
	}

	matched, err := f.spansetFilter.MatchSpans(idx.spans)
	if err != nil {
		return nil, err
	}

	// keyed by *Span pointer, and not span id, so two spans sharing an id don't both get kept when only one matched.
	keptSpans := make(map[*tracev1.Span]struct{}, len(matched))
	for _, s := range matched {
		if ps, ok := s.(*protoSpan); ok {
			keptSpans[ps.span] = struct{}{}
		}
	}

	var keptAncestorIDs, keptDescendantIDs map[string]struct{}
	if f.keepHierarchy {
		keptAncestorIDs = depthBoundedWalk(idx.parentsByID, keptSpans, f.ancestorDepth)
	}
	if f.matchDepth != 0 {
		keptDescendantIDs = depthBoundedWalk(idx.childrenByID, keptSpans, f.matchDepth)
	}

	return rebuildTrace(trace, keptSpans, keptAncestorIDs, keptDescendantIDs), nil
}

// idAndDepth pairs a span id with the number of hops it took to reach it from a seed, for BFS.
type idAndDepth struct {
	id    string
	depth int
}

// depthBoundedWalk does a breadth-first walk of adjacency starting from the ids of seeds, returning
// every id reachable within maxDepth hops (cumulative). maxDepth -1 means unlimited (whole reachable
// graph) and is the only negative Compile lets through; maxDepth == 0 returns an empty result. The
// guard below is written as maxDepth < 0 so a stray negative degrades to unlimited instead of
// silently truncating, but callers are validated, not trusted to be lucky. adjacency maps an id to
// its neighbor ids: pass
// parentsByID for an ancestor walk (skipping the "" root sentinel) or childrenByID for a descendant walk.
// True BFS (front-of-queue pop) is required: a node must be assigned its shortest-path depth, since a
// depth cutoff applied to a DFS could reach a node via a longer path first and wrongly exclude it.
func depthBoundedWalk(adjacency map[string][]string, seeds map[*tracev1.Span]struct{}, maxDepth int) map[string]struct{} {
	result := make(map[string]struct{})
	if maxDepth == 0 {
		return result
	}

	queue := make([]idAndDepth, 0, len(seeds))
	for s := range seeds {
		queue = append(queue, idAndDepth{id: string(s.SpanId), depth: 0})
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, neighborID := range adjacency[current.id] {
			if neighborID == "" {
				continue // reached a root.
			}
			if _, seen := result[neighborID]; seen {
				continue // already recorded, also breaks cycles.
			}
			nextDepth := current.depth + 1
			if maxDepth >= 0 && nextDepth > maxDepth {
				continue
			}
			// a dangling neighborID is harmless: rebuildTrace only emits spans that exist.
			result[neighborID] = struct{}{}
			queue = append(queue, idAndDepth{id: neighborID, depth: nextDepth})
		}
	}

	return result
}

// spanIndex is a flattened view of a trace for matching and ancestor/descendant walks.
type spanIndex struct {
	spans []traceql.Span
	// hierarchy is keyed by span id, so identical ids across batches merge parents/children and can
	// over-include ancestors/descendants (rare, bad instrumentation). Matching is pointer-keyed, so
	// matches stay exact.
	parentsByID map[string][]string
	// childrenByID maps a parent span id to all of its child span ids. Unlike parentsByID, entries are
	// not deduped: a parent can legitimately have many distinct children.
	childrenByID map[string][]string
	// truncatedSpans counts spans whose event x link fan-out hit maxBindingsPerSpan and was cut short.
	truncatedSpans int
}

func newSpanIndex(trace *tempopb.Trace, expandElements, keepHierarchy, buildChildren bool) *spanIndex {
	// pre-size to the span count, a lower bound since events/links expand a span into more (append grows it).
	idx := &spanIndex{spans: make([]traceql.Span, 0, countSpans(trace))}
	// parentsByID is only read by the ancestor walk (keep_hierarchy), so only allocate it then.
	if keepHierarchy {
		idx.parentsByID = make(map[string][]string)
	}
	// childrenByID is only read by the descendant walk (match_depth != 0), so only allocate it then.
	if buildChildren {
		idx.childrenByID = make(map[string][]string)
	}
	for _, rs := range trace.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			for _, span := range ss.Spans {
				var truncated bool
				idx.spans, truncated = expandSpanBindings(idx.spans, span, rs.Resource, ss.Scope, expandElements)
				if truncated {
					idx.truncatedSpans++
				}
				if keepHierarchy {
					idx.addParent(string(span.SpanId), string(span.ParentSpanId))
				}
				if buildChildren {
					idx.addChild(string(span.ParentSpanId), string(span.SpanId))
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

// addChild records a child id under its parent id. A parent can legitimately have many distinct
// children, so unlike addParent this does not dedup.
func (idx *spanIndex) addChild(parentID, childID string) {
	idx.childrenByID[parentID] = append(idx.childrenByID[parentID], childID)
}

// rebuildTrace returns a new trace of only the kept spans, preserving grouping and dropping empties.
// A span is kept if it matched, its id is an ancestor to keep (keep_hierarchy=true), or its id is a
// descendant to keep (match_depth != 0). It reuses the input's *Span/*Resource/*Scope pointers (only
// the slices are new), so the result must be treated as read only, and the input trace may be cached.
func rebuildTrace(trace *tempopb.Trace, keptSpans map[*tracev1.Span]struct{}, keptAncestorIDs, keptDescendantIDs map[string]struct{}) *tempopb.Trace {
	out := &tempopb.Trace{}

	for _, rs := range trace.ResourceSpans {
		var keptScopes []*tracev1.ScopeSpans
		for _, ss := range rs.ScopeSpans {
			var kept []*tracev1.Span
			for _, span := range ss.Spans {
				if _, ok := keptSpans[span]; ok {
					kept = append(kept, span)
					continue
				}
				if _, ok := keptAncestorIDs[string(span.SpanId)]; ok {
					kept = append(kept, span)
					continue
				}
				if _, ok := keptDescendantIDs[string(span.SpanId)]; ok {
					kept = append(kept, span)
				}
			}
			if len(kept) == 0 {
				continue
			}
			keptScopes = append(keptScopes, &tracev1.ScopeSpans{
				Scope:     ss.Scope,
				SchemaUrl: ss.SchemaUrl,
				Spans:     kept,
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
