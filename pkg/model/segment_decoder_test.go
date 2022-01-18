package model

import (
	"math/rand"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/model/decoder"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/require"
)

func TestBatchDecoderToObjectDecoder(t *testing.T) {
	for _, e := range AllEncodings {
		t.Run(e, func(t *testing.T) {
			objectDecoder, err := NewObjectDecoder(e)
			require.NoError(t, err)

			batchDecoder, err := NewSegmentDecoder(e)
			require.NoError(t, err)

			// random trace
			trace := test.MakeTrace(100, nil)

			batch, err := batchDecoder.PrepareForWrite(trace, 0, 0)
			require.NoError(t, err)

			// batch prepareforread
			actual, err := batchDecoder.PrepareForRead([][]byte{batch})
			require.NoError(t, err)
			require.True(t, proto.Equal(trace, actual))

			// convert to object
			object, err := batchDecoder.ToObject([][]byte{batch})
			require.NoError(t, err)

			actual, err = objectDecoder.PrepareForRead(object)
			require.NoError(t, err)
			require.True(t, proto.Equal(trace, actual))
		})
	}
}

func TestBatchDecoderToObjectDecoderRange(t *testing.T) {
	for _, e := range AllEncodings {
		t.Run(e, func(t *testing.T) {
			start := rand.Uint32()
			end := rand.Uint32()

			objectDecoder, err := NewObjectDecoder(e)
			require.NoError(t, err)

			batchDecoder, err := NewSegmentDecoder(e)
			require.NoError(t, err)

			// random trace
			trace := test.MakeTrace(100, nil)

			batch, err := batchDecoder.PrepareForWrite(trace, start, end)
			require.NoError(t, err)

			// convert to object
			object, err := batchDecoder.ToObject([][]byte{batch})
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
