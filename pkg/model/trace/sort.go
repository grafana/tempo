package trace

import (
	"bytes"
	"sort"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

// SortTrace deeply sorts a *tempopb.Trace. All scopes, spans, events, etc are sorted by
// data intrinsic like timestamp or id.
func SortTrace(t *tempopb.Trace) {
	// Sort bottom up by span start times
	for _, b := range t.ResourceSpans {
		for _, ss := range b.ScopeSpans {
			for _, span := range ss.Spans {
				sort.Slice(span.Events, func(i, j int) bool {
					return compareEvents(span.Events[i], span.Events[j])
				})
				sort.Slice(span.Links, func(i, j int) bool {
					return compareLinks(span.Links[i], span.Links[j])
				})
			}
			sort.Slice(ss.Spans, func(i, j int) bool {
				return compareSpans(ss.Spans[i], ss.Spans[j])
			})
		}
		sort.Slice(b.ScopeSpans, func(i, j int) bool {
			return compareScopeSpans(b.ScopeSpans[i], b.ScopeSpans[j])
		})
	}
	sort.Slice(t.ResourceSpans, func(i, j int) bool {
		return compareBatches(t.ResourceSpans[i], t.ResourceSpans[j])
	})
}

// SortTraceAndAttributes sorts a *tempopb.Trace like SortTrace, but also
// sorts all resource and span attributes by name.
func SortTraceAndAttributes(t *tempopb.Trace) {
	SortTrace(t)
	for _, b := range t.ResourceSpans {
		if res := b.Resource; res != nil {
			sort.Slice(res.Attributes, func(i, j int) bool {
				return res.Attributes[i].Key < res.Attributes[j].Key
			})
		}
		for _, ss := range b.ScopeSpans {
			if ss.Scope != nil {
				sort.Slice(ss.Scope.Attributes, func(i, j int) bool {
					return ss.Scope.Attributes[i].Key < ss.Scope.Attributes[j].Key
				})
			}
			for _, span := range ss.Spans {
				sort.Slice(span.Attributes, func(i, j int) bool {
					return span.Attributes[i].Key < span.Attributes[j].Key
				})
				for _, event := range span.Events {
					sort.Slice(event.Attributes, func(i, j int) bool {
						return event.Attributes[i].Key < event.Attributes[j].Key
					})
				}
				for _, link := range span.Links {
					sort.Slice(link.Attributes, func(i, j int) bool {
						return link.Attributes[i].Key < link.Attributes[j].Key
					})
				}
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

func compareEvents(a, b *v1.Span_Event) bool {
	if a.TimeUnixNano == b.TimeUnixNano {
		return a.Name < b.Name
	}

	return a.TimeUnixNano < b.TimeUnixNano
}

func compareLinks(a, b *v1.Span_Link) bool {
	if bytes.Equal(a.TraceId, b.TraceId) {
		return bytes.Compare(a.SpanId, b.SpanId) == -1
	}

	return bytes.Compare(a.TraceId, b.TraceId) == -1
}
