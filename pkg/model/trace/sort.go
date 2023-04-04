package trace

import (
	"bytes"
	"sort"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

// SortTrace sorts a *tempopb.Trace
func SortTrace(t *tempopb.Trace) {
	// Sort bottom up by span start times
	for _, b := range t.Batches {
		for _, ils := range b.ScopeSpans {
			sort.Slice(ils.Spans, func(i, j int) bool {
				return compareSpans(ils.Spans[i], ils.Spans[j])
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

func compareBatches(a *v1.ResourceSpans, b *v1.ResourceSpans) bool {
	if len(a.ScopeSpans) > 0 && len(b.ScopeSpans) > 0 {
		return compareScopeSpans(a.ScopeSpans[0], b.ScopeSpans[0])
	}
	return false
}

func compareScopeSpans(a *v1.ScopeSpans, b *v1.ScopeSpans) bool {
	if len(a.Spans) > 0 && len(b.Spans) > 0 {
		return compareSpans(a.Spans[0], b.Spans[0])
	}
	return false
}

func compareSpans(a *v1.Span, b *v1.Span) bool {
	// Sort by start time, then id
	if a.StartTimeUnixNano == b.StartTimeUnixNano {
		return bytes.Compare(a.SpanId, b.SpanId) == -1
	}

	return a.StartTimeUnixNano < b.StartTimeUnixNano
}
