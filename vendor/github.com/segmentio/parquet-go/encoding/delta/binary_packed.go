package delta

import (
	"io"

	"github.com/segmentio/parquet-go/encoding"
	"github.com/segmentio/parquet-go/format"
)

type BinaryPackedEncoding struct {
}

func (e *BinaryPackedEncoding) Encoding() format.Encoding {
	return format.DeltaBinaryPacked
}

func (e *BinaryPackedEncoding) CanEncode(t format.Type) bool {
	return t == format.Int32 || t == format.Int64
}

func (e *BinaryPackedEncoding) NewDecoder(r io.Reader) encoding.Decoder {
	return NewBinaryPackedDecoder(r)
}

func (e *BinaryPackedEncoding) NewEncoder(w io.Writer) encoding.Encoder {
	return NewBinaryPackedEncoder(w)
}

func (e *BinaryPackedEncoding) String() string {
	return "DELTA_BINARY_PACKED"
}
