package delta

import (
	"fmt"
	"io"
	"math"

	"github.com/segmentio/parquet-go/encoding"
)

type LengthByteArrayEncoder struct {
	encoding.NotSupportedEncoder
	binpack BinaryPackedEncoder
	lengths []int32
}

func NewLengthByteArrayEncoder(w io.Writer) *LengthByteArrayEncoder {
	e := &LengthByteArrayEncoder{lengths: make([]int32, defaultBufferSize/4)}
	e.Reset(w)
	return e
}

func (e *LengthByteArrayEncoder) Reset(w io.Writer) {
	e.binpack.Reset(w)
}

func (e *LengthByteArrayEncoder) EncodeByteArray(data encoding.ByteArrayList) (err error) {
	e.lengths = e.lengths[:0]

	data.Range(func(value []byte) bool {
		if len(value) > math.MaxInt32 {
			err = fmt.Errorf("DELTA_LENGTH_BYTE_ARRAY: byte array of length %d is too large to be encoded", len(value))
			return false
		}
		e.lengths = append(e.lengths, int32(len(value)))
		return true
	})
	if err != nil {
		return err
	}
	if err = e.binpack.EncodeInt32(e.lengths); err != nil {
		return err
	}

	data.Range(func(value []byte) bool {
		_, err = e.binpack.writer.Write(value)
		return err == nil
	})
	return err
}
