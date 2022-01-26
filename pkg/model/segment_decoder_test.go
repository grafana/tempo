package model

import (
	"math/rand"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/model/decoder"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/require"
)

func TestSegmentDecoderToObjectDecoder(t *testing.T) {
	for _, e := range AllEncodings {
		t.Run(e, func(t *testing.T) {
			objectDecoder, err := NewObjectDecoder(e)
			require.NoError(t, err)

			segmentDecoder, err := NewSegmentDecoder(e)
			require.NoError(t, err)

			// random trace
			trace := test.MakeTrace(100, nil)

			segment, err := segmentDecoder.PrepareForWrite(trace, 0, 0)
			require.NoError(t, err)

			// segment prepareforread
			actual, err := segmentDecoder.PrepareForRead([][]byte{segment})
			require.NoError(t, err)
			require.True(t, proto.Equal(trace, actual))

			// convert to object
			object, err := segmentDecoder.ToObject([][]byte{segment})
			require.NoError(t, err)

			actual, err = objectDecoder.PrepareForRead(object)
			require.NoError(t, err)
			require.True(t, proto.Equal(trace, actual))
		})
	}
}

func TestSegmentDecoderToObjectDecoderRange(t *testing.T) {
	for _, e := range AllEncodings {
		t.Run(e, func(t *testing.T) {
			start := rand.Uint32()
			end := rand.Uint32()

			objectDecoder, err := NewObjectDecoder(e)
			require.NoError(t, err)

			segmentDecoder, err := NewSegmentDecoder(e)
			require.NoError(t, err)

			// random trace
			trace := test.MakeTrace(100, nil)

			segment, err := segmentDecoder.PrepareForWrite(trace, start, end)
			require.NoError(t, err)

			// convert to object
			object, err := segmentDecoder.ToObject([][]byte{segment})
			require.NoError(t, err)

			// test range
			actualStart, actualEnd, err := objectDecoder.FastRange(object)
			if err == decoder.ErrUnsupported {
				return
			}

			require.NoError(t, err)
			require.Equal(t, start, actualStart)
			require.Equal(t, end, actualEnd)
		})
	}
}
