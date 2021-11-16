package model

import (
	"fmt"
	"hash/fnv"
	"math/rand"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCombineTraceBytes(t *testing.T) {
	t1 := test.MakeTrace(10, []byte{0x01, 0x02})
	t2 := test.MakeTrace(10, []byte{0x01, 0x03})

	SortTrace(t1)
	SortTrace(t2)

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

	tests := []struct {
		name           string
		trace1         *tempopb.Trace
		trace2         *tempopb.Trace
		expected       *tempopb.Trace
		expectError    bool
		expectCombined bool
	}{
		{
			name:     "same trace",
			trace1:   t1,
			trace2:   t1,
			expected: t1,
		},
		{
			name:        "t2 is bad",
			trace1:      t1,
			trace2:      nil,
			expected:    nil,
			expectError: true,
		},
		{
			name:        "t1 is bad",
			trace1:      nil,
			trace2:      t2,
			expected:    nil,
			expectError: true,
		},
		{
			name:           "combine trace",
			trace1:         t2a,
			trace2:         t2b,
			expected:       t2,
			expectCombined: true,
		},
		{
			name:        "both bad",
			trace1:      nil,
			trace2:      nil,
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		for _, enc1 := range allEncodings {
			for _, enc2 := range allEncodings {
				t.Run(fmt.Sprintf("%s:%s:%s", tt.name, enc1, enc2), func(t *testing.T) {
					var b1 []byte
					var b2 []byte
					if tt.trace1 != nil { // nil means substitute garbage data
						b1 = mustMarshal(tt.trace1, enc1)
					} else {
						b1 = []byte{0x01, 0x02}
					}
					if tt.trace2 != nil { // nil means substitute garbage data
						b2 = mustMarshal(tt.trace2, enc2)
					} else {
						b2 = []byte{0x01, 0x02, 0x03}
					}

					// CombineTraceBytes()
					actual, combined, err := CombineTraceBytes(b1, b2, enc1, enc2)
					if enc1 == enc2 {
						assert.Equal(t, tt.expectCombined, combined)
					}
					if tt.expectError {
						require.Error(t, err)
					} else {
						require.NoError(t, err)
					}
					if tt.expected != nil {
						expected := mustMarshal(tt.expected, enc1)
						assert.Equal(t, expected, actual)
					}
				})
			}
		}
	}
}

func TestCombine(t *testing.T) {
	t1 := test.MakeTrace(10, []byte{0x01, 0x02})
	t2 := test.MakeTrace(10, []byte{0x01, 0x03})

	SortTrace(t1)
	SortTrace(t2)

	// split t2 into two traces
	t2a := &tempopb.Trace{}
	t2b := &tempopb.Trace{}
	t2c := &tempopb.Trace{}
	for _, b := range t2.Batches {
		switch rand.Int() % 3 {
		case 0:
			t2a.Batches = append(t2a.Batches, b)
		case 1:
			t2b.Batches = append(t2b.Batches, b)
		case 2:
			t2c.Batches = append(t2c.Batches, b)
		}
	}

	tests := []struct {
		name           string
		traces         [][]byte
		expected       *tempopb.Trace
		expectError    bool
		expectCombined bool
	}{
		{
			name:        "no traces",
			expectError: true,
		},
		{
			name:     "same trace",
			traces:   [][]byte{mustMarshal(t1, CurrentEncoding), mustMarshal(t1, CurrentEncoding)},
			expected: t1,
		},
		{
			name:           "3 traces",
			traces:         [][]byte{mustMarshal(t2a, CurrentEncoding), mustMarshal(t2b, CurrentEncoding), mustMarshal(t2c, CurrentEncoding)},
			expected:       t2,
			expectCombined: true,
		},
		{
			name:     "1 trace",
			traces:   [][]byte{mustMarshal(t1, CurrentEncoding)},
			expected: t1,
		},
		{
			name:           "nil trace",
			traces:         [][]byte{mustMarshal(t1, CurrentEncoding), nil},
			expected:       t1,
			expectCombined: true,
		},
		{
			name:           "nil trace 2",
			traces:         [][]byte{nil, mustMarshal(t1, CurrentEncoding)},
			expected:       t1,
			expectCombined: true,
		},
		{
			name:        "bad trace",
			traces:      [][]byte{mustMarshal(t1, CurrentEncoding), {0x01, 0x02}},
			expectError: true,
		},
		{
			name:        "bad trace 2",
			traces:      [][]byte{{0x01, 0x02}, mustMarshal(t1, CurrentEncoding)},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s", tt.name), func(t *testing.T) {
			traceBytes := [][]byte{}

			for _, trace := range tt.traces {
				traceBytes = append(traceBytes, trace)
			}

			actual, combined, err := ObjectCombiner.Combine(CurrentEncoding, traceBytes...)
			assert.Equal(t, tt.expectCombined, combined)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			if tt.expected != nil {
				expected := mustMarshal(tt.expected, CurrentEncoding)
				assert.Equal(t, expected, actual)
			}
		})
	}
}

func TestCombineNils(t *testing.T) {
	test := test.MakeTrace(1, nil)
	SortTrace(test)

	for _, enc1 := range allEncodings {
		for _, enc2 := range allEncodings {
			t.Run(fmt.Sprintf("%s:%s", enc1, enc2), func(t *testing.T) {
				// both nil
				actualBytes, _, err := CombineTraceBytes(nil, nil, enc1, enc2)
				require.NoError(t, err)
				assert.Equal(t, []byte(nil), actualBytes)

				testBytes1, err := marshal(test, enc1)
				require.NoError(t, err)
				testBytes2, err := marshal(test, enc2)
				require.NoError(t, err)

				// objB nil
				actualBytes, _, err = CombineTraceBytes(testBytes1, nil, enc1, enc2)
				require.NoError(t, err)

				actual, err := Unmarshal(actualBytes, enc1)
				require.NoError(t, err)
				assert.Equal(t, test, actual)

				// objA nil
				actualBytes, _, err = CombineTraceBytes(nil, testBytes2, enc1, enc2)
				require.NoError(t, err)

				actual, err = Unmarshal(actualBytes, enc1)
				require.NoError(t, err)
				assert.Equal(t, test, actual)
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

func mustMarshal(trace *tempopb.Trace, encoding string) []byte {
	b, err := marshal(trace, encoding)
	if err != nil {
		panic(err)
	}

	return b
}
