package vparquet4

import (
	"bytes"
	"slices"

	tracev1 "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
)

// buildServiceGraphEdges records every direct parent/child link, including
// internal and same-service spans. Presentation rules never affect this data.
// Parents missing from a fragment remain explicit unresolved records, and can
// be resolved when fragments are combined during compaction.
func buildServiceGraphEdges(trace *Trace) []ServiceGraphEdge {
	type endpoint struct {
		service string
		kind    int32
	}
	spanCount := 0
	for _, rs := range trace.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			spanCount += len(ss.Spans)
		}
	}
	parents := make(map[string]endpoint, spanCount)
	edgeCount := 0
	for _, rs := range trace.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			for _, span := range ss.Spans {
				if !span.IsRoot() {
					edgeCount++
				}
				existing, exists := parents[string(span.SpanID)]
				if !exists || (existing.kind == int32(tracev1.Span_SPAN_KIND_CLIENT) && span.Kind == int(tracev1.Span_SPAN_KIND_SERVER)) {
					parents[string(span.SpanID)] = endpoint{rs.Resource.ServiceName, int32(span.Kind)}
				}
			}
		}
	}
	type key struct {
		parent, child string
		kind          int32
	}
	seen := make(map[key]struct{}, edgeCount)
	edges := make([]ServiceGraphEdge, 0, edgeCount)
	for _, rs := range trace.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			for _, span := range ss.Spans {
				if span.IsRoot() {
					continue
				}
				k := key{string(span.ParentSpanID), string(span.SpanID), int32(span.Kind)}
				if _, exists := seen[k]; exists {
					continue
				}
				seen[k] = struct{}{}
				parent, exists := parents[k.parent]
				edges = append(edges, ServiceGraphEdge{
					ParentSpanID: bytes.Clone(span.ParentSpanID), ChildSpanID: bytes.Clone(span.SpanID),
					ParentService: parent.service, ChildService: rs.Resource.ServiceName,
					ParentKind: parent.kind, ChildKind: int32(span.Kind), ParentMissing: !exists,
				})
			}
		}
	}
	slices.SortFunc(edges, func(a, b ServiceGraphEdge) int {
		if cmp := bytes.Compare(a.ChildSpanID, b.ChildSpanID); cmp != 0 {
			return cmp
		}
		if a.ChildKind < b.ChildKind {
			return -1
		}
		if a.ChildKind > b.ChildKind {
			return 1
		}
		return bytes.Compare(a.ParentSpanID, b.ParentSpanID)
	})
	return edges
}

// reconcileStoredServiceGraphEdges updates only records already present. This
// preserves them through compaction without backfilling disabled trace fragments.
func reconcileStoredServiceGraphEdges(trace *Trace) {
	type key struct {
		parent, child string
		kind          int32
	}
	resolved := make(map[key]ServiceGraphEdge)
	for _, edge := range buildServiceGraphEdges(trace) {
		resolved[key{string(edge.ParentSpanID), string(edge.ChildSpanID), edge.ChildKind}] = edge
	}
	seen := make(map[key]struct{})
	records := trace.ServiceGraphEdges[:0]
	for _, edge := range trace.ServiceGraphEdges {
		k := key{string(edge.ParentSpanID), string(edge.ChildSpanID), edge.ChildKind}
		if _, exists := seen[k]; exists {
			continue
		}
		seen[k] = struct{}{}
		if updated, exists := resolved[k]; exists {
			edge = updated
		}
		records = append(records, edge)
	}
	trace.ServiceGraphEdges = records
}
