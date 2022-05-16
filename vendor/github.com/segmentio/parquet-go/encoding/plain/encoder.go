package plain

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/segmentio/parquet-go/deprecated"
	"github.com/segmentio/parquet-go/encoding"
	"github.com/segmentio/parquet-go/encoding/rle"
	"github.com/segmentio/parquet-go/internal/bits"
)

type Encoder struct {
	encoding.NotSupportedEncoder
	writer io.Writer
	buffer [4]byte
	rle    *rle.Encoder
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{writer: w}
}

func (e *Encoder) Reset(w io.Writer) {
	e.writer = w

	if e.rle != nil {
		e.rle.Reset(w)
	}
}

func (e *Encoder) EncodeBoolean(data []bool) error {
	if e.rle == nil {
		e.rle = rle.NewEncoder(e.writer)
	}
	return e.rle.EncodeBoolean(data)
}

func (e *Encoder) EncodeInt32(data []int32) error {
	_, err := e.writer.Write(bits.Int32ToBytes(data))
	return err
}

func (e *Encoder) EncodeInt64(data []int64) error {
	_, err := e.writer.Write(bits.Int64ToBytes(data))
	return err
}

func (e *Encoder) EncodeInt96(data []deprecated.Int96) error {
	_, err := e.writer.Write(deprecated.Int96ToBytes(data))
	return err
}

func (e *Encoder) EncodeFloat(data []float32) error {
	_, err := e.writer.Write(bits.Float32ToBytes(data))
	return err
}

func (e *Encoder) EncodeDouble(data []float64) error {
	_, err := e.writer.Write(bits.Float64ToBytes(data))
	return err
}

func (e *Encoder) EncodeByteArray(data encoding.ByteArrayList) (err error) {
	data.Range(func(value []byte) bool {
		binary.LittleEndian.PutUint32(e.buffer[:4], uint32(len(value)))
		if _, err = e.writer.Write(e.buffer[:4]); err != nil {
			return false
		}
		if _, err = e.writer.Write(value); err != nil {
			return false
		}
		return true
	})
	return err
}

func (e *Encoder) EncodeFixedLenByteArray(size int, data []byte) error {
	if size <= 0 {
		return fmt.Errorf("PLAIN: %w: size of encoded FIXED_LEN_BYTE_ARRAY must be positive", encoding.ErrInvalidArgument)
	}

	if (len(data) % size) != 0 {
		return fmt.Errorf("PLAIN: %w: length of encoded FIXED_LEN_BYTE_ARRAY must be a multiple of its size: size=%d length=%d", encoding.ErrInvalidArgument, size, len(data))
	}

	_, err := e.writer.Write(data)
	return err
}

func (e *Encoder) SetBitWidth(bitWidth int) {}
