package model

import (
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
	for _, b := range t2.ResourceSpans {
		switch rand.Int() % 3 {
		case 0:
			t2a.ResourceSpans = append(t2a.ResourceSpans, b)
		case 1:
			t2b.ResourceSpans = append(t2b.ResourceSpans, b)
		case 2:
			t2c.ResourceSpans = append(t2c.ResourceSpans, b)
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
			traces:   [][]byte{mustMarshalToObject(t1, CurrentEncoding), mustMarshalToObject(t1, CurrentEncoding)},
			expected: t1,
		},
		{
			name:           "3 traces",
			traces:         [][]byte{mustMarshalToObject(t2a, CurrentEncoding), mustMarshalToObject(t2b, CurrentEncoding), mustMarshalToObject(t2c, CurrentEncoding)},
			expected:       t2,
			expectCombined: true,
		},
		{
			name:     "1 trace",
			traces:   [][]byte{mustMarshalToObject(t1, CurrentEncoding)},
			expected: t1,
		},
		{
			name:           "nil trace",
			traces:         [][]byte{mustMarshalToObject(t1, CurrentEncoding), nil},
			expected:       t1,
			expectCombined: true,
		},
		{
			name:           "nil trace 2",
			traces:         [][]byte{nil, mustMarshalToObject(t1, CurrentEncoding)},
			expected:       t1,
			expectCombined: true,
		},
		{
			name:        "bad trace",
			traces:      [][]byte{mustMarshalToObject(t1, CurrentEncoding), {0x01, 0x02}},
			expectError: true,
		},
		{
			name:        "bad trace 2",
			traces:      [][]byte{{0x01, 0x02}, mustMarshalToObject(t1, CurrentEncoding)},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, combined, err := StaticCombiner.Combine(CurrentEncoding, tt.traces...)
			assert.Equal(t, tt.expectCombined, combined)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			if tt.expected != nil {
				expected := mustMarshalToObject(tt.expected, CurrentEncoding)
				assert.Equal(t, expected, actual)
			}
		})
	}
}

func mustMarshalToObject(trace *tempopb.Trace, encoding string) []byte {
	return mustMarshalToObjectWithRange(trace, encoding, 0, 0)
}

func mustMarshalToObjectWithRange(trace *tempopb.Trace, encoding string, start, end uint32) []byte {
	b := MustNewSegmentDecoder(encoding)
	batch, err := b.PrepareForWrite(trace, start, end)
	if err != nil {
		panic(err)
	}

	obj, err := b.ToObject([][]byte{batch})
	if err != nil {
		panic(err)
	}

	return obj
}
