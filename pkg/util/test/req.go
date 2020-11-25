package test

import (
	"math/rand"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
	opentelemetry_proto_trace_v1 "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"
)

func MakeRequest(spans int, traceID []byte) *tempopb.PushRequest {
	if len(traceID) == 0 {
		traceID = make([]byte, 16)
		rand.Read(traceID)
	}

	req := &tempopb.PushRequest{
		Batch: &opentelemetry_proto_trace_v1.ResourceSpans{},
	}

	var ils *opentelemetry_proto_trace_v1.InstrumentationLibrarySpans

	for i := 0; i < spans; i++ {
		// occasionally make a new ils
		if ils == nil || rand.Int()%3 == 0 {
			ils = &opentelemetry_proto_trace_v1.InstrumentationLibrarySpans{
				InstrumentationLibrary: &v1.InstrumentationLibrary{
					Name:    "super library",
					Version: "0.0.1",
				},
			}

			req.Batch.InstrumentationLibrarySpans = append(req.Batch.InstrumentationLibrarySpans, ils)
		}

		sampleSpan := opentelemetry_proto_trace_v1.Span{
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
		Batches: make([]*opentelemetry_proto_trace_v1.ResourceSpans, 0),
	}

	for i := 0; i < requests; i++ {
		trace.Batches = append(trace.Batches, MakeRequest(rand.Int()%20+1, traceID).Batch)
	}

	return trace
}

func MakeTraceWithSpanCount(requests int, spansEach int, traceID []byte) *tempopb.Trace {
	trace := &tempopb.Trace{
		Batches: make([]*opentelemetry_proto_trace_v1.ResourceSpans, 0),
	}

	for i := 0; i < requests; i++ {
		trace.Batches = append(trace.Batches, MakeRequest(spansEach, traceID).Batch)
	}

	return trace
}
