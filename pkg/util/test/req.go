package test

import (
	"math/rand"

	"github.com/grafana/tempo/pkg/tempopb"
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

	for i := 0; i < spans; i++ {
		sampleSpan := opentelemetry_proto_trace_v1.Span{
			Name:    "test",
			TraceId: traceID,
			SpanId:  make([]byte, 8),
		}
		rand.Read(sampleSpan.SpanId)

		req.Batch.Spans = append(req.Batch.Spans, &sampleSpan)
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
