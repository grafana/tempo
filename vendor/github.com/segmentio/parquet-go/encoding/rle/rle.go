// Package rle implements the hybrid RLE/Bit-Packed encoding employed in
// repetition and definition levels, dictionary indexed data pages, and
// boolean values in the PLAIN encoding.
//
// https://github.com/apache/parquet-format/blob/master/Encodings.md#run-length-encoding--bit-packing-hybrid-rle--3
package rle

import (
	"io"

	"github.com/segmentio/parquet-go/encoding"
	"github.com/segmentio/parquet-go/format"
)

type Encoding struct {
}

func (e *Encoding) Encoding() format.Encoding {
	return format.RLE
}

func (e *Encoding) CanEncode(t format.Type) bool {
	return t == format.Boolean || t == format.Int32 || t == format.Int64
}

func (e *Encoding) NewDecoder(r io.Reader) encoding.Decoder {
	return NewDecoder(r)
}

func (e *Encoding) NewEncoder(w io.Writer) encoding.Encoder {
	return NewEncoder(w)
}

func (e *Encoding) String() string {
	return "RLE"
}
