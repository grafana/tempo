package model

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"math/rand"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/assert"
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
		encoding1 string
		trace2    []byte
		encoding2 string
		expected  []byte
		errString string
	}{
		{
			trace1:    b1,
			encoding1: TracePBEncoding,
			trace2:    b1,
			encoding2: TracePBEncoding,
			expected:  b1,
		},
		{
			trace1:    b1,
			encoding1: TracePBEncoding,
			trace2:    []byte{0x01},
			encoding2: TracePBEncoding,
			expected:  b1,
			errString: "error unsmarshaling objB: proto: Trace: illegal tag 0 (wire type 1)",
		},
		{
			trace1:    []byte{0x01},
			encoding1: TracePBEncoding,
			trace2:    b2,
			encoding2: TracePBEncoding,
			expected:  b2,
			errString: "error unsmarshaling objA: proto: Trace: illegal tag 0 (wire type 1)",
		},
		{
			// exactly matching byte slices are not unmarshalled
			trace1:    []byte{0x01, 0x02, 0x03},
			encoding1: TracePBEncoding,
			trace2:    []byte{0x01, 0x02, 0x03},
			encoding2: TracePBEncoding,
			expected:  []byte{0x01, 0x02, 0x03},
		},
		{
			trace1:    b2a,
			encoding1: TracePBEncoding,
			trace2:    b2b,
			encoding2: TracePBEncoding,
			expected:  b2,
		},
		{
			trace1:    []byte{0x01},
			encoding1: TracePBEncoding,
			trace2:    []byte{0x02},
			encoding2: TracePBEncoding,
			expected:  nil,
			errString: "both A and B failed to unmarshal.  returning an empty trace: proto: Trace: illegal tag 0 (wire type 1)",
		},
		{
			// bytes encoding
		},
		{
			// trace/tracebytes encoding
		},
	}

	for _, tt := range tests {
		actual, _, err := CombineTraceBytes(tt.trace1, tt.trace2, tt.encoding1, tt.encoding2)
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
		CombineTraceBytes(b1, b2, "", "")
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
		CombineTraceBytes(b1, b2, "", "")
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

func BenchmarkTokenForID(b *testing.B) {
	h := fnv.New32()
	id := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	buffer := make([]byte, 4)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tokenForID(h, buffer, 0, id)
	}
}
