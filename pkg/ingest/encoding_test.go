package ingest

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	resourcev1 "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
)

func TestEncoderDecoder(t *testing.T) {
	tests := []struct {
		name        string
		req         *tempopb.PushBytesRequest
		maxSize     int
		expectSplit bool
	}{
		{
			name:        "Small trace, no split",
			req:         generateRequest(10, 100),
			maxSize:     1024 * 1024,
			expectSplit: false,
		},
		{
			name:        "Large trace, expect split",
			req:         generateRequest(1000, 1000),
			maxSize:     1024 * 10,
			expectSplit: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := NewDecoder()

			records, err := Encode(0, "test-tenant", tt.req, tt.maxSize)
			require.NoError(t, err)

			if tt.expectSplit {
				require.Greater(t, len(records), 1)
			} else {
				require.Equal(t, 1, len(records))
			}

			var decodedEntries []tempopb.PreallocBytes
			var decodedIDs [][]byte

			for _, record := range records {
				decoder.Reset()
				req, err := decoder.Decode(record.Value)
				require.NoError(t, err)
				decodedEntries = append(decodedEntries, req.Traces...)
				decodedIDs = append(decodedIDs, req.Ids...)
			}

			require.Equal(t, len(tt.req.Traces), len(decodedEntries))
			for i := range tt.req.Traces {
				require.Equal(t, tt.req.Traces[i], decodedEntries[i])
				require.Equal(t, tt.req.Ids[i], decodedIDs[i])
			}
		})
	}
}

func TestEncoderSingleEntryTooLarge(t *testing.T) {
	stream := generateRequest(1, 1000)

	_, err := Encode(0, "test-tenant", stream, 100)
	require.Error(t, err)
	require.Contains(t, err.Error(), "single entry size")
}

func TestDecoderInvalidData(t *testing.T) {
	decoder := NewDecoder()

	_, err := decoder.Decode([]byte("invalid data"))
	require.Error(t, err)
}

func TestPushBytesDecoder(t *testing.T) {
	firstTrace := marshalBatches(t, []*v1.ResourceSpans{
		test.MakeBatch(1, []byte("test batch 1")),
	})
	secondTrace := marshalBatches(t, []*v1.ResourceSpans{
		test.MakeBatch(2, []byte("test batch 2")),
		test.MakeBatch(3, []byte("test batch 3")),
	})
	req := &tempopb.PushBytesRequest{
		Traces: []tempopb.PreallocBytes{
			{Slice: firstTrace},
			{Slice: secondTrace},
		},
		Ids:                   [][]byte{[]byte("first"), []byte("second")},
		SkipMetricsGeneration: true,
	}
	data, err := req.Marshal()
	require.NoError(t, err)

	decoder := NewPushBytesDecoder()
	iterator, err := decoder.Decode(data)
	require.NoError(t, err)

	var got []*tempopb.PushSpansRequest
	for req, err := range iterator {
		require.NoError(t, err)
		got = append(got, req)
	}

	require.Len(t, got, 2)
	require.True(t, got[0].SkipMetricsGeneration)
	require.True(t, got[1].SkipMetricsGeneration)
	require.Len(t, got[0].Batches, 1)
	require.Len(t, got[1].Batches, 2)
}

func TestPushBytesDecoderInvalidData(t *testing.T) {
	decoder := NewPushBytesDecoder()

	_, err := decoder.Decode([]byte("invalid data"))
	require.Error(t, err)
}

// TestPushBytesDecoder_DoesNotRetainPriorPayload verifies that decoding a
// large payload followed by a smaller one does not keep the large one alive
// through the traceSlices tail capacity. Before the fix, `traceSlices = [:0]`
// preserved the previous Decode's []byte entries in the backing array,
// pinning the prior `data` buffer.
func TestPushBytesDecoder_DoesNotRetainPriorPayload(t *testing.T) {
	decoder := NewPushBytesDecoder()

	largePayload := &tempopb.PushBytesRequest{Traces: make([]tempopb.PreallocBytes, 0, 4)}
	for i := 0; i < 4; i++ {
		largePayload.Traces = append(largePayload.Traces, tempopb.PreallocBytes{
			Slice: marshalBatches(t, []*v1.ResourceSpans{test.MakeBatch(1, []byte(fmt.Sprintf("large-%d", i)))}),
		})
		largePayload.Ids = append(largePayload.Ids, []byte(fmt.Sprintf("id-%d", i)))
	}
	largeData, err := largePayload.Marshal()
	require.NoError(t, err)

	smallPayload := &tempopb.PushBytesRequest{
		Traces: []tempopb.PreallocBytes{
			{Slice: marshalBatches(t, []*v1.ResourceSpans{test.MakeBatch(1, []byte("small"))})},
		},
		Ids: [][]byte{[]byte("only")},
	}
	smallData, err := smallPayload.Marshal()
	require.NoError(t, err)

	// Drain the large payload.
	largeIter, err := decoder.Decode(largeData)
	require.NoError(t, err)
	for _, err := range largeIter {
		require.NoError(t, err)
	}

	// After re-decoding the small payload, none of the tail traceSlices entries
	// should still point at the largeData buffer.
	smallIter, err := decoder.Decode(smallData)
	require.NoError(t, err)
	for _, err := range smallIter {
		require.NoError(t, err)
	}

	// Every traceSlices entry past the current length must be a nil header so
	// it cannot pin the previous Decode's data buffer.
	tail := decoder.traceSlices[len(decoder.traceSlices):cap(decoder.traceSlices)]
	for i, b := range tail {
		require.Nil(t, b, "traceSlices tail entry %d retained a stale payload reference", i)
	}
}

// TestPushBytesDecoder_EarlyTerminate verifies the iterator stops yielding
// after the consumer returns false (e.g. by `break` in a range-over-func).
func TestPushBytesDecoder_EarlyTerminate(t *testing.T) {
	req := &tempopb.PushBytesRequest{
		Traces: []tempopb.PreallocBytes{
			{Slice: marshalBatches(t, []*v1.ResourceSpans{test.MakeBatch(1, []byte("first"))})},
			{Slice: marshalBatches(t, []*v1.ResourceSpans{test.MakeBatch(1, []byte("second"))})},
			{Slice: marshalBatches(t, []*v1.ResourceSpans{test.MakeBatch(1, []byte("third"))})},
		},
		Ids: [][]byte{[]byte("a"), []byte("b"), []byte("c")},
	}
	data, err := req.Marshal()
	require.NoError(t, err)

	iterator, err := NewPushBytesDecoder().Decode(data)
	require.NoError(t, err)

	seen := 0
	for req, err := range iterator {
		require.NoError(t, err)
		require.NotNil(t, req)
		seen++
		if seen == 1 {
			break
		}
	}
	require.Equal(t, 1, seen, "iterator must stop yielding after consumer breaks")
}

// TestPushBytesDecoder_PerTraceUnmarshalError verifies that a malformed
// sub-trace within an otherwise valid PushBytesRequest yields its error
// through the iterator instead of being swallowed.
func TestPushBytesDecoder_PerTraceUnmarshalError(t *testing.T) {
	good := marshalBatches(t, []*v1.ResourceSpans{test.MakeBatch(1, []byte("good"))})
	req := &tempopb.PushBytesRequest{
		Traces: []tempopb.PreallocBytes{
			{Slice: good},
			{Slice: []byte{0xff, 0xff, 0xff, 0xff}}, // not a valid Trace proto
			{Slice: good},
		},
		Ids: [][]byte{[]byte("a"), []byte("b"), []byte("c")},
	}
	data, err := req.Marshal()
	require.NoError(t, err)

	iterator, err := NewPushBytesDecoder().Decode(data)
	require.NoError(t, err)

	errs := 0
	oks := 0
	for _, err := range iterator {
		if err != nil {
			errs++
		} else {
			oks++
		}
	}
	require.Equal(t, 1, errs, "exactly one malformed sub-trace should error")
	require.Equal(t, 2, oks, "the two valid sub-traces should still yield")
}

func TestOTLPDecoderReuseDoesNotLeakPreviousFields(t *testing.T) {
	first := []*v1.ResourceSpans{
		{
			Resource: &resourcev1.Resource{
				Attributes: []*commonv1.KeyValue{
					test.MakeAttribute("service.name", "first"),
					test.MakeAttribute("first.only", "resource"),
				},
				DroppedAttributesCount: 7,
			},
			ScopeSpans: []*v1.ScopeSpans{
				{
					Scope: &commonv1.InstrumentationScope{
						Name:    "scope-first",
						Version: "v1",
						Attributes: []*commonv1.KeyValue{
							test.MakeAttribute("scope.attr", "first"),
						},
						DroppedAttributesCount: 3,
					},
					Spans: []*v1.Span{
						{
							TraceId:           []byte("0123456789abcdef"),
							SpanId:            []byte("12345678"),
							ParentSpanId:      []byte("87654321"),
							TraceState:        "state=first",
							Name:              "first",
							Kind:              v1.Span_SPAN_KIND_CLIENT,
							StartTimeUnixNano: 1,
							EndTimeUnixNano:   2,
							Attributes: []*commonv1.KeyValue{
								test.MakeAttribute("span.attr", "first"),
							},
							Events: []*v1.Span_Event{
								{
									TimeUnixNano: 1,
									Name:         "event-first",
									Attributes: []*commonv1.KeyValue{
										test.MakeAttribute("event.attr", "first"),
									},
									DroppedAttributesCount: 4,
								},
							},
							Links: []*v1.Span_Link{
								{
									TraceId:    []byte("fedcba9876543210"),
									SpanId:     []byte("87654321"),
									TraceState: "state=link",
									Attributes: []*commonv1.KeyValue{
										test.MakeAttribute("link.attr", "first"),
									},
									DroppedAttributesCount: 5,
									Flags:                  6,
								},
							},
							Status: &v1.Status{
								Message: "first status",
								Code:    v1.Status_STATUS_CODE_ERROR,
							},
							DroppedAttributesCount: 8,
							DroppedEventsCount:     9,
							DroppedLinksCount:      10,
							Flags:                  11,
						},
					},
				},
			},
			SchemaUrl: "schema-first",
		},
	}
	second := []*v1.ResourceSpans{
		{
			ScopeSpans: []*v1.ScopeSpans{
				{
					Spans: []*v1.Span{
						{
							TraceId:           []byte("fedcba9876543210"),
							SpanId:            []byte("abcdefgh"),
							Name:              "second",
							Kind:              v1.Span_SPAN_KIND_SERVER,
							StartTimeUnixNano: 3,
							EndTimeUnixNano:   4,
							Attributes: []*commonv1.KeyValue{
								{
									Key: "http.status_code",
									Value: &commonv1.AnyValue{
										Value: &commonv1.AnyValue_IntValue{IntValue: 200},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	decoder := NewOTLPDecoder()
	require.Equal(t, first, decodeSinglePushSpans(t, decoder, marshalBatches(t, first)).Batches)

	got := decodeSinglePushSpans(t, decoder, marshalBatches(t, second))
	require.Len(t, got.Batches, 1)
	require.Nil(t, got.Batches[0].Resource)
	require.Empty(t, got.Batches[0].SchemaUrl)
	require.Len(t, got.Batches[0].ScopeSpans, 1)
	require.Nil(t, got.Batches[0].ScopeSpans[0].Scope)
	require.Empty(t, got.Batches[0].ScopeSpans[0].SchemaUrl)
	require.Len(t, got.Batches[0].ScopeSpans[0].Spans, 1)

	span := got.Batches[0].ScopeSpans[0].Spans[0]
	require.Equal(t, []byte("fedcba9876543210"), span.TraceId)
	require.Equal(t, []byte("abcdefgh"), span.SpanId)
	require.Empty(t, span.ParentSpanId)
	require.Empty(t, span.TraceState)
	require.Equal(t, "second", span.Name)
	require.Equal(t, v1.Span_SPAN_KIND_SERVER, span.Kind)
	require.Equal(t, uint64(3), span.StartTimeUnixNano)
	require.Equal(t, uint64(4), span.EndTimeUnixNano)
	require.Len(t, span.Attributes, 1)
	require.Equal(t, "http.status_code", span.Attributes[0].Key)
	require.Equal(t, int64(200), span.Attributes[0].Value.GetIntValue())
	require.Empty(t, span.Events)
	require.Empty(t, span.Links)
	require.Nil(t, span.Status)
	require.Zero(t, span.DroppedAttributesCount)
	require.Zero(t, span.DroppedEventsCount)
	require.Zero(t, span.DroppedLinksCount)
	require.Zero(t, span.Flags)
}

func TestEncoderDecoderEmptyStream(t *testing.T) {
	decoder := NewDecoder()

	req := &tempopb.PushBytesRequest{}

	records, err := Encode(0, "test-tenant", req, 10<<20)
	require.NoError(t, err)
	require.Len(t, records, 1)

	decodedReq, err := decoder.Decode(records[0].Value)
	require.NoError(t, err)
	require.Equal(t, req.Traces, decodedReq.Traces)
}

func BenchmarkEncodeDecode(b *testing.B) {
	decoder := NewDecoder()
	stream := generateRequest(1000, 200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		records, err := Encode(0, "test-tenant", stream, 10<<20)
		if err != nil {
			b.Fatal(err)
		}
		for _, record := range records {
			_, err := decoder.Decode(record.Value)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func TestResetPushBytesRequest(t *testing.T) {
	// Create a request with all fields set to non-zero values
	traces := []tempopb.PreallocBytes{
		{Slice: []byte("trace1")},
		{Slice: []byte("trace2")},
		{Slice: []byte("trace3")},
	}
	ids := [][]byte{
		[]byte("id1"),
		[]byte("id2"),
		[]byte("id3"),
	}
	req := &tempopb.PushBytesRequest{
		Traces:                traces,
		Ids:                   ids,
		SkipMetricsGeneration: true,
	}

	resetPushBytesRequest(req)

	// Verify all fields are properly reset
	assert.NotNil(t, req.Traces, "Traces should not be nil")
	assert.NotNil(t, req.Ids, "Ids should not be nil")
	assert.Equal(t, 0, len(req.Traces), "Traces should be empty after reset")
	assert.Equal(t, 0, len(req.Ids), "Ids should be empty after reset")
	assert.Equal(t, false, req.SkipMetricsGeneration, "SkipMetricsGeneration should be reset to false")

	// Verify slices are reused (not reallocated) by comparing pointers
	originalTracesPtr := reflect.ValueOf(traces).Pointer()
	newTracesPtr := reflect.ValueOf(req.Traces).Pointer()
	assert.Equal(t, originalTracesPtr, newTracesPtr, "Traces slice should be reused, not reallocated")

	originalIDsPtr := reflect.ValueOf(ids).Pointer()
	newIDsPtr := reflect.ValueOf(req.Ids).Pointer()
	assert.Equal(t, originalIDsPtr, newIDsPtr, "Ids slice should be reused, not reallocated")
}

// Helper function to generate a test trace
func generateRequest(entries, lineLength int) *tempopb.PushBytesRequest {
	stream := &tempopb.PushBytesRequest{
		Traces: make([]tempopb.PreallocBytes, entries),
		Ids:    make([][]byte, entries),
	}

	for i := 0; i < entries; i++ {
		stream.Traces[i].Slice = generateRandomString(lineLength)
		stream.Ids[i] = generateRandomString(lineLength)
	}

	return stream
}

// Helper function to generate a random string
func generateRandomString(length int) []byte {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return b
}

func BenchmarkGeneratorDecoderOTLP(b *testing.B) {
	traceBytes := marshalBatches(b, []*v1.ResourceSpans{
		test.MakeBatch(15, []byte("test batch 1")),
		test.MakeBatch(50, []byte("test batch 2")),
		test.MakeBatch(42, []byte("test batch 3")),
	})

	b.ReportAllocs()
	decoder := NewOTLPDecoder()

	b.ResetTimer()
	for b.Loop() {
		iterator, err := decoder.Decode(traceBytes)
		require.NoError(b, err)
		for range iterator { // nolint:revive // we want to run the side effects of ranging itself
		}
	}
}

func marshalBatches(t testing.TB, batches []*v1.ResourceSpans) []byte {
	t.Helper()

	trace := tempopb.Trace{ResourceSpans: batches}

	m, err := trace.Marshal()
	require.NoError(t, err)

	return m
}

func decodeSinglePushSpans(t testing.TB, decoder GeneratorCodec, data []byte) *tempopb.PushSpansRequest {
	t.Helper()

	iterator, err := decoder.Decode(data)
	require.NoError(t, err)

	var got []*tempopb.PushSpansRequest
	for req, err := range iterator {
		require.NoError(t, err)
		got = append(got, req)
	}
	require.Len(t, got, 1)

	return got[0]
}

func BenchmarkGeneratorDecoderPushBytes(b *testing.B) {
	stream := generateRequest(1000, 200)
	traceBytes, err := stream.Marshal()
	require.NoError(b, err)

	b.ReportAllocs()
	decoder := NewPushBytesDecoder()

	b.ResetTimer()
	for b.Loop() {
		iterator, err := decoder.Decode(traceBytes)
		require.NoError(b, err)
		for range iterator { // nolint:revive // we want to run the side effects of ranging itself
		}
	}
}

// Original implementation without clear() for comparison
func encoderPoolPutOriginal(req *tempopb.PushBytesRequest) {
	req.Traces = req.Traces[:0]
	req.Ids = req.Ids[:0]
	req.SkipMetricsGeneration = false
	encoderPool.Put(req)
}

// Benchmark with different request sizes to see scaling behavior
func BenchmarkEncoderPoolPutDifferentSizes(b *testing.B) {
	sizes := []struct {
		name    string
		entries int
		length  int
	}{
		{"Small", 10, 50},
		{"Medium", 100, 200},
		{"Large", 1000, 500},
	}

	for _, size := range sizes {
		req := generateRequest(size.entries, size.length)

		b.Run(size.name+"_Original", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				pooled := encoderPool.Get().(*tempopb.PushBytesRequest)
				pooled.Traces = append(pooled.Traces, req.Traces...)
				pooled.Ids = append(pooled.Ids, req.Ids...)
				encoderPoolPutOriginal(pooled)
			}
		})

		b.Run(size.name+"_WithClear", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				pooled := encoderPool.Get().(*tempopb.PushBytesRequest)
				pooled.Traces = append(pooled.Traces, req.Traces...)
				pooled.Ids = append(pooled.Ids, req.Ids...)
				encoderPoolPut(pooled)
			}
		})
	}
}
