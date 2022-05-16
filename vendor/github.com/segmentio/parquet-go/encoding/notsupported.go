package encoding

import (
	"errors"
	"fmt"
	"io"

	"github.com/segmentio/parquet-go/deprecated"
	"github.com/segmentio/parquet-go/format"
)

var (
	// ErrNotSupported is an error returned when the underlying encoding does
	// not support the type of values being encoded or decoded.
	//
	// This error may be wrapped with type information, applications must use
	// errors.Is rather than equality comparisons to test the error values
	// returned by encoders and decoders.
	ErrNotSupported = errors.New("encoding not supported")

	// ErrInvalidArgument is an error returned one or more arguments passed to
	// the encoding functions are incorrect.
	//
	// As with ErrNotSupported, this error may be wrapped with specific
	// information about the problem and applications are expected to use
	// errors.Is for comparisons.
	ErrInvalidArgument = errors.New("invalid argument")
)

// NotSupported is a type satisfying the Encoding interface which does not
// support encoding nor decoding any value types.
type NotSupported struct {
}

func (NotSupported) Encoding() format.Encoding {
	return -1
}

func (NotSupported) CanEncode(format.Type) bool {
	return false
}

func (NotSupported) NewDecoder(io.Reader) Decoder {
	return NotSupportedDecoder{}
}

func (NotSupported) NewEncoder(io.Writer) Encoder {
	return NotSupportedEncoder{}
}

func (NotSupported) String() string {
	return "NOT_SUPPORTED"
}

// NotSupportedDecoder is an implementation of the Decoder interface which does
// not support decoding any value types.
//
// Many parquet encodings only support decoding a subset of the parquet types,
// they can embed this type to default to not supporting any decoding, then
// override specific Decode* methods to provide implementations for the types
// they do support.
type NotSupportedDecoder struct {
}

func (NotSupportedDecoder) Encoding() format.Encoding {
	return -1
}

func (NotSupportedDecoder) Reset(io.Reader) {
}

func (NotSupportedDecoder) DecodeBoolean([]bool) (int, error) {
	return 0, errNotSupported("BOOLEAN")
}

func (NotSupportedDecoder) DecodeInt8([]int8) (int, error) {
	return 0, errNotSupported("INT8")
}

func (NotSupportedDecoder) DecodeInt16([]int16) (int, error) {
	return 0, errNotSupported("INT16")
}

func (NotSupportedDecoder) DecodeInt32([]int32) (int, error) {
	return 0, errNotSupported("INT32")
}

func (NotSupportedDecoder) DecodeInt64([]int64) (int, error) {
	return 0, errNotSupported("INT64")
}

func (NotSupportedDecoder) DecodeInt96([]deprecated.Int96) (int, error) {
	return 0, errNotSupported("INT96")
}

func (NotSupportedDecoder) DecodeFloat([]float32) (int, error) {
	return 0, errNotSupported("FLOAT")
}

func (NotSupportedDecoder) DecodeDouble([]float64) (int, error) {
	return 0, errNotSupported("DOUBLE")
}

func (NotSupportedDecoder) DecodeByteArray(*ByteArrayList) (int, error) {
	return 0, errNotSupported("BYTE_ARRAY")
}

func (NotSupportedDecoder) DecodeFixedLenByteArray(size int, data []byte) (int, error) {
	return 0, errNotSupported("FIXED_LEN_BYTE_ARRAY")
}

func (NotSupportedDecoder) SetBitWidth(int) {
}

// NotSupportedEncoder is an implementation of the Encoder interface which does
// not support encoding any value types.
//
// Many parquet encodings only support encoding a subset of the parquet types,
// they can embed this type to default to not supporting any encoding, then
// override specific Encode* methods to provide implementations for the types
// they do support.
type NotSupportedEncoder struct {
}

func (NotSupportedEncoder) Encoding() format.Encoding {
	return -1
}

func (NotSupportedEncoder) Reset(io.Writer) {
}

func (NotSupportedEncoder) EncodeBoolean([]bool) error {
	return errNotSupported("BOOLEAN")
}

func (NotSupportedEncoder) EncodeInt8([]int8) error {
	return errNotSupported("INT8")
}

func (NotSupportedEncoder) EncodeInt16([]int16) error {
	return errNotSupported("INT16")
}

func (NotSupportedEncoder) EncodeInt32([]int32) error {
	return errNotSupported("INT32")
}

func (NotSupportedEncoder) EncodeInt64([]int64) error {
	return errNotSupported("INT64")
}

func (NotSupportedEncoder) EncodeInt96([]deprecated.Int96) error {
	return errNotSupported("INT96")
}

func (NotSupportedEncoder) EncodeFloat([]float32) error {
	return errNotSupported("FLOAT")
}

func (NotSupportedEncoder) EncodeDouble([]float64) error {
	return errNotSupported("DOUBLE")
}

func (NotSupportedEncoder) EncodeByteArray(ByteArrayList) error {
	return errNotSupported("BYTE_ARRAY")
}

func (NotSupportedEncoder) EncodeFixedLenByteArray(int, []byte) error {
	return errNotSupported("FIXED_LEN_BYTE_ARRAY")
}

func (NotSupportedEncoder) SetBitWidth(int) {
}

func errNotSupported(typ string) error {
	return fmt.Errorf("%w for type %s", ErrNotSupported, typ)
}
