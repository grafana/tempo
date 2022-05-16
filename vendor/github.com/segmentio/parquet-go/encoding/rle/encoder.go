package rle

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/segmentio/parquet-go/encoding"
	"github.com/segmentio/parquet-go/internal/bits"
)

type Encoder struct {
	encoding.NotSupportedEncoder
	writer    io.Writer
	bitWidth  uint
	buffer    [64]byte
	runLength runLengthRunEncoder
	bitPack   bitPackRunEncoder
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{writer: w}
}

func (e *Encoder) Write(b []byte) (int, error) {
	return e.writer.Write(b)
}

func (e *Encoder) WriteByte(b byte) error {
	e.buffer[0] = b
	_, err := e.Write(e.buffer[:1])
	return err
}

func (e *Encoder) WriteUvarint(u uint64) (int, error) {
	n := binary.PutUvarint(e.buffer[:], u)
	return e.Write(e.buffer[:n])
}

func (e *Encoder) BitWidth() int {
	return int(e.bitWidth)
}

func (e *Encoder) SetBitWidth(bitWidth int) {
	e.bitWidth = uint(bitWidth)
}

func (e *Encoder) Reset(w io.Writer) {
	e.writer = w
}

func (e *Encoder) EncodeBoolean(data []bool) error {
	// When encoding booleans, the BIT_PACKED encoding is used without the
	// varint header.
	e.bitPack.reset(e.writer, 1)
	bytes := bits.BoolToBytes(data)
	int8s := bits.BytesToInt8(bytes)
	e.bitPack.encodeInt8(int8s, 1)
	return e.bitPack.flush()
}

func (e *Encoder) EncodeInt8(data []int8) error {
	return e.encode(bits.Int8ToBytes(data), e.bitWidth, 8)
}

func (e *Encoder) EncodeInt16(data []int16) error {
	return e.encode(bits.Int16ToBytes(data), e.bitWidth, 16)
}

func (e *Encoder) EncodeInt32(data []int32) error {
	return e.encode(bits.Int32ToBytes(data), e.bitWidth, 32)
}

func (e *Encoder) EncodeInt64(data []int64) error {
	return e.encode(bits.Int64ToBytes(data), e.bitWidth, 64)
}

func (e *Encoder) encode(data []byte, dstWidth, srcWidth uint) error {
	if dstWidth == 0 {
		return fmt.Errorf("the destination bit-width must be configured on a RLE encoder before writing %d bits integer values", srcWidth)
	}

	wordSize := uint(bits.ByteCount(srcWidth))
	eightWordSize := 8 * wordSize
	i := uint(0)
	n := uint(len(data))
	pattern := e.buffer[:eightWordSize]

	for i < n {
		j := i
		k := i + eightWordSize
		fill(pattern, data[i:i+wordSize])

		for k <= n && !bytes.Equal(data[j:k], pattern) {
			j += eightWordSize
			k += eightWordSize
		}

		if i < j {
			if err := e.encodeBitPack(data[i:j], dstWidth, srcWidth); err != nil {
				return err
			}
		} else {
			if k <= n {
				j += eightWordSize
				k += eightWordSize
			}

			for k <= n && bytes.Equal(data[j:k], pattern) {
				j += eightWordSize
				k += eightWordSize
			}

			k = j + wordSize
			for k <= n && bytes.Equal(data[j:k], pattern[:wordSize]) {
				j += wordSize
				k += wordSize
			}

			if i < j {
				if err := e.encodeRunLength(data[i:j], dstWidth, srcWidth); err != nil {
					return err
				}
			}
		}

		i = j
	}

	return nil
}

func (e *Encoder) encodeBitPack(run []byte, dstWidth, srcWidth uint) error {
	if _, err := e.WriteUvarint((uint64(len(run)/(8*bits.ByteCount(srcWidth))) << 1) | 1); err != nil {
		return err
	}
	e.bitPack.reset(e.writer, dstWidth)
	return e.bitPack.encode(run, srcWidth)
}

func (e *Encoder) encodeRunLength(run []byte, dstWidth, srcWidth uint) error {
	if _, err := e.WriteUvarint(uint64(len(run)/bits.ByteCount(srcWidth)) << 1); err != nil {
		return err
	}
	e.runLength.reset(e.writer, dstWidth)
	return e.runLength.encode(run, srcWidth)
}

func fill(b, v []byte) int {
	n := copy(b, v)

	for i := n; i < len(b); {
		n += copy(b[i:], b[:i])
		i *= 2
	}

	return n
}

var (
	_ io.ByteWriter = (*Encoder)(nil)
	_ io.Writer     = (*Encoder)(nil)
)
