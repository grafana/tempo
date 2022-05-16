package delta

import (
	"io"

	"github.com/segmentio/parquet-go/encoding"
	"github.com/segmentio/parquet-go/format"
)

type LengthByteArrayEncoding struct {
}

func (e *LengthByteArrayEncoding) Encoding() format.Encoding {
	return format.DeltaLengthByteArray
}

func (e *LengthByteArrayEncoding) CanEncode(t format.Type) bool {
	return t == format.ByteArray
}

func (e *LengthByteArrayEncoding) NewDecoder(r io.Reader) encoding.Decoder {
	return NewLengthByteArrayDecoder(r)
}

func (e *LengthByteArrayEncoding) NewEncoder(w io.Writer) encoding.Encoder {
	return NewLengthByteArrayEncoder(w)
}

func (e *LengthByteArrayEncoding) String() string {
	return "DELTA_LENGTH_BYTE_ARRAY"
}
