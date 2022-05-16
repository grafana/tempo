package rle

import (
	"encoding/binary"
	"io"

	"github.com/segmentio/parquet-go/internal/bits"
)

type runLengthRunDecoder struct {
	reader   io.Reader
	remain   uint
	length   uint
	bitWidth uint
	buffer   [8]byte
}

func (d *runLengthRunDecoder) String() string { return "RLE" }

func (d *runLengthRunDecoder) reset(r io.Reader, bitWidth, numValues uint) {
	d.reader = r
	d.remain = numValues
	d.length = uint(bits.ByteCount(bitWidth))
	d.bitWidth = bitWidth
	d.buffer = [8]byte{}
}

func (d *runLengthRunDecoder) decode(dst []byte, dstWidth uint) (int, error) {
	if d.remain == 0 {
		return 0, io.EOF
	}

	if d.length != 0 {
		_, err := io.ReadFull(d.reader, d.buffer[:d.length])
		if err != nil {
			return 0, err
		}
		d.length = 0
	}

	n := bits.BitCount(len(dst)) / dstWidth
	if n > d.remain {
		n = d.remain
	}
	dst = dst[:bits.ByteCount(n*dstWidth)]
	bits.Fill(dst, dstWidth, binary.LittleEndian.Uint64(d.buffer[:]), d.bitWidth)
	d.remain -= n
	return int(n), nil
}

type runLengthRunEncoder struct {
	writer   io.Writer
	bitWidth uint
	buffer   [8]byte
}

func (e *runLengthRunEncoder) reset(w io.Writer, bitWidth uint) {
	e.writer, e.bitWidth = w, bitWidth
}

func (e *runLengthRunEncoder) encode(src []byte, srcWidth uint) error {
	// At this stage we make the assumption that the source buffer contains a
	// sequence of repeated values of the given bit width; we pack the first
	// value only into the encoder's buffer to adjust the bit width then write
	// it to the underlying io.Writer.
	v := uint64(0)
	switch srcWidth {
	case 8:
		v = uint64(src[0])
	case 16:
		v = uint64(binary.LittleEndian.Uint16(src))
	case 32:
		v = uint64(binary.LittleEndian.Uint32(src))
	case 64:
		v = binary.LittleEndian.Uint64(src)
	default:
		panic("BUG: unsupported source bit-width")
	}
	v &= (1 << uint64(e.bitWidth)) - 1
	binary.LittleEndian.PutUint64(e.buffer[:], v)
	_, err := e.writer.Write(e.buffer[:bits.ByteCount(e.bitWidth)])
	return err
}
