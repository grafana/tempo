package rle

import (
	"fmt"
	"io"
	. "math/bits"

	"github.com/segmentio/parquet-go/internal/bits"
)

const (
	unlimited = ^uint(0)
)

type bitPackRunDecoder struct {
	source   io.LimitedReader
	reader   bits.Reader
	remain   uint
	bitWidth uint
}

func (d *bitPackRunDecoder) String() string { return "BIT_PACK" }

func (d *bitPackRunDecoder) reset(r io.Reader, bitWidth, numValues uint) {
	if numValues == unlimited {
		d.reader.Reset(r)
	} else {
		d.source.R = r
		d.source.N = int64(bits.ByteCount(numValues * bitWidth))
		d.reader.Reset(&d.source)
	}
	d.remain = numValues
	d.bitWidth = bitWidth
}

func (d *bitPackRunDecoder) decode(dst []byte, dstWidth uint) (n int, err error) {
	dstBitCount := bits.BitCount(len(dst))

	if dstWidth < 8 || dstWidth > 64 || OnesCount(dstWidth) != 1 {
		return 0, fmt.Errorf("BIT_PACK decoder expects the output size to be a power of 8 bits but got %d bits", dstWidth)
	}

	if (dstBitCount & (dstWidth - 1)) != 0 { // (dstBitCount % dstWidth) != 0
		return 0, fmt.Errorf("BIT_PACK decoder expects the input size to be a multiple of the destination width: bit-count=%d bit-width=%d",
			dstBitCount, dstWidth)
	}

	if dstWidth < d.bitWidth {
		return 0, fmt.Errorf("BIT_PACK decoder cannot encode %d bits values to %d bits: the source width must be less or equal to the destination width",
			d.bitWidth, dstWidth)
	}

	switch dstWidth {
	case 8:
		n, err = d.decodeInt8(bits.BytesToInt8(dst), d.bitWidth)
	case 16:
		n, err = d.decodeInt16(bits.BytesToInt16(dst), d.bitWidth)
	case 32:
		n, err = d.decodeInt32(bits.BytesToInt32(dst), d.bitWidth)
	case 64:
		n, err = d.decodeInt64(bits.BytesToInt64(dst), d.bitWidth)
	default:
		panic("BUG: unsupported destination bit-width")
	}

	if d.remain != unlimited {
		if d.remain -= uint(n); d.remain == 0 {
			err = io.EOF
		} else if err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
		}
	}

	return n, err
}

func (d *bitPackRunDecoder) decodeInt8(dst []int8, bitWidth uint) (n int, err error) {
	if uint(len(dst)) > d.remain {
		dst = dst[:d.remain]
	}
	for n < len(dst) {
		b, _, err := d.reader.ReadBits(bitWidth)
		if err != nil {
			return n, err
		}
		dst[n] = int8(b)
		n++
	}
	return n, nil
}

func (d *bitPackRunDecoder) decodeInt16(dst []int16, bitWidth uint) (n int, err error) {
	if uint(len(dst)) > d.remain {
		dst = dst[:d.remain]
	}
	for n < len(dst) {
		b, _, err := d.reader.ReadBits(bitWidth)
		if err != nil {
			return n, err
		}
		dst[n] = int16(b)
		n++
	}
	return n, nil
}

func (d *bitPackRunDecoder) decodeInt32(dst []int32, bitWidth uint) (n int, err error) {
	if uint(len(dst)) > d.remain {
		dst = dst[:d.remain]
	}
	for n < len(dst) {
		b, _, err := d.reader.ReadBits(bitWidth)
		if err != nil {
			return n, err
		}
		dst[n] = int32(b)
		n++
	}
	return n, nil
}

func (d *bitPackRunDecoder) decodeInt64(dst []int64, bitWidth uint) (n int, err error) {
	if uint(len(dst)) > d.remain {
		dst = dst[:d.remain]
	}
	for n < len(dst) {
		b, _, err := d.reader.ReadBits(bitWidth)
		if err != nil {
			return n, err
		}
		dst[n] = int64(b)
		n++
	}
	return n, nil
}

type bitPackRunEncoder struct {
	writer   bits.Writer
	bitWidth uint
}

func (e *bitPackRunEncoder) reset(w io.Writer, bitWidth uint) {
	e.writer.Reset(w)
	e.bitWidth = bitWidth
}

func (e *bitPackRunEncoder) flush() error {
	return e.writer.Flush()
}

func (e *bitPackRunEncoder) encode(src []byte, srcWidth uint) error {
	srcBitCount := bits.BitCount(len(src))

	if srcWidth < 8 || srcWidth > 64 || OnesCount(srcWidth) != 1 {
		return fmt.Errorf("BIT_PACK encoder expects the input size to be a power of 8 bits but got %d bits", srcWidth)
	}

	if (srcBitCount & (srcWidth - 1)) != 0 { // (srcBitCount % srcWidth) != 0
		return fmt.Errorf("BIT_PACK encoder expects the input size to be a multiple of the source width: bit-count=%d bit-width=%d", srcBitCount, srcWidth)
	}

	if ((srcBitCount / srcWidth) % 8) != 0 {
		return fmt.Errorf("BIT_PACK encoder expects sequences of 8 values but %d were written", srcBitCount/srcWidth)
	}

	if srcWidth < e.bitWidth {
		return fmt.Errorf("BIT_PACK encoder cannot encode %d bits values to %d bits: the source width must be less or equal to the destination width",
			srcWidth, e.bitWidth)
	}

	switch srcWidth {
	case 8:
		e.encodeInt8(bits.BytesToInt8(src), e.bitWidth)
	case 16:
		e.encodeInt16(bits.BytesToInt16(src), e.bitWidth)
	case 32:
		e.encodeInt32(bits.BytesToInt32(src), e.bitWidth)
	case 64:
		e.encodeInt64(bits.BytesToInt64(src), e.bitWidth)
	default:
		panic("BUG: unsupported source bit-width")
	}

	return e.flush()
}

func (e *bitPackRunEncoder) encodeInt8(src []int8, bitWidth uint) {
	for _, v := range src {
		e.writer.WriteBits(uint64(v), bitWidth)
	}
}

func (e *bitPackRunEncoder) encodeInt16(src []int16, bitWidth uint) {
	for _, v := range src {
		e.writer.WriteBits(uint64(v), bitWidth)
	}
}

func (e *bitPackRunEncoder) encodeInt32(src []int32, bitWidth uint) {
	for _, v := range src {
		e.writer.WriteBits(uint64(v), bitWidth)
	}
}

func (e *bitPackRunEncoder) encodeInt64(src []int64, bitWidth uint) {
	for _, v := range src {
		e.writer.WriteBits(uint64(v), bitWidth)
	}
}
