package test

import (
	"bytes"
	"math/rand"
	"sort"

	"github.com/grafana/tempo/pkg/tempopb"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

func MakeRequest(spans int, traceID []byte) *tempopb.PushRequest {
	if len(traceID) == 0 {
		traceID = make([]byte, 16)
		rand.Read(traceID)
	}

	req := &tempopb.PushRequest{
		Batch: &v1_trace.ResourceSpans{},
	}

	var ils *v1_trace.InstrumentationLibrarySpans

	for i := 0; i < spans; i++ {
		// occasionally make a new ils
		if ils == nil || rand.Int()%3 == 0 {
			ils = &v1_trace.InstrumentationLibrarySpans{
				InstrumentationLibrary: &v1_common.InstrumentationLibrary{
					Name:    "super library",
					Version: "0.0.1",
				},
			}

			req.Batch.InstrumentationLibrarySpans = append(req.Batch.InstrumentationLibrarySpans, ils)
		}

		sampleSpan := v1_trace.Span{
			Name:    "test",
			TraceId: traceID,
			SpanId:  make([]byte, 8),
		}
		rand.Read(sampleSpan.SpanId)

		ils.Spans = append(ils.Spans, &sampleSpan)
	}

	return req
}

func MakeTrace(requests int, traceID []byte) *tempopb.Trace {
	trace := &tempopb.Trace{
		Batches: make([]*v1_trace.ResourceSpans, 0),
	}

	for i := 0; i < requests; i++ {
		trace.Batches = append(trace.Batches, MakeRequest(rand.Int()%20+1, traceID).Batch)
	}

	return trace
}

func MakeTraceWithSpanCount(requests int, spansEach int, traceID []byte) *tempopb.Trace {
	trace := &tempopb.Trace{
		Batches: make([]*v1_trace.ResourceSpans, 0),
	}

	for i := 0; i < requests; i++ {
		trace.Batches = append(trace.Batches, MakeRequest(spansEach, traceID).Batch)
	}

	return trace
}

func SortTrace(t *tempopb.Trace) {
	sort.Slice(t.Batches, func(i, j int) bool {
		return bytes.Compare(t.Batches[i].InstrumentationLibrarySpans[0].Spans[0].SpanId, t.Batches[j].InstrumentationLibrarySpans[0].Spans[0].SpanId) == 1
	})

	for _, b := range t.Batches {
		sort.Slice(b.InstrumentationLibrarySpans, func(i, j int) bool {
			return bytes.Compare(b.InstrumentationLibrarySpans[i].Spans[0].SpanId, b.InstrumentationLibrarySpans[j].Spans[0].SpanId) == 1
		})

		for _, ils := range b.InstrumentationLibrarySpans {
			sort.Slice(ils.Spans, func(i, j int) bool {
				return bytes.Compare(ils.Spans[i].SpanId, ils.Spans[j].SpanId) == 1
			})
		}
	}
}
