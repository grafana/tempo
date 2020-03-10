package test

import (
	"math/rand"

	"github.com/grafana/tempo/pkg/tempopb"
	opentelemetry_proto_trace_v1 "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"
)

func MakeRequest(spans int, traceID []byte) *tempopb.PushRequest {
	if len(traceID) == 0 {
		traceID = []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}
	}

	sampleSpan := opentelemetry_proto_trace_v1.Span{
		Name:    "test",
		TraceId: traceID,
		SpanId:  []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
	}

	req := &tempopb.PushRequest{
		Batch: &opentelemetry_proto_trace_v1.ResourceSpans{},
	}

	for i := 0; i < spans; i++ {
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
