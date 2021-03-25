package test

import (
	"math/rand"

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

// Note that this fn will generate a request with size **close to** maxBytes
func MakeRequestWithByteLimit(maxBytes int, traceID []byte) *tempopb.PushRequest {
	if len(traceID) == 0 {
		traceID = make([]byte, 16)
		rand.Read(traceID)
	}

	req := &tempopb.PushRequest{
		Batch: &v1_trace.ResourceSpans{},
	}

	ils := &v1_trace.InstrumentationLibrarySpans{
		InstrumentationLibrary: &v1_common.InstrumentationLibrary{
			Name:    "super library",
			Version: "0.0.1",
		},
	}
	req.Batch.InstrumentationLibrarySpans = append(req.Batch.InstrumentationLibrarySpans, ils)

	for req.Size() < maxBytes {
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