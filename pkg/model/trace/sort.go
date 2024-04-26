package trace

import (
	"bytes"
	"sort"

	v1 "github.com/grafana/tempo/pkg/tempopb/opentelemetry/proto/trace/v1"

	"github.com/grafana/tempo/pkg/tempopb"
)

// SortTrace sorts a *tempopb.Trace
func SortTrace(t *tempopb.Trace) {
	// Sort bottom up by span start times
	for _, b := range t.Batches {
		for _, ss := range b.ScopeSpans {
			sort.Slice(ss.Spans, func(i, j int) bool {
				return compareSpans(ss.Spans[i], ss.Spans[j])
			})
		}
		sort.Slice(b.ScopeSpans, func(i, j int) bool {
			return compareScopeSpans(b.ScopeSpans[i], b.ScopeSpans[j])
		})
	}
	sort.Slice(t.Batches, func(i, j int) bool {
		return compareBatches(t.Batches[i], t.Batches[j])
	})
}

// SortTraceAndAttributes sorts a *tempopb.Trace like SortTrace, but also
// sorts all resource and span attributes by name.
func SortTraceAndAttributes(t *tempopb.Trace) {
	SortTrace(t)
	for _, b := range t.Batches {
		res := b.Resource
		sort.Slice(res.Attributes, func(i, j int) bool {
			return res.Attributes[i].Key < res.Attributes[j].Key
		})
		for _, ss := range b.ScopeSpans {
			for _, span := range ss.Spans {
				sort.Slice(span.Attributes, func(i, j int) bool {
					return span.Attributes[i].Key < span.Attributes[j].Key
				})
			}
		}
	}
}

func compareBatches(a, b *v1.ResourceSpans) bool {
	if len(a.ScopeSpans) > 0 && len(b.ScopeSpans) > 0 {
		return compareScopeSpans(a.ScopeSpans[0], b.ScopeSpans[0])
	}
	return false
}

func compareScopeSpans(a, b *v1.ScopeSpans) bool {
	if len(a.Spans) > 0 && len(b.Spans) > 0 {
		return compareSpans(a.Spans[0], b.Spans[0])
	}
	return false
}

func compareSpans(a, b *v1.Span) bool {
	// Sort by start time, then id
	if a.StartTimeUnixNano == b.StartTimeUnixNano {
		return bytes.Compare(a.SpanId, b.SpanId) == -1
	}

	return a.StartTimeUnixNano < b.StartTimeUnixNano
}
