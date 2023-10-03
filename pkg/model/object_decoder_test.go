package model

import (
	"errors"
	"math/rand"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/model/decoder"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
)

func TestObjectDecoderMarshalUnmarshal(t *testing.T) {
	empty := &tempopb.Trace{}

	for _, e := range AllEncodings {
		t.Run(e, func(t *testing.T) {
			encoding, err := NewObjectDecoder(e)
			require.NoError(t, err)

			// random trace
			trace := test.MakeTrace(100, nil)
			bytes := mustMarshalToObject(trace, e)

			actual, err := encoding.PrepareForRead(bytes)
			require.NoError(t, err)
			assert.True(t, proto.Equal(trace, actual))

			// nil trace
			actual, err = encoding.PrepareForRead(nil)
			assert.NoError(t, err)
			assert.True(t, proto.Equal(empty, actual))

			// empty byte slice
			actual, err = encoding.PrepareForRead([]byte{})
			assert.NoError(t, err)
			assert.True(t, proto.Equal(empty, actual))
		})
	}
}

func TestCombines(t *testing.T) {
	t1 := test.MakeTrace(10, []byte{0x01, 0x02})
	t2 := test.MakeTrace(10, []byte{0x01, 0x03})

	trace.SortTrace(t1)
	trace.SortTrace(t2)

	// split t2 into 3 traces
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

	for _, e := range AllEncodings {
		tests := []struct {
			name          string
			traces        [][]byte
			expected      *tempopb.Trace
			expectedStart uint32
			expectedEnd   uint32
			expectError   bool
		}{
			{
				name:          "one trace",
				traces:        [][]byte{mustMarshalToObjectWithRange(t1, e, 10, 20)},
				expected:      t1,
				expectedStart: 10,
				expectedEnd:   20,
			},
			{
				name:          "same trace - replace end",
				traces:        [][]byte{mustMarshalToObjectWithRange(t1, e, 10, 20), mustMarshalToObjectWithRange(t1, e, 30, 40)},
				expected:      t1,
				expectedStart: 10,
				expectedEnd:   40,
			},
			{
				name:          "same trace - replace start",
				traces:        [][]byte{mustMarshalToObjectWithRange(t1, e, 10, 20), mustMarshalToObjectWithRange(t1, e, 5, 15)},
				expected:      t1,
				expectedStart: 5,
				expectedEnd:   20,
			},
			{
				name:          "same trace - replace both",
				traces:        [][]byte{mustMarshalToObjectWithRange(t1, e, 10, 20), mustMarshalToObjectWithRange(t1, e, 5, 30)},
				expected:      t1,
				expectedStart: 5,
				expectedEnd:   30,
			},
			{
				name:          "same trace - replace neither",
				traces:        [][]byte{mustMarshalToObjectWithRange(t1, e, 10, 20), mustMarshalToObjectWithRange(t1, e, 12, 14)},
				expected:      t1,
				expectedStart: 10,
				expectedEnd:   20,
			},
			{
				name:          "3 traces",
				traces:        [][]byte{mustMarshalToObjectWithRange(t2a, e, 10, 20), mustMarshalToObjectWithRange(t2b, e, 5, 15), mustMarshalToObjectWithRange(t2c, e, 20, 30)},
				expected:      t2,
				expectedStart: 5,
				expectedEnd:   30,
			},
			{
				name:          "nil trace",
				traces:        [][]byte{mustMarshalToObjectWithRange(t1, e, 10, 20), nil},
				expected:      t1,
				expectedStart: 10,
				expectedEnd:   20,
			},
			{
				name:          "nil trace 2",
				traces:        [][]byte{nil, mustMarshalToObjectWithRange(t1, e, 10, 20)},
				expected:      t1,
				expectedStart: 10,
				expectedEnd:   20,
			},
			{
				name:        "bad trace",
				traces:      [][]byte{mustMarshalToObjectWithRange(t1, e, 10, 20), {0x01, 0x02}},
				expectError: true,
			},
			{
				name:        "bad trace 2",
				traces:      [][]byte{{0x01, 0x02}, mustMarshalToObjectWithRange(t1, e, 10, 20)},
				expectError: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name+"-"+e, func(t *testing.T) {
				d := MustNewObjectDecoder(e)
				actualBytes, err := d.Combine(tt.traces...)

				if tt.expectError {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}

				if tt.expected != nil {
					actual, err := d.PrepareForRead(actualBytes)
					require.NoError(t, err)
					assert.Equal(t, tt.expected, actual)

					start, end, err := d.FastRange(actualBytes)
					if errors.Is(err, decoder.ErrUnsupported) {
						return
					}
					require.NoError(t, err)
					assert.Equal(t, tt.expectedStart, start)
					assert.Equal(t, tt.expectedEnd, end)
				}
			})
		}
	}
}
