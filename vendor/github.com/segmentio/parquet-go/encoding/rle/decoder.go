package rle

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/segmentio/parquet-go/encoding"
	"github.com/segmentio/parquet-go/internal/bits"
)

type Decoder struct {
	encoding.NotSupportedDecoder
	reader    io.Reader
	buffer    [1]byte
	bitWidth  uint
	decoder   hybridDecoder
	runLength runLengthRunDecoder
	bitPack   bitPackRunDecoder
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{reader: r}
}

func (d *Decoder) BitWidth() int {
	return int(d.bitWidth)
}

func (d *Decoder) SetBitWidth(bitWidth int) {
	d.bitWidth = uint(bitWidth)
}

func (d *Decoder) Reset(r io.Reader) {
	d.reader, d.decoder = r, nil
}

func (d *Decoder) Read(b []byte) (int, error) {
	return d.reader.Read(b)
}

func (d *Decoder) ReadByte() (byte, error) {
	_, err := d.Read(d.buffer[:1])
	return d.buffer[0], err
}

func (d *Decoder) DecodeBoolean(data []bool) (int, error) {
	// When decoding booleans with the RLE encoding, only the BIT_PACKED version
	// is used, which skips encoding of the varint header, and consumes bits
	// until EOF is reached.
	if d.decoder == nil {
		d.bitPack.reset(d.reader, 1, unlimited)
		d.decoder = &d.bitPack
	}
	return d.decode(bits.BoolToBytes(data), 8, 1)
}

func (d *Decoder) DecodeInt8(data []int8) (int, error) {
	return d.decode(bits.Int8ToBytes(data), 8, d.bitWidth)
}

func (d *Decoder) DecodeInt16(data []int16) (int, error) {
	return d.decode(bits.Int16ToBytes(data), 16, d.bitWidth)
}

func (d *Decoder) DecodeInt32(data []int32) (int, error) {
	return d.decode(bits.Int32ToBytes(data), 32, d.bitWidth)
}

func (d *Decoder) DecodeInt64(data []int64) (int, error) {
	return d.decode(bits.Int64ToBytes(data), 64, d.bitWidth)
}

func (d *Decoder) decode(data []byte, dstWidth, srcWidth uint) (int, error) {
	if srcWidth == 0 {
		return 0, fmt.Errorf("the source bit-width must be configured on a RLE decoder before reading %d bits integer values", dstWidth)
	}
	decoded := 0
	wordSize := bits.ByteCount(dstWidth)

	for len(data) >= wordSize {
		if d.decoder == nil {
			u, err := binary.ReadUvarint(d)
			switch err {
			case nil:
				count, bitpack := uint(u>>1), (u&1) != 0
				if bitpack {
					d.bitPack.reset(d.reader, srcWidth, count*8)
					d.decoder = &d.bitPack
				} else {
					d.runLength.reset(d.reader, srcWidth, count)
					d.decoder = &d.runLength
				}
			case io.EOF:
				if decoded > 0 {
					err = nil
				}
				return decoded, err
			default:
				return decoded, fmt.Errorf("decoding RLE run length: %w", err)
			}
		}

		n, err := d.decoder.decode(data, dstWidth)
		decoded += n

		if err != nil {
			if err == io.EOF {
				d.decoder = nil
			} else {
				return decoded, fmt.Errorf("decoding RLE values from %s encoded run: %w", d.decoder, err)
			}
		}

		data = data[n*wordSize:]
	}

	return decoded, nil
}

type hybridDecoder interface {
	decode(dst []byte, dstWidth uint) (int, error)
}

var (
	_ io.ByteReader = (*Decoder)(nil)
	_ io.Reader     = (*Decoder)(nil)
)
