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
		for _, ils := range b.InstrumentationLibrarySpans {
			sort.Slice(ils.Spans, func(i, j int) bool {
				return compareSpans(ils.Spans[i], ils.Spans[j])
			})
		}
		sort.Slice(b.InstrumentationLibrarySpans, func(i, j int) bool {
			return compareIls(b.InstrumentationLibrarySpans[i], b.InstrumentationLibrarySpans[j])
		})
	}
	sort.Slice(t.Batches, func(i, j int) bool {
		return compareBatches(t.Batches[i], t.Batches[j])
	})
}

func compareBatches(a *v1.ResourceSpans, b *v1.ResourceSpans) bool {
	if len(a.InstrumentationLibrarySpans) > 0 && len(b.InstrumentationLibrarySpans) > 0 {
		return compareIls(a.InstrumentationLibrarySpans[0], b.InstrumentationLibrarySpans[0])
	}
	return false
}

func compareIls(a *v1.InstrumentationLibrarySpans, b *v1.InstrumentationLibrarySpans) bool {
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

// SortTraceBytes sorts a *tempopb.TraceBytes
func SortTraceBytes(t *tempopb.TraceBytes) {
	sort.Slice(t.Traces, func(i, j int) bool {
		traceI := t.Traces[i]
		traceJ := t.Traces[j]

		return bytes.Compare(traceI, traceJ) == -1
	})
}
