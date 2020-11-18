package util

import (
	"bytes"
	"math/rand"
	"sort"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/assert"
)

func TestCombine(t *testing.T) {
	t1 := test.MakeTrace(10, []byte{0x01, 0x02})
	t2 := test.MakeTrace(10, []byte{0x01, 0x03})

	b1, err := proto.Marshal(t1)
	assert.NoError(t, err)
	b2, err := proto.Marshal(t2)
	assert.NoError(t, err)

	// split t2 into two traces
	t2a := &tempopb.Trace{}
	t2b := &tempopb.Trace{}
	for _, b := range t2.Batches {
		if rand.Int()%2 == 0 {
			t2a.Batches = append(t2a.Batches, b)
		} else {
			t2b.Batches = append(t2b.Batches, b)
		}
	}

	b2a, err := proto.Marshal(t2a)
	assert.NoError(t, err)
	b2b, err := proto.Marshal(t2b)
	assert.NoError(t, err)

	tests := []struct {
		trace1   []byte
		trace2   []byte
		expected []byte
	}{
		{
			trace1:   b1,
			trace2:   b1,
			expected: b1,
		},
		{
			trace1:   b1,
			trace2:   []byte{0x01},
			expected: b1,
		},
		{
			trace1:   []byte{0x01},
			trace2:   b2,
			expected: b2,
		},
		{
			trace1:   []byte{0x01, 0x02, 0x03},
			trace2:   []byte{0x01, 0x02, 0x03},
			expected: []byte{0x01, 0x02, 0x03},
		},
		{
			trace1:   b2a,
			trace2:   b2b,
			expected: b2,
		},
	}

	for _, tt := range tests {
		actual := CombineTraces(tt.trace1, tt.trace2)

		if !bytes.Equal(tt.expected, actual) {
			actualTrace := &tempopb.Trace{}
			expectedTrace := &tempopb.Trace{}

			err = proto.Unmarshal(tt.expected, expectedTrace)
			assert.NoError(t, err)
			err = proto.Unmarshal(actual, actualTrace)
			assert.NoError(t, err)

			sortTrace(actualTrace)
			sortTrace(expectedTrace)

			assert.Equal(t, expectedTrace, actualTrace)
		}
	}
}

func sortTrace(t *tempopb.Trace) {
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

// logic of actually combining traces should be tested above.  focusing on the spancounts here
func TestCombineProtos(t *testing.T) {
	sameTrace := test.MakeTraceWithSpanCount(10, 10, []byte{0x01, 0x03})

	tests := []struct {
		traceA        *tempopb.Trace
		traceB        *tempopb.Trace
		expectedA     int
		expectedB     int
		expectedTotal int
	}{
		{
			traceA:        nil,
			traceB:        test.MakeTraceWithSpanCount(10, 10, []byte{0x01, 0x03}),
			expectedA:     0,
			expectedB:     -1,
			expectedTotal: -1,
		},
		{
			traceA:        test.MakeTraceWithSpanCount(10, 10, []byte{0x01, 0x03}),
			traceB:        nil,
			expectedA:     -1,
			expectedB:     0,
			expectedTotal: -1,
		},
		{
			traceA:        test.MakeTraceWithSpanCount(10, 10, []byte{0x01, 0x03}),
			traceB:        test.MakeTraceWithSpanCount(10, 10, []byte{0x01, 0x01}),
			expectedA:     100,
			expectedB:     100,
			expectedTotal: 200,
		},
		{
			traceA:        sameTrace,
			traceB:        sameTrace,
			expectedA:     100,
			expectedB:     100,
			expectedTotal: 100,
		},
	}

	for _, tt := range tests {
		_, actualA, actualB, actualTotal := CombineTraceProtos(tt.traceA, tt.traceB)

		assert.Equal(t, tt.expectedA, actualA)
		assert.Equal(t, tt.expectedB, actualB)
		assert.Equal(t, tt.expectedTotal, actualTotal)
	}
}
