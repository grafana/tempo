// Package encoding provides the generic APIs implemented by parquet encodings
// in its sub-packages.
package encoding

import (
	"io"

	"github.com/segmentio/parquet-go/deprecated"
	"github.com/segmentio/parquet-go/format"
)

// The Encoding interface is implemented by types representing parquet column
// encodings.
//
// Encoding instances must be safe to use concurrently from multiple goroutines.
type Encoding interface {
	// Returns a human-readable name for the encoding.
	String() string

	// Returns the parquet code representing the encoding.
	Encoding() format.Encoding

	// Checks whether the encoding is capable of serializing parquet values of
	// the given type.
	CanEncode(format.Type) bool

	// Creates a decoder reading encoded values to the io.Reader passed as
	// argument.
	//
	// The io.Reader may be nil, in which case the decoder's Reset method must
	// be called with a non-nil io.Reader prior to decoding values.
	NewDecoder(io.Reader) Decoder

	// Creates an encoder writing values to the io.Writer passed as argument.
	//
	// The io.Writer may be nil, in which case the encoder's Reset method must
	// be called with a non-nil io.Writer prior to encoding values.
	NewEncoder(io.Writer) Encoder
}

// The Encoder interface is implemented by encoders types.
//
// Some encodings only support partial
type Encoder interface {
	// Calling Reset clears the encoder state and changes the io.Writer where
	// encoded values are written to the one given as argument.
	//
	// The io.Writer may be nil, in which case the encoder must not be used
	// until Reset is called again with a non-nil writer.
	//
	// Calling Reset does not override the bit-width configured on the encoder.
	Reset(io.Writer)

	// Encodes an array of boolean values using this encoder.
	EncodeBoolean(data []bool) error

	// Encodes an array of 8 bits integer values using this encoder.
	//
	// The parquet type system does not have a 8 bits integers, this method
	// is intended to encode INT32 values but receives them as an array of
	// int8 values to enable greater memory efficiency when the application
	// knows that all values can fit in 8 bits.
	EncodeInt8(data []int8) error

	// Encodes an array of boolean values using this encoder.
	//
	// The parquet type system does not have a 16 bits integers, this method
	// is intended to encode INT32 values but receives them as an array of
	// int8 values to enable greater memory efficiency when the application
	// knows that all values can fit in 16 bits.
	EncodeInt16(data []int16) error

	// Encodes an array of 32 bit integer values using this encoder.
	EncodeInt32(data []int32) error

	// Encodes an array of 64 bit integer values using this encoder.
	EncodeInt64(data []int64) error

	// Encodes an array of 96 bit integer values using this encoder.
	EncodeInt96(data []deprecated.Int96) error

	// Encodes an array of 32 bit floating point values using this encoder.
	EncodeFloat(data []float32) error

	// Encodes an array of 64 bit floating point values using this encoder.
	EncodeDouble(data []float64) error

	// Encodes an array of variable length byte array values using this encoder.
	EncodeByteArray(data ByteArrayList) error

	// Encodes an array of fixed length byte array values using this encoder.
	//
	// The list is encoded contiguously in the `data` byte slice, in chunks of
	// `size` elements
	EncodeFixedLenByteArray(size int, data []byte) error

	// Configures the bit-width on the encoder.
	//
	// Not all encodings require declaring the bit-width, but applications that
	// use the Encoder abstraction should not make assumptions about the
	// underlying type of the encoder, and therefore should call SetBitWidth
	// prior to encoding repetition and definition levels.
	SetBitWidth(bitWidth int)
}

// The Decoder interface is implemented by decoder types.
type Decoder interface {
	// Calling Reset clears the decoder state and changes the io.Reader where
	// decoded values are written to the one given as argument.
	//
	// The io.Reader may be nil, in which case the decoder must not be used
	// until Reset is called again with a non-nil reader.
	//
	// Calling Reset does not override the bit-width configured on the decoder.
	Reset(io.Reader)

	// Decodes an array of boolean values using this decoder, returning
	// the number of decoded values, and io.EOF if the end of the underlying
	// io.Reader was reached.
	DecodeBoolean(data []bool) (int, error)

	// Decodes an array of 8 bits integer values using this decoder, returning
	// the number of decoded values, and io.EOF if the end of the underlying
	// io.Reader was reached.
	//
	// The parquet type system does not have a 8 bits integers, this method
	// is intended to decode INT32 values but receives them as an array of
	// int8 values to enable greater memory efficiency when the application
	// knows that all values can fit in 8 bits.
	DecodeInt8(data []int8) (int, error)

	// Decodes an array of 16 bits integer values using this decoder, returning
	// the number of decoded values, and io.EOF if the end of the underlying
	// io.Reader was reached.
	//
	// The parquet type system does not have a 16 bits integers, this method
	// is intended to decode INT32 values but receives them as an array of
	// int8 values to enable greater memory efficiency when the application
	// knows that all values can fit in 16 bits.
	DecodeInt16(data []int16) (int, error)

	// Decodes an array of 32 bits integer values using this decoder, returning
	// the number of decoded values, and io.EOF if the end of the underlying
	// io.Reader was reached.
	DecodeInt32(data []int32) (int, error)

	// Decodes an array of 64 bits integer values using this decoder, returning
	// the number of decoded values, and io.EOF if the end of the underlying
	// io.Reader was reached.
	DecodeInt64(data []int64) (int, error)

	// Decodes an array of 96 bits integer values using this decoder, returning
	// the number of decoded values, and io.EOF if the end of the underlying
	// io.Reader was reached.
	DecodeInt96(data []deprecated.Int96) (int, error)

	// Decodes an array of 32 bits floating point values using this decoder,
	// returning the number of decoded values, and io.EOF if the end of the
	// underlying io.Reader was reached.
	DecodeFloat(data []float32) (int, error)

	// Decodes an array of 64 bits floating point values using this decoder,
	// returning the number of decoded values, and io.EOF if the end of the
	// underlying io.Reader was reached.
	DecodeDouble(data []float64) (int, error)

	// Decodes an array of variable length byte array values using this decoder,
	// returning the number of decoded values, and io.EOF if the end of the
	// underlying io.Reader was reached.
	//
	// The values are written to the `data` buffer by calling the Push method,
	// the method returns the number of values written. DecodeByteArray will
	// stop pushing value to the output ByteArrayList if its total capacity is
	// reached.
	DecodeByteArray(data *ByteArrayList) (int, error)

	// Decodes an array of fixed length byte array values using this decoder,
	// returning the number of decoded values, and io.EOF if the end of the
	// underlying io.Reader was reached.
	DecodeFixedLenByteArray(size int, data []byte) (int, error)

	// Configures the bit-width on the decoder.
	//
	// Not all encodings require declaring the bit-width, but applications that
	// use the Decoder abstraction should not make assumptions about the
	// underlying type of the decoder, and therefore should call SetBitWidth
	// prior to decoding repetition and definition levels.
	SetBitWidth(bitWidth int)
}
