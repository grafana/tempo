package test

import (
	"math/rand"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/tempopb"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

func makeSpan(traceID []byte) *v1_trace.Span {
	s := &v1_trace.Span{
		Name:    "test",
		TraceId: traceID,
		SpanId:  make([]byte, 8),
	}
	rand.Read(s.SpanId)
	return s
}

func MakeBatch(spans int, traceID []byte) *v1_trace.ResourceSpans {
	traceID = populateTraceID(traceID)

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

		ils.Spans = append(ils.Spans, makeSpan(traceID))
	}
	return batch
}

func makePushBytesRequest(traceID []byte, batch *v1_trace.ResourceSpans) *tempopb.PushBytesRequest {
	trace := &tempopb.Trace{Batches: []*v1_trace.ResourceSpans{batch}}

	// Buffer must come from the pool.
	buffer := tempopb.SliceFromBytePool(trace.Size())
	_, err := trace.MarshalToSizedBuffer(buffer)
	if err != nil {
		panic(err)
	}

	return &tempopb.PushBytesRequest{
		Ids: []tempopb.PreallocBytes{{
			Slice: traceID,
		}},
		Traces: []tempopb.PreallocBytes{{
			Slice: buffer,
		}},
	}
}

func MakeRequest(spans int, traceID []byte) *tempopb.PushBytesRequest {
	traceID = populateTraceID(traceID)
	return makePushBytesRequest(traceID, MakeBatch(spans, traceID))
}

func MakeTraceBytes(requests int, traceID []byte) *tempopb.TraceBytes {
	trace := &tempopb.Trace{
		Batches: make([]*v1_trace.ResourceSpans, 0),
	}

	for i := 0; i < requests; i++ {
		trace.Batches = append(trace.Batches, MakeBatch(rand.Int()%20+1, traceID))
	}

	bytes, err := proto.Marshal(trace)
	if err != nil {
		panic(err)
	}

	traceBytes := &tempopb.TraceBytes{
		Traces: [][]byte{bytes},
	}

	return traceBytes
}

func MakeTrace(requests int, traceID []byte) *tempopb.Trace {
	traceID = populateTraceID(traceID)

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

// Note that this fn will generate a request with size **close to** maxBytes
func MakeRequestWithByteLimit(maxBytes int, traceID []byte) *tempopb.PushBytesRequest {
	traceID = populateTraceID(traceID)
	batch := MakeBatch(1, traceID)

	for batch.Size() < maxBytes {
		batch.InstrumentationLibrarySpans[0].Spans = append(batch.InstrumentationLibrarySpans[0].Spans, makeSpan(traceID))
	}

	return makePushBytesRequest(traceID, batch)
}

func populateTraceID(traceID []byte) []byte {
	if len(traceID) == 0 {
		traceID = make([]byte, 16)
		rand.Read(traceID)
	}

	for len(traceID) < 16 {
		traceID = append(traceID, 0)
	}

	return traceID
}
