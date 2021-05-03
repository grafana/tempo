package model

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCombine(t *testing.T) {
	t1 := test.MakeTrace(10, []byte{0x01, 0x02})
	t2 := test.MakeTrace(10, []byte{0x01, 0x03})

	SortTrace(t1)
	SortTrace(t2)

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
		trace1    []byte
		trace2    []byte
		expected  []byte
		errString string
	}{
		{
			trace1:   b1,
			trace2:   b1,
			expected: b1,
		},
		{
			trace1:    b1,
			trace2:    []byte{0x01},
			expected:  b1,
			errString: "error unsmarshaling objB: proto: Trace: illegal tag 0 (wire type 1)",
		},
		{
			trace1:    []byte{0x01},
			trace2:    b2,
			expected:  b2,
			errString: "error unsmarshaling objA: proto: Trace: illegal tag 0 (wire type 1)",
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
		{
			trace1:    []byte{0x01},
			trace2:    []byte{0x02},
			expected:  nil,
			errString: "both A and B failed to unmarshal.  returning an empty trace: proto: Trace: illegal tag 0 (wire type 1)",
		},
	}

	for _, tt := range tests {
		actual, _, err := CombineTraceBytes(tt.trace1, tt.trace2)
		if len(tt.errString) > 0 {
			assert.EqualError(t, err, tt.errString)
		} else {
			assert.NoError(t, err)
		}

		if !bytes.Equal(tt.expected, actual) {
			actualTrace := &tempopb.Trace{}
			expectedTrace := &tempopb.Trace{}

			err = proto.Unmarshal(tt.expected, expectedTrace)
			assert.NoError(t, err)
			err = proto.Unmarshal(actual, actualTrace)
			assert.NoError(t, err)

			assert.Equal(t, expectedTrace, actualTrace)
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

func BenchmarkCombineTraces(b *testing.B) {
	t1 := test.MakeTrace(10, []byte{0x01, 0x02})
	t2 := test.MakeTrace(10, []byte{0x01, 0x03})

	b1, err := proto.Marshal(t1)
	assert.NoError(b, err)
	b2, err := proto.Marshal(t2)
	assert.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// nolint:errcheck
		CombineTraceBytes(b1, b2)
	}
}

func BenchmarkCombineTracesIdentical(b *testing.B) {
	t1 := test.MakeTrace(10, []byte{0x01, 0x02})

	b1, err := proto.Marshal(t1)
	assert.NoError(b, err)

	var b2 []byte
	b2 = append(b2, b1...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// nolint:errcheck
		CombineTraceBytes(b1, b2)
	}
}

func BenchmarkCombineTraceProtos(b *testing.B) {
	sizes := []int{1, 10, 1000, 10000, 100000}

	for _, size := range sizes {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			t1 := test.MakeTraceWithSpanCount(1, size, []byte{0x01, 0x02})
			t2 := test.MakeTraceWithSpanCount(1, size, []byte{0x01, 0x03})

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				CombineTraceProtos(t1, t2)
			}
		})
	}
}

func TestSortTrace(t *testing.T) {
	tests := []struct {
		input    *tempopb.Trace
		expected *tempopb.Trace
	}{
		{
			input:    &tempopb.Trace{},
			expected: &tempopb.Trace{},
		},

		{
			input: &tempopb.Trace{
				Batches: []*v1.ResourceSpans{
					{
						InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
							{
								Spans: []*v1.Span{
									{
										StartTimeUnixNano: 2,
									},
								},
							},
						},
					},
					{
						InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
							{
								Spans: []*v1.Span{
									{
										StartTimeUnixNano: 1,
									},
								},
							},
						},
					},
				},
			},
			expected: &tempopb.Trace{
				Batches: []*v1.ResourceSpans{
					{
						InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
							{
								Spans: []*v1.Span{
									{
										StartTimeUnixNano: 1,
									},
								},
							},
						},
					},
					{
						InstrumentationLibrarySpans: []*v1.InstrumentationLibrarySpans{
							{
								Spans: []*v1.Span{
									{
										StartTimeUnixNano: 2,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		SortTrace(tt.input)

		assert.Equal(t, tt.expected, tt.input)
	}
}

func TestUnmarshal(t *testing.T) {
	trace := test.MakeTrace(100, nil)
	bytes, err := proto.Marshal(trace)
	require.NoError(t, err)

	actual, err := Unmarshal(bytes, CurrentEncoding)
	require.NoError(t, err)

	assert.True(t, proto.Equal(trace, actual))
}
