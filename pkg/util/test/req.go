package test

import (
	"math/rand"

	"github.com/joe-elliott/frigg/pkg/friggpb"
)

func MakeRequest(spans int, traceID []byte) *friggpb.PushRequest {
	if len(traceID) == 0 {
		traceID = []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}
	}

	sampleSpan := friggpb.Span{
		Name:    "test",
		TraceID: traceID,
		SpanID:  []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
	}

	req := &friggpb.PushRequest{
		Spans: []*friggpb.Span{},
		Process: &friggpb.Process{
			Name: "test",
		},
	}

	for i := 0; i < spans; i++ {
		req.Spans = append(req.Spans, &sampleSpan)
	}

	return req
}

func MakeTrace(requests int, traceID []byte) *friggpb.Trace {
	trace := &friggpb.Trace{
		Batches: make([]*friggpb.PushRequest, 0),
	}

	for i := 0; i < requests; i++ {
		trace.Batches = append(trace.Batches, MakeRequest(rand.Int()%20+1, traceID))
	}

	return trace
}
