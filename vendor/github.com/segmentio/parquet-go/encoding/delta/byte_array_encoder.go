package delta

import (
	"bytes"
	"fmt"
	"io"
	"math"

	"github.com/segmentio/parquet-go/encoding"
)

type ByteArrayEncoder struct {
	encoding.NotSupportedEncoder
	deltas   BinaryPackedEncoder
	arrays   LengthByteArrayEncoder
	prefixes []int32
	suffixes encoding.ByteArrayList
}

func NewByteArrayEncoder(w io.Writer) *ByteArrayEncoder {
	e := &ByteArrayEncoder{prefixes: make([]int32, defaultBufferSize/4)}
	e.Reset(w)
	return e
}

func (e *ByteArrayEncoder) Reset(w io.Writer) {
	e.deltas.Reset(w)
	e.arrays.Reset(w)
	e.prefixes = e.prefixes[:0]
	e.suffixes.Reset()
}

func (e *ByteArrayEncoder) EncodeByteArray(data encoding.ByteArrayList) error {
	return e.encode(data.Len(), data.Index)
}

func (e *ByteArrayEncoder) EncodeFixedLenByteArray(size int, data []byte) error {
	if size <= 0 {
		return fmt.Errorf("DELTA_BYTE_ARRAY: %w: size of encoded FIXED_LEN_BYTE_ARRAY must be positive", encoding.ErrInvalidArgument)
	}
	return e.encode(len(data)/size, func(i int) []byte { return data[i*size : (i+1)*size] })
}

func (e *ByteArrayEncoder) encode(count int, valueAt func(int) []byte) error {
	lastValue := ([]byte)(nil)
	e.prefixes = e.prefixes[:0]
	e.suffixes.Reset()

	for i := 0; i < count; i++ {
		value := valueAt(i)
		if len(value) > math.MaxInt32 {
			return fmt.Errorf("DELTA_BYTE_ARRAY: byte array of length %d is too large to be encoded", len(value))
		}
		n := prefixLength(lastValue, value)
		e.prefixes = append(e.prefixes, int32(n))
		e.suffixes.Push(value[n:])
		lastValue = value
	}

	if err := e.deltas.EncodeInt32(e.prefixes); err != nil {
		return err
	}
	return e.arrays.EncodeByteArray(e.suffixes)
}

func prefixLength(base, data []byte) int {
	return binarySearchPrefixLength(len(base)/2, base, data)
}

func binarySearchPrefixLength(max int, base, data []byte) int {
	for len(base) > 0 {
		if bytes.HasPrefix(data, base[:max]) {
			if max == len(base) {
				return max
			}
			max += (len(base)-max)/2 + 1
		} else {
			base = base[:max-1]
			max /= 2
		}
	}
	return 0
}
