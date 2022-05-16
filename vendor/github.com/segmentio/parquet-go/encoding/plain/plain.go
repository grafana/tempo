// Package plain implements the PLAIN parquet encoding.
//
// https://github.com/apache/parquet-format/blob/master/Encodings.md#plain-plain--0
package plain

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/segmentio/parquet-go/deprecated"
	"github.com/segmentio/parquet-go/encoding"
	"github.com/segmentio/parquet-go/format"
)

const (
	ByteArrayLengthSize = 4
)

type Encoding struct {
}

func (e *Encoding) Encoding() format.Encoding {
	return format.Plain
}

func (e *Encoding) CanEncode(format.Type) bool {
	return true
}

func (e *Encoding) NewDecoder(r io.Reader) encoding.Decoder {
	return NewDecoder(r)
}

func (e *Encoding) NewEncoder(w io.Writer) encoding.Encoder {
	return NewEncoder(w)
}

func (e *Encoding) String() string {
	return "PLAIN"
}

func Boolean(v bool) []byte { return AppendBoolean(nil, v) }

func Int32(v int32) []byte { return AppendInt32(nil, v) }

func Int64(v int64) []byte { return AppendInt64(nil, v) }

func Int96(v deprecated.Int96) []byte { return AppendInt96(nil, v) }

func Float(v float32) []byte { return AppendFloat(nil, v) }

func Double(v float64) []byte { return AppendDouble(nil, v) }

func ByteArray(v []byte) []byte { return AppendByteArray(nil, v) }

func AppendBoolean(b []byte, v bool) []byte {
	if v {
		b = append(b, 1)
	} else {
		b = append(b, 0)
	}
	return b
}

func AppendInt32(b []byte, v int32) []byte {
	x := [4]byte{}
	binary.LittleEndian.PutUint32(x[:], uint32(v))
	return append(b, x[:]...)
}

func AppendInt64(b []byte, v int64) []byte {
	x := [8]byte{}
	binary.LittleEndian.PutUint64(x[:], uint64(v))
	return append(b, x[:]...)
}

func AppendInt96(b []byte, v deprecated.Int96) []byte {
	x := [12]byte{}
	binary.LittleEndian.PutUint32(x[0:4], v[0])
	binary.LittleEndian.PutUint32(x[4:8], v[1])
	binary.LittleEndian.PutUint32(x[8:12], v[2])
	return append(b, x[:]...)
}

func AppendFloat(b []byte, v float32) []byte {
	x := [4]byte{}
	binary.LittleEndian.PutUint32(x[:], math.Float32bits(v))
	return append(b, x[:]...)
}

func AppendDouble(b []byte, v float64) []byte {
	x := [8]byte{}
	binary.LittleEndian.PutUint64(x[:], math.Float64bits(v))
	return append(b, x[:]...)
}

func AppendByteArray(b, v []byte) []byte {
	i := len(b)
	j := i + 4
	b = append(b, 0, 0, 0, 0)
	b = append(b, v...)
	PutByteArrayLength(b[i:j:j], len(v))
	return b
}

func PutByteArrayLength(b []byte, n int) {
	binary.LittleEndian.PutUint32(b, uint32(n))
}

func RangeByteArrays(b []byte, do func([]byte) error) (err error) {
	for len(b) > 0 {
		var v []byte
		if v, b, err = NextByteArray(b); err != nil {
			return err
		}
		if err = do(v); err != nil {
			return err
		}
	}
	return nil
}

func NextByteArray(b []byte) (v, r []byte, err error) {
	if len(b) < 4 {
		return nil, b, fmt.Errorf("input of length %d is too short to contain a PLAIN encoded byte array: %w", len(b), io.ErrUnexpectedEOF)
	}
	n := 4 + int(binary.LittleEndian.Uint32(b))
	if n > len(b) {
		return nil, b, fmt.Errorf("input of length %d is too short to contain a PLAIN encoded byte array of length %d: %w", len(b)-4, n-4, io.ErrUnexpectedEOF)
	}
	return b[4:n], b[n:], nil
}
