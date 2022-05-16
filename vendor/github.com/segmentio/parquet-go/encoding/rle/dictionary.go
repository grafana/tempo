package rle

import (
	"fmt"
	"io"

	"github.com/segmentio/parquet-go/encoding"
	"github.com/segmentio/parquet-go/format"
	"github.com/segmentio/parquet-go/internal/bits"
)

type DictionaryEncoding struct {
}

func (e *DictionaryEncoding) Encoding() format.Encoding {
	return format.RLEDictionary
}

func (e *DictionaryEncoding) CanEncode(t format.Type) bool {
	return true
}

func (e *DictionaryEncoding) NewDecoder(r io.Reader) encoding.Decoder {
	return dictionaryDecoder{rle: NewDecoder(r)}
}

func (e *DictionaryEncoding) NewEncoder(w io.Writer) encoding.Encoder {
	return dictionaryEncoder{rle: NewEncoder(w)}
}

func (e *DictionaryEncoding) String() string {
	return "RLE_DICTIONARY"
}

type dictionaryDecoder struct {
	encoding.NotSupportedDecoder
	rle  *Decoder
	zero bool
}

func (d dictionaryDecoder) Reset(r io.Reader) {
	d.rle.Reset(r)
	d.rle.SetBitWidth(0)
	d.zero = false
}

func (d dictionaryDecoder) DecodeInt32(data []int32) (int, error) {
	if d.zero {
		clearInt32(data)
		return len(data), nil
	}
	if d.rle.BitWidth() == 0 {
		bitWidth, err := d.decodeBitWidth()
		if err != nil {
			return 0, err
		}
		// Sometimes, when the dictionary contains only a single value, the page
		// can be encoded as a zero bit width to indicate that all indexes will
		// be zero.
		if bitWidth == 0 {
			d.zero = true
			clearInt32(data)
			return len(data), nil
		}
		d.rle.SetBitWidth(bitWidth)
	}
	return d.rle.DecodeInt32(data)
}

func (d dictionaryDecoder) decodeBitWidth() (int, error) {
	b, err := d.rle.ReadByte()
	switch err {
	case nil:
		if b > 32 {
			return 0, fmt.Errorf("decoding RLE bit width: %d>32", b)
		}
		return int(b), nil
	case io.EOF:
		return 0, err
	default:
		return 0, fmt.Errorf("decoding RLE bit width: %w", err)
	}
}

type dictionaryEncoder struct {
	encoding.NotSupportedEncoder
	rle *Encoder
}

func (e dictionaryEncoder) Reset(w io.Writer) {
	e.rle.Reset(w)
}

func (e dictionaryEncoder) EncodeInt32(data []int32) error {
	bitWidth := bits.MaxLen32(data)
	if bitWidth == 0 {
		bitWidth = 1
	}
	if err := e.encodeBitWidth(bitWidth); err != nil {
		return err
	}
	e.rle.SetBitWidth(bitWidth)
	return e.rle.EncodeInt32(data)
}

func (e dictionaryEncoder) encodeBitWidth(bitWidth int) error {
	return e.rle.WriteByte(byte(bitWidth))
}

func clearInt32(data []int32) {
	for i := range data {
		data[i] = 0
	}
}
