package test

import (
	"math/rand"

	"github.com/grafana/tempo/pkg/tempopb"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

func MakeSpan(traceID []byte) *v1_trace.Span {
	s := &v1_trace.Span{
		Name:    "test",
		TraceId: traceID,
		SpanId:  make([]byte, 8),
	}
	rand.Read(s.SpanId)
	return s
}

func MakeBatch(spans int, traceID []byte) *v1_trace.ResourceSpans {
	traceID = ValidTraceID(traceID)

	batch := &v1_trace.ResourceSpans{}
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

			batch.InstrumentationLibrarySpans = append(batch.InstrumentationLibrarySpans, ils)
		}

		ils.Spans = append(ils.Spans, MakeSpan(traceID))
	}
	return batch
}

func MakeTrace(requests int, traceID []byte) *tempopb.Trace {
	traceID = ValidTraceID(traceID)

	trace := &tempopb.Trace{
		Batches: make([]*v1_trace.ResourceSpans, 0),
	}

	for i := 0; i < requests; i++ {
		trace.Batches = append(trace.Batches, MakeBatch(rand.Int()%20+1, traceID))
	}

	return trace
}

func MakeTraceWithSpanCount(requests int, spansEach int, traceID []byte) *tempopb.Trace {
	trace := &tempopb.Trace{
		Batches: make([]*v1_trace.ResourceSpans, 0),
	}

	for i := 0; i < requests; i++ {
		trace.Batches = append(trace.Batches, MakeBatch(spansEach, traceID))
	}

	return trace
}

func ValidTraceID(traceID []byte) []byte {
	if len(traceID) == 0 {
		traceID = make([]byte, 16)
		rand.Read(traceID)
	}

	for len(traceID) < 16 {
		traceID = append(traceID, 0)
	}

	return traceID
}
