package model

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCombine(t *testing.T) {
	t1 := test.MakeTrace(10, []byte{0x01, 0x02})
	t2 := test.MakeTrace(10, []byte{0x01, 0x03})

	trace.SortTrace(t1)
	trace.SortTrace(t2)

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
		t.Run(tt.name, func(t *testing.T) {
			actual, combined, err := ObjectCombiner.Combine(CurrentEncoding, tt.traces...)
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

func BenchmarkCombineTraceProtos(b *testing.B) {
	sizes := []int{1, 10, 1000, 10000, 100000}

	for _, size := range sizes {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			t1 := test.MakeTraceWithSpanCount(1, size, []byte{0x01, 0x02})
			t2 := test.MakeTraceWithSpanCount(1, size, []byte{0x01, 0x03})

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				trace.CombineTraceProtos(t1, t2)
			}
		})
	}
}

// nolint:unparam
func mustMarshal(trace *tempopb.Trace, encoding string) []byte {
	d := MustNewDecoder(encoding)
	b, err := d.(encoderDecoder).Marshal(trace)
	if err != nil {
		panic(err)
	}

	return b
}
