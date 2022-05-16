package delta

import (
	"io"

	"github.com/segmentio/parquet-go/encoding"
	"github.com/segmentio/parquet-go/format"
)

type ByteArrayEncoding struct {
}

func (e *ByteArrayEncoding) Encoding() format.Encoding {
	return format.DeltaByteArray
}

func (e *ByteArrayEncoding) CanEncode(t format.Type) bool {
	// The parquet specs say that this encoding is only supported for BYTE_ARRAY
	// values, but the reference Java implementation appears to support
	// FIXED_LEN_BYTE_ARRAY as well:
	// https://github.com/apache/parquet-mr/blob/5608695f5777de1eb0899d9075ec9411cfdf31d3/parquet-column/src/main/java/org/apache/parquet/column/Encoding.java#L211
	return t == format.ByteArray || t == format.FixedLenByteArray
}

func (e *ByteArrayEncoding) NewDecoder(r io.Reader) encoding.Decoder {
	return NewByteArrayDecoder(r)
}

func (e *ByteArrayEncoding) NewEncoder(w io.Writer) encoding.Encoder {
	return NewByteArrayEncoder(w)
}

func (e *ByteArrayEncoding) String() string {
	return "DELTA_BYTE_ARRAY"
}
