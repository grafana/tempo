// Plumbing for the trace-by-id q filter: parse options, compile, build the span index, walk ancestors
// for keep_hierarchy, and rebuild the trace. Span matching itself lives in protospan.go.

package tracefilter

import (
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level" //nolint:all //deprecated

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
	// expandElements is used to expand event/link elements.
	expandElements bool
	// logger warns when a span's event/link fan-out is truncated. Defaults to a nop logger.
	logger log.Logger
}

// NewFilterFromValues parses and compiles filtering options in one step. Returns (nil, nil) when no
// filtering is requested. Errors are the caller's to map to a 400.
func NewFilterFromValues(vals url.Values, logger log.Logger) (*Filter, error) {
	opts, err := OptionsFromValues(vals)
	if err != nil {
		return nil, err
	}
	f, err := opts.Compile()
	if f != nil && logger != nil {
		f.logger = logger
	}
	return f, err
}

// Compile compiles the options into a Filter. Returns (nil, nil) for an empty Query (passthrough).
func (o Options) Compile() (*Filter, error) {
	if o.Query == "" {
		return nil, nil
	}

	sf, err := traceql.CompileSpansetFilter(o.Query)
	if err != nil {
		return nil, fmt.Errorf("invalid TraceQL filter: %w", err)
	}

	return &Filter{
		spansetFilter:  sf,
		keepHierarchy:  o.KeepHierarchy,
		expandElements: sf.ReferencesEventOrLink(),
		logger:         log.NewNopLogger(),
	}, nil
}

// Process returns a new trace with only the kept spans. It never mutates the input, which may be cached.
func (f *Filter) Process(trace *tempopb.Trace) (*tempopb.Trace, error) {
	if f == nil || trace == nil {
		return trace, nil
	}

	idx := newSpanIndex(trace, f.expandElements, f.keepHierarchy)
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

	var keptAncestorIDs map[string]struct{}
	if f.keepHierarchy {
		keptAncestorIDs = ancestorIDs(idx, keptSpans)
	}

	return rebuildTrace(trace, keptSpans, keptAncestorIDs), nil
}

func ancestorIDs(idx *spanIndex, matched map[*tracev1.Span]struct{}) map[string]struct{} {
	ancestors := make(map[string]struct{})

	queue := make([]string, 0, len(matched))
	for s := range matched {
		queue = append(queue, string(s.SpanId))
	}

	for len(queue) > 0 {
		current := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		for _, parentID := range idx.parentsByID[current] {
			if parentID == "" {
				continue // reached a root.
			}
			if _, seen := ancestors[parentID]; seen {
				continue // already recorded, also breaks cycles.
			}
			// a dangling parentID is harmless: rebuildTrace only emits spans that exist.
			ancestors[parentID] = struct{}{}
			queue = append(queue, parentID)
		}
	}

	return ancestors
}

// spanIndex is a flattened view of a trace for matching and ancestor walks.
type spanIndex struct {
	spans []traceql.Span // proto-backed spans for the engine.
	// Hierarchy is keyed by span id, so identical ids across batches merge parents and can over-include
	// ancestors (rare, bad instrumentation). Matching is pointer-keyed, so matches stay exact.
	parentsByID map[string][]string // span id -> distinct parent span ids of all spans sharing that id.
	// truncatedSpans counts spans whose event x link fan-out hit maxBindingsPerSpan and was cut short.
	truncatedSpans int
}

func newSpanIndex(trace *tempopb.Trace, expandElements, keepHierarchy bool) *spanIndex {
	// pre-size to the span count, a lower bound since events/links expand a span into more (append grows it).
	idx := &spanIndex{spans: make([]traceql.Span, 0, countSpans(trace))}
	// parentsByID is only read by ancestorIDs (keep_hierarchy), so only allocate it then.
	if keepHierarchy {
		idx.parentsByID = make(map[string][]string)
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
// A span is kept if it matched or its id is an ancestor to keep(keep_hierarchy=true). It reuses the
// input's *Span/*Resource/*Scope pointers (only the slices are new), so the result must be treated as read only, and the input trace may be cached.
func rebuildTrace(trace *tempopb.Trace, keptSpans map[*tracev1.Span]struct{}, keptAncestorIDs map[string]struct{}) *tempopb.Trace {
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
