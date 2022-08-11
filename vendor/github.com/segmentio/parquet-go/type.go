package parquet

import (
	"bytes"
	"fmt"
	"math/bits"
	"time"

	"github.com/segmentio/parquet-go/deprecated"
	"github.com/segmentio/parquet-go/encoding"
	"github.com/segmentio/parquet-go/format"
)

// Kind is an enumeration type representing the physical types supported by the
// parquet type system.
type Kind int8

const (
	Boolean           Kind = Kind(format.Boolean)
	Int32             Kind = Kind(format.Int32)
	Int64             Kind = Kind(format.Int64)
	Int96             Kind = Kind(format.Int96)
	Float             Kind = Kind(format.Float)
	Double            Kind = Kind(format.Double)
	ByteArray         Kind = Kind(format.ByteArray)
	FixedLenByteArray Kind = Kind(format.FixedLenByteArray)
)

// String returns a human-readable representation of the physical type.
func (k Kind) String() string { return format.Type(k).String() }

// Value constructs a value from k and v.
//
// The method panics if the data is not a valid representation of the value
// kind; for example, if the kind is Int32 but the data is not 4 bytes long.
func (k Kind) Value(v []byte) Value {
	x, err := parseValue(k, v)
	if err != nil {
		panic(err)
	}
	return x
}

// The Type interface represents logical types of the parquet type system.
//
// Types are immutable and therefore safe to access from multiple goroutines.
type Type interface {
	// Returns a human-readable representation of the parquet type.
	String() string

	// Returns the Kind value representing the underlying physical type.
	//
	// The method panics if it is called on a group type.
	Kind() Kind

	// For integer and floating point physical types, the method returns the
	// size of values in bits.
	//
	// For fixed-length byte arrays, the method returns the size of elements
	// in bytes.
	//
	// For other types, the value is zero.
	Length() int

	// Returns an estimation of the number of bytes required to hold the given
	// number of values of this type in memory.
	//
	// The method returns zero for group types.
	EstimateSize(numValues int) int64

	// Compares two values and returns a negative integer if a < b, positive if
	// a > b, or zero if a == b.
	//
	// The values' Kind must match the type, otherwise the result is undefined.
	//
	// The method panics if it is called on a group type.
	Compare(a, b Value) int

	// ColumnOrder returns the type's column order. For group types, this method
	// returns nil.
	//
	// The order describes the comparison logic implemented by the Less method.
	//
	// As an optimization, the method may return the same pointer across
	// multiple calls. Applications must treat the returned value as immutable,
	// mutating the value will result in undefined behavior.
	ColumnOrder() *format.ColumnOrder

	// Returns the physical type as a *format.Type value. For group types, this
	// method returns nil.
	//
	// As an optimization, the method may return the same pointer across
	// multiple calls. Applications must treat the returned value as immutable,
	// mutating the value will result in undefined behavior.
	PhysicalType() *format.Type

	// Returns the logical type as a *format.LogicalType value. When the logical
	// type is unknown, the method returns nil.
	//
	// As an optimization, the method may return the same pointer across
	// multiple calls. Applications must treat the returned value as immutable,
	// mutating the value will result in undefined behavior.
	LogicalType() *format.LogicalType

	// Returns the logical type's equivalent converted type. When there are
	// no equivalent converted type, the method returns nil.
	//
	// As an optimization, the method may return the same pointer across
	// multiple calls. Applications must treat the returned value as immutable,
	// mutating the value will result in undefined behavior.
	ConvertedType() *deprecated.ConvertedType

	// Creates a column indexer for values of this type.
	//
	// The size limit is a hint to the column indexer that it is allowed to
	// truncate the page boundaries to the given size. Only BYTE_ARRAY and
	// FIXED_LEN_BYTE_ARRAY types currently take this value into account.
	//
	// A value of zero or less means no limits.
	//
	// The method panics if it is called on a group type.
	NewColumnIndexer(sizeLimit int) ColumnIndexer

	// Creates a row group buffer column for values of this type.
	//
	// Column buffers are created using the index of the column they are
	// accumulating values in memory for (relative to the parent schema),
	// and the size of their memory buffer.
	//
	// The application may give an estimate of the number of values it expects
	// to write to the buffer as second argument. This estimate helps set the
	// initialize buffer capacity but is not a hard limit, the underlying memory
	// buffer will grown as needed to allow more values to be written. Programs
	// may use the Size method of the column buffer (or the parent row group,
	// when relevant) to determine how many bytes are being used, and perform a
	// flush of the buffers to a storage layer.
	//
	// The method panics if it is called on a group type.
	NewColumnBuffer(columnIndex, numValues int) ColumnBuffer

	// Creates a dictionary holding values of this type.
	//
	// The dictionary retains the data buffer, it does not make a copy of it.
	// If the application needs to share ownership of the memory buffer, it must
	// ensure that it will not be modified while the page is in use, or it must
	// make a copy of it prior to creating the dictionary.
	//
	// The method panics if the data type does not correspond to the parquet
	// type it is called on.
	NewDictionary(columnIndex, numValues int, data encoding.Values) Dictionary

	// Creates a page belonging to a column at the given index, backed by the
	// data buffer.
	//
	// The page retains the data buffer, it does not make a copy of it. If the
	// application needs to share ownership of the memory buffer, it must ensure
	// that it will not be modified while the page is in use, or it must make a
	// copy of it prior to creating the page.
	//
	// The method panics if the data type does not correspond to the parquet
	// type it is called on.
	NewPage(columnIndex, numValues int, data encoding.Values) Page

	// Creates an encoding.Values instance backed by the given buffers.
	//
	// The offsets is only used by BYTE_ARRAY types, where it represents the
	// positions of each variable length value in the values buffer.
	//
	// The following expression creates an empty instance for any type:
	//
	//		values := typ.NewValues(nil, nil)
	//
	// The method panics if it is called on group types.
	NewValues(values []byte, offsets []uint32) encoding.Values

	// Assuming the src buffer contains PLAIN encoded values of the type it is
	// called on, applies the given encoding and produces the output to the dst
	// buffer passed as first argument by dispatching the call to one of the
	// encoding methods.
	Encode(dst []byte, src encoding.Values, enc encoding.Encoding) ([]byte, error)

	// Assuming the src buffer contains values encoding in the given encoding,
	// decodes the input and produces the encoded values into the dst output
	// buffer passed as first argument by dispatching the call to one of the
	// encoding methods.
	Decode(dst encoding.Values, src []byte, enc encoding.Encoding) (encoding.Values, error)
}

var (
	BooleanType   Type = booleanType{}
	Int32Type     Type = int32Type{}
	Int64Type     Type = int64Type{}
	Int96Type     Type = int96Type{}
	FloatType     Type = floatType{}
	DoubleType    Type = doubleType{}
	ByteArrayType Type = byteArrayType{}
)

// In the current parquet version supported by this library, only type-defined
// orders are supported.
var typeDefinedColumnOrder = format.ColumnOrder{
	TypeOrder: new(format.TypeDefinedOrder),
}

var physicalTypes = [...]format.Type{
	0: format.Boolean,
	1: format.Int32,
	2: format.Int64,
	3: format.Int96,
	4: format.Float,
	5: format.Double,
	6: format.ByteArray,
	7: format.FixedLenByteArray,
}

var convertedTypes = [...]deprecated.ConvertedType{
	0:  deprecated.UTF8,
	1:  deprecated.Map,
	2:  deprecated.MapKeyValue,
	3:  deprecated.List,
	4:  deprecated.Enum,
	5:  deprecated.Decimal,
	6:  deprecated.Date,
	7:  deprecated.TimeMillis,
	8:  deprecated.TimeMicros,
	9:  deprecated.TimestampMillis,
	10: deprecated.TimestampMicros,
	11: deprecated.Uint8,
	12: deprecated.Uint16,
	13: deprecated.Uint32,
	14: deprecated.Uint64,
	15: deprecated.Int8,
	16: deprecated.Int16,
	17: deprecated.Int32,
	18: deprecated.Int64,
	19: deprecated.Json,
	20: deprecated.Bson,
	21: deprecated.Interval,
}

type booleanType struct{}

func (t booleanType) String() string                           { return "BOOLEAN" }
func (t booleanType) Kind() Kind                               { return Boolean }
func (t booleanType) Length() int                              { return 1 }
func (t booleanType) EstimateSize(n int) int64                 { return (int64(n) + 7) / 8 }
func (t booleanType) Compare(a, b Value) int                   { return compareBool(a.Boolean(), b.Boolean()) }
func (t booleanType) ColumnOrder() *format.ColumnOrder         { return &typeDefinedColumnOrder }
func (t booleanType) LogicalType() *format.LogicalType         { return nil }
func (t booleanType) ConvertedType() *deprecated.ConvertedType { return nil }
func (t booleanType) PhysicalType() *format.Type               { return &physicalTypes[Boolean] }

func (t booleanType) NewColumnIndexer(sizeLimit int) ColumnIndexer {
	return newBooleanColumnIndexer()
}

func (t booleanType) NewColumnBuffer(columnIndex, numValues int) ColumnBuffer {
	return newBooleanColumnBuffer(t, makeColumnIndex(columnIndex), makeNumValues(numValues))
}

func (t booleanType) NewDictionary(columnIndex, numValues int, data encoding.Values) Dictionary {
	return newBooleanDictionary(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t booleanType) NewPage(columnIndex, numValues int, data encoding.Values) Page {
	return newBooleanPage(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t booleanType) NewValues(values []byte, _ []uint32) encoding.Values {
	return encoding.BooleanValues(values)
}

func (t booleanType) Encode(dst []byte, src encoding.Values, enc encoding.Encoding) ([]byte, error) {
	return encoding.EncodeBoolean(dst, src, enc)
}

func (t booleanType) Decode(dst encoding.Values, src []byte, enc encoding.Encoding) (encoding.Values, error) {
	return encoding.DecodeBoolean(dst, src, enc)
}

type int32Type struct{}

func (t int32Type) String() string                           { return "INT32" }
func (t int32Type) Kind() Kind                               { return Int32 }
func (t int32Type) Length() int                              { return 32 }
func (t int32Type) EstimateSize(n int) int64                 { return 4 * int64(n) }
func (t int32Type) Compare(a, b Value) int                   { return compareInt32(a.Int32(), b.Int32()) }
func (t int32Type) ColumnOrder() *format.ColumnOrder         { return &typeDefinedColumnOrder }
func (t int32Type) LogicalType() *format.LogicalType         { return nil }
func (t int32Type) ConvertedType() *deprecated.ConvertedType { return nil }
func (t int32Type) PhysicalType() *format.Type               { return &physicalTypes[Int32] }

func (t int32Type) NewColumnIndexer(sizeLimit int) ColumnIndexer {
	return newInt32ColumnIndexer()
}

func (t int32Type) NewColumnBuffer(columnIndex, numValues int) ColumnBuffer {
	return newInt32ColumnBuffer(t, makeColumnIndex(columnIndex), makeNumValues(numValues))
}

func (t int32Type) NewDictionary(columnIndex, numValues int, data encoding.Values) Dictionary {
	return newInt32Dictionary(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t int32Type) NewPage(columnIndex, numValues int, data encoding.Values) Page {
	return newInt32Page(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t int32Type) NewValues(values []byte, _ []uint32) encoding.Values {
	return encoding.Int32ValuesFromBytes(values)
}

func (t int32Type) Encode(dst []byte, src encoding.Values, enc encoding.Encoding) ([]byte, error) {
	return encoding.EncodeInt32(dst, src, enc)
}

func (t int32Type) Decode(dst encoding.Values, src []byte, enc encoding.Encoding) (encoding.Values, error) {
	return encoding.DecodeInt32(dst, src, enc)
}

type int64Type struct{}

func (t int64Type) String() string                           { return "INT64" }
func (t int64Type) Kind() Kind                               { return Int64 }
func (t int64Type) Length() int                              { return 64 }
func (t int64Type) EstimateSize(n int) int64                 { return 8 * int64(n) }
func (t int64Type) Compare(a, b Value) int                   { return compareInt64(a.Int64(), b.Int64()) }
func (t int64Type) ColumnOrder() *format.ColumnOrder         { return &typeDefinedColumnOrder }
func (t int64Type) LogicalType() *format.LogicalType         { return nil }
func (t int64Type) ConvertedType() *deprecated.ConvertedType { return nil }
func (t int64Type) PhysicalType() *format.Type               { return &physicalTypes[Int64] }

func (t int64Type) NewColumnIndexer(sizeLimit int) ColumnIndexer {
	return newInt64ColumnIndexer()
}

func (t int64Type) NewColumnBuffer(columnIndex, numValues int) ColumnBuffer {
	return newInt64ColumnBuffer(t, makeColumnIndex(columnIndex), makeNumValues(numValues))
}

func (t int64Type) NewDictionary(columnIndex, numValues int, data encoding.Values) Dictionary {
	return newInt64Dictionary(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t int64Type) NewPage(columnIndex, numValues int, data encoding.Values) Page {
	return newInt64Page(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t int64Type) NewValues(values []byte, _ []uint32) encoding.Values {
	return encoding.Int64ValuesFromBytes(values)
}

func (t int64Type) Encode(dst []byte, src encoding.Values, enc encoding.Encoding) ([]byte, error) {
	return encoding.EncodeInt64(dst, src, enc)
}

func (t int64Type) Decode(dst encoding.Values, src []byte, enc encoding.Encoding) (encoding.Values, error) {
	return encoding.DecodeInt64(dst, src, enc)
}

type int96Type struct{}

func (t int96Type) String() string { return "INT96" }

func (t int96Type) Kind() Kind                               { return Int96 }
func (t int96Type) Length() int                              { return 96 }
func (t int96Type) EstimateSize(n int) int64                 { return 12 * int64(n) }
func (t int96Type) Compare(a, b Value) int                   { return compareInt96(a.Int96(), b.Int96()) }
func (t int96Type) ColumnOrder() *format.ColumnOrder         { return &typeDefinedColumnOrder }
func (t int96Type) LogicalType() *format.LogicalType         { return nil }
func (t int96Type) ConvertedType() *deprecated.ConvertedType { return nil }
func (t int96Type) PhysicalType() *format.Type               { return &physicalTypes[Int96] }

func (t int96Type) NewColumnIndexer(sizeLimit int) ColumnIndexer {
	return newInt96ColumnIndexer()
}

func (t int96Type) NewColumnBuffer(columnIndex, numValues int) ColumnBuffer {
	return newInt96ColumnBuffer(t, makeColumnIndex(columnIndex), makeNumValues(numValues))
}

func (t int96Type) NewDictionary(columnIndex, numValues int, data encoding.Values) Dictionary {
	return newInt96Dictionary(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t int96Type) NewPage(columnIndex, numValues int, data encoding.Values) Page {
	return newInt96Page(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t int96Type) NewValues(values []byte, _ []uint32) encoding.Values {
	return encoding.Int96ValuesFromBytes(values)
}

func (t int96Type) Encode(dst []byte, src encoding.Values, enc encoding.Encoding) ([]byte, error) {
	return encoding.EncodeInt96(dst, src, enc)
}

func (t int96Type) Decode(dst encoding.Values, src []byte, enc encoding.Encoding) (encoding.Values, error) {
	return encoding.DecodeInt96(dst, src, enc)
}

type floatType struct{}

func (t floatType) String() string                           { return "FLOAT" }
func (t floatType) Kind() Kind                               { return Float }
func (t floatType) Length() int                              { return 32 }
func (t floatType) EstimateSize(n int) int64                 { return 4 * int64(n) }
func (t floatType) Compare(a, b Value) int                   { return compareFloat32(a.Float(), b.Float()) }
func (t floatType) ColumnOrder() *format.ColumnOrder         { return &typeDefinedColumnOrder }
func (t floatType) LogicalType() *format.LogicalType         { return nil }
func (t floatType) ConvertedType() *deprecated.ConvertedType { return nil }
func (t floatType) PhysicalType() *format.Type               { return &physicalTypes[Float] }

func (t floatType) NewColumnIndexer(sizeLimit int) ColumnIndexer {
	return newFloatColumnIndexer()
}

func (t floatType) NewColumnBuffer(columnIndex, numValues int) ColumnBuffer {
	return newFloatColumnBuffer(t, makeColumnIndex(columnIndex), makeNumValues(numValues))
}

func (t floatType) NewDictionary(columnIndex, numValues int, data encoding.Values) Dictionary {
	return newFloatDictionary(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t floatType) NewPage(columnIndex, numValues int, data encoding.Values) Page {
	return newFloatPage(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t floatType) NewValues(values []byte, _ []uint32) encoding.Values {
	return encoding.FloatValuesFromBytes(values)
}

func (t floatType) Encode(dst []byte, src encoding.Values, enc encoding.Encoding) ([]byte, error) {
	return encoding.EncodeFloat(dst, src, enc)
}

func (t floatType) Decode(dst encoding.Values, src []byte, enc encoding.Encoding) (encoding.Values, error) {
	return encoding.DecodeFloat(dst, src, enc)
}

type doubleType struct{}

func (t doubleType) String() string                           { return "DOUBLE" }
func (t doubleType) Kind() Kind                               { return Double }
func (t doubleType) Length() int                              { return 64 }
func (t doubleType) EstimateSize(n int) int64                 { return 8 * int64(n) }
func (t doubleType) Compare(a, b Value) int                   { return compareFloat64(a.Double(), b.Double()) }
func (t doubleType) ColumnOrder() *format.ColumnOrder         { return &typeDefinedColumnOrder }
func (t doubleType) LogicalType() *format.LogicalType         { return nil }
func (t doubleType) ConvertedType() *deprecated.ConvertedType { return nil }
func (t doubleType) PhysicalType() *format.Type               { return &physicalTypes[Double] }

func (t doubleType) NewColumnIndexer(sizeLimit int) ColumnIndexer {
	return newDoubleColumnIndexer()
}

func (t doubleType) NewColumnBuffer(columnIndex, numValues int) ColumnBuffer {
	return newDoubleColumnBuffer(t, makeColumnIndex(columnIndex), makeNumValues(numValues))
}

func (t doubleType) NewDictionary(columnIndex, numValues int, data encoding.Values) Dictionary {
	return newDoubleDictionary(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t doubleType) NewPage(columnIndex, numValues int, data encoding.Values) Page {
	return newDoublePage(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t doubleType) NewValues(values []byte, _ []uint32) encoding.Values {
	return encoding.DoubleValuesFromBytes(values)
}

func (t doubleType) Encode(dst []byte, src encoding.Values, enc encoding.Encoding) ([]byte, error) {
	return encoding.EncodeDouble(dst, src, enc)
}

func (t doubleType) Decode(dst encoding.Values, src []byte, enc encoding.Encoding) (encoding.Values, error) {
	return encoding.DecodeDouble(dst, src, enc)
}

type byteArrayType struct{}

func (t byteArrayType) String() string                           { return "BYTE_ARRAY" }
func (t byteArrayType) Kind() Kind                               { return ByteArray }
func (t byteArrayType) Length() int                              { return 0 }
func (t byteArrayType) EstimateSize(n int) int64                 { return 10 * int64(n) }
func (t byteArrayType) Compare(a, b Value) int                   { return bytes.Compare(a.ByteArray(), b.ByteArray()) }
func (t byteArrayType) ColumnOrder() *format.ColumnOrder         { return &typeDefinedColumnOrder }
func (t byteArrayType) LogicalType() *format.LogicalType         { return nil }
func (t byteArrayType) ConvertedType() *deprecated.ConvertedType { return nil }
func (t byteArrayType) PhysicalType() *format.Type               { return &physicalTypes[ByteArray] }

func (t byteArrayType) NewColumnIndexer(sizeLimit int) ColumnIndexer {
	return newByteArrayColumnIndexer(sizeLimit)
}

func (t byteArrayType) NewColumnBuffer(columnIndex, numValues int) ColumnBuffer {
	return newByteArrayColumnBuffer(t, makeColumnIndex(columnIndex), makeNumValues(numValues))
}

func (t byteArrayType) NewDictionary(columnIndex, numValues int, data encoding.Values) Dictionary {
	return newByteArrayDictionary(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t byteArrayType) NewPage(columnIndex, numValues int, data encoding.Values) Page {
	return newByteArrayPage(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t byteArrayType) NewValues(values []byte, offsets []uint32) encoding.Values {
	return encoding.ByteArrayValues(values, offsets)
}

func (t byteArrayType) Encode(dst []byte, src encoding.Values, enc encoding.Encoding) ([]byte, error) {
	return encoding.EncodeByteArray(dst, src, enc)
}

func (t byteArrayType) Decode(dst encoding.Values, src []byte, enc encoding.Encoding) (encoding.Values, error) {
	return encoding.DecodeByteArray(dst, src, enc)
}

type fixedLenByteArrayType struct{ length int }

func (t fixedLenByteArrayType) String() string {
	return fmt.Sprintf("FIXED_LEN_BYTE_ARRAY(%d)", t.length)
}

func (t fixedLenByteArrayType) Kind() Kind { return FixedLenByteArray }

func (t fixedLenByteArrayType) Length() int { return t.length }

func (t fixedLenByteArrayType) EstimateSize(n int) int64 { return int64(t.length) * int64(n) }

func (t fixedLenByteArrayType) Compare(a, b Value) int {
	return bytes.Compare(a.ByteArray(), b.ByteArray())
}

func (t fixedLenByteArrayType) ColumnOrder() *format.ColumnOrder { return &typeDefinedColumnOrder }

func (t fixedLenByteArrayType) LogicalType() *format.LogicalType { return nil }

func (t fixedLenByteArrayType) ConvertedType() *deprecated.ConvertedType { return nil }

func (t fixedLenByteArrayType) PhysicalType() *format.Type { return &physicalTypes[FixedLenByteArray] }

func (t fixedLenByteArrayType) NewColumnIndexer(sizeLimit int) ColumnIndexer {
	return newFixedLenByteArrayColumnIndexer(t.length, sizeLimit)
}

func (t fixedLenByteArrayType) NewColumnBuffer(columnIndex, numValues int) ColumnBuffer {
	return newFixedLenByteArrayColumnBuffer(t, makeColumnIndex(columnIndex), makeNumValues(numValues))
}

func (t fixedLenByteArrayType) NewDictionary(columnIndex, numValues int, data encoding.Values) Dictionary {
	return newFixedLenByteArrayDictionary(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t fixedLenByteArrayType) NewPage(columnIndex, numValues int, data encoding.Values) Page {
	return newFixedLenByteArrayPage(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t fixedLenByteArrayType) NewValues(values []byte, _ []uint32) encoding.Values {
	return encoding.FixedLenByteArrayValues(values, t.length)
}

func (t fixedLenByteArrayType) Encode(dst []byte, src encoding.Values, enc encoding.Encoding) ([]byte, error) {
	return encoding.EncodeFixedLenByteArray(dst, src, enc)
}

func (t fixedLenByteArrayType) Decode(dst encoding.Values, src []byte, enc encoding.Encoding) (encoding.Values, error) {
	return encoding.DecodeFixedLenByteArray(dst, src, enc)
}

// BE128 stands for "big-endian 128 bits". This type is used as a special case
// for fixed-length byte arrays of 16 bytes, which are commonly used to
// represent columns of random unique identifiers such as UUIDs.
//
// Comparisons of BE128 values use the natural byte order, the zeroth byte is
// the most significant byte.
//
// The special case is intended to provide optimizations based on the knowledge
// that the values are 16 bytes long. Stronger type checking can also be applied
// by the compiler when using [16]byte values rather than []byte, reducing the
// risk of errors on these common code paths.
type be128Type struct{}

func (t be128Type) String() string { return "FIXED_LEN_BYTE_ARRAY(16)" }

func (t be128Type) Kind() Kind { return FixedLenByteArray }

func (t be128Type) Length() int { return 16 }

func (t be128Type) EstimateSize(n int) int64 { return 16 * int64(n) }

func (t be128Type) Compare(a, b Value) int {
	return compareBE128((*[16]byte)(a.ByteArray()), (*[16]byte)(b.ByteArray()))
}

func (t be128Type) ColumnOrder() *format.ColumnOrder { return &typeDefinedColumnOrder }

func (t be128Type) LogicalType() *format.LogicalType { return nil }

func (t be128Type) ConvertedType() *deprecated.ConvertedType { return nil }

func (t be128Type) PhysicalType() *format.Type { return &physicalTypes[FixedLenByteArray] }

func (t be128Type) NewColumnIndexer(sizeLimit int) ColumnIndexer {
	return newBE128ColumnIndexer()
}

func (t be128Type) NewColumnBuffer(columnIndex, numValues int) ColumnBuffer {
	return newBE128ColumnBuffer(t, makeColumnIndex(columnIndex), makeNumValues(numValues))
}

func (t be128Type) NewDictionary(columnIndex, numValues int, data encoding.Values) Dictionary {
	return newBE128Dictionary(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t be128Type) NewPage(columnIndex, numValues int, data encoding.Values) Page {
	return newBE128Page(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t be128Type) NewValues(values []byte, _ []uint32) encoding.Values {
	return encoding.FixedLenByteArrayValues(values, 16)
}

func (t be128Type) Encode(dst []byte, src encoding.Values, enc encoding.Encoding) ([]byte, error) {
	return encoding.EncodeFixedLenByteArray(dst, src, enc)
}

func (t be128Type) Decode(dst encoding.Values, src []byte, enc encoding.Encoding) (encoding.Values, error) {
	return encoding.DecodeFixedLenByteArray(dst, src, enc)
}

// FixedLenByteArrayType constructs a type for fixed-length values of the given
// size (in bytes).
func FixedLenByteArrayType(length int) Type {
	switch length {
	case 16:
		return be128Type{}
	default:
		return fixedLenByteArrayType{length: length}
	}
}

// Int constructs a leaf node of signed integer logical type of the given bit
// width.
//
// The bit width must be one of 8, 16, 32, 64, or the function will panic.
func Int(bitWidth int) Node {
	return Leaf(integerType(bitWidth, &signedIntTypes))
}

// Uint constructs a leaf node of unsigned integer logical type of the given
// bit width.
//
// The bit width must be one of 8, 16, 32, 64, or the function will panic.
func Uint(bitWidth int) Node {
	return Leaf(integerType(bitWidth, &unsignedIntTypes))
}

func integerType(bitWidth int, types *[4]intType) *intType {
	switch bitWidth {
	case 8:
		return &types[0]
	case 16:
		return &types[1]
	case 32:
		return &types[2]
	case 64:
		return &types[3]
	default:
		panic(fmt.Sprintf("cannot create a %d bits parquet integer node", bitWidth))
	}
}

var signedIntTypes = [...]intType{
	{BitWidth: 8, IsSigned: true},
	{BitWidth: 16, IsSigned: true},
	{BitWidth: 32, IsSigned: true},
	{BitWidth: 64, IsSigned: true},
}

var unsignedIntTypes = [...]intType{
	{BitWidth: 8, IsSigned: false},
	{BitWidth: 16, IsSigned: false},
	{BitWidth: 32, IsSigned: false},
	{BitWidth: 64, IsSigned: false},
}

type intType format.IntType

func (t *intType) String() string { return (*format.IntType)(t).String() }

func (t *intType) Kind() Kind {
	if t.BitWidth == 64 {
		return Int64
	} else {
		return Int32
	}
}

func (t *intType) Length() int { return int(t.BitWidth) }

func (t *intType) EstimateSize(n int) int64 { return int64(t.BitWidth/8) * int64(n) }

func (t *intType) Compare(a, b Value) int {
	if t.BitWidth == 64 {
		i1 := a.Int64()
		i2 := b.Int64()
		if t.IsSigned {
			return compareInt64(i1, i2)
		} else {
			return compareUint64(uint64(i1), uint64(i2))
		}
	} else {
		i1 := a.Int32()
		i2 := b.Int32()
		if t.IsSigned {
			return compareInt32(i1, i2)
		} else {
			return compareUint32(uint32(i1), uint32(i2))
		}
	}
}

func (t *intType) ColumnOrder() *format.ColumnOrder {
	return &typeDefinedColumnOrder
}

func (t *intType) PhysicalType() *format.Type {
	if t.BitWidth == 64 {
		return &physicalTypes[Int64]
	} else {
		return &physicalTypes[Int32]
	}
}

func (t *intType) LogicalType() *format.LogicalType {
	return &format.LogicalType{Integer: (*format.IntType)(t)}
}

func (t *intType) ConvertedType() *deprecated.ConvertedType {
	convertedType := bits.Len8(uint8(t.BitWidth)/8) - 1 // 8=>0, 16=>1, 32=>2, 64=>4
	if t.IsSigned {
		convertedType += int(deprecated.Int8)
	} else {
		convertedType += int(deprecated.Uint8)
	}
	return &convertedTypes[convertedType]
}

func (t *intType) NewColumnIndexer(sizeLimit int) ColumnIndexer {
	if t.IsSigned {
		if t.BitWidth == 64 {
			return newInt64ColumnIndexer()
		} else {
			return newInt32ColumnIndexer()
		}
	} else {
		if t.BitWidth == 64 {
			return newUint64ColumnIndexer()
		} else {
			return newUint32ColumnIndexer()
		}
	}
}

func (t *intType) NewColumnBuffer(columnIndex, numValues int) ColumnBuffer {
	if t.IsSigned {
		if t.BitWidth == 64 {
			return newInt64ColumnBuffer(t, makeColumnIndex(columnIndex), makeNumValues(numValues))
		} else {
			return newInt32ColumnBuffer(t, makeColumnIndex(columnIndex), makeNumValues(numValues))
		}
	} else {
		if t.BitWidth == 64 {
			return newUint64ColumnBuffer(t, makeColumnIndex(columnIndex), makeNumValues(numValues))
		} else {
			return newUint32ColumnBuffer(t, makeColumnIndex(columnIndex), makeNumValues(numValues))
		}
	}
}

func (t *intType) NewDictionary(columnIndex, numValues int, data encoding.Values) Dictionary {
	if t.IsSigned {
		if t.BitWidth == 64 {
			return newInt64Dictionary(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
		} else {
			return newInt32Dictionary(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
		}
	} else {
		if t.BitWidth == 64 {
			return newUint64Dictionary(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
		} else {
			return newUint32Dictionary(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
		}
	}
}

func (t *intType) NewPage(columnIndex, numValues int, data encoding.Values) Page {
	if t.IsSigned {
		if t.BitWidth == 64 {
			return newInt64Page(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
		} else {
			return newInt32Page(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
		}
	} else {
		if t.BitWidth == 64 {
			return newUint64Page(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
		} else {
			return newUint32Page(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
		}
	}
}

func (t *intType) NewValues(values []byte, _ []uint32) encoding.Values {
	if t.BitWidth == 64 {
		return encoding.Int64ValuesFromBytes(values)
	} else {
		return encoding.Int32ValuesFromBytes(values)
	}
}

func (t *intType) Encode(dst []byte, src encoding.Values, enc encoding.Encoding) ([]byte, error) {
	if t.BitWidth == 64 {
		return encoding.EncodeInt64(dst, src, enc)
	} else {
		return encoding.EncodeInt32(dst, src, enc)
	}
}

func (t *intType) Decode(dst encoding.Values, src []byte, enc encoding.Encoding) (encoding.Values, error) {
	if t.BitWidth == 64 {
		return encoding.DecodeInt64(dst, src, enc)
	} else {
		return encoding.DecodeInt32(dst, src, enc)
	}
}

// Decimal constructs a leaf node of decimal logical type with the given
// scale, precision, and underlying type.
//
// https://github.com/apache/parquet-format/blob/master/LogicalTypes.md#decimal
func Decimal(scale, precision int, typ Type) Node {
	switch typ.Kind() {
	case Int32, Int64, FixedLenByteArray:
	default:
		panic("DECIMAL node must annotate Int32, Int64 or FixedLenByteArray but got " + typ.String())
	}
	return Leaf(&decimalType{
		decimal: format.DecimalType{
			Scale:     int32(scale),
			Precision: int32(precision),
		},
		Type: typ,
	})
}

type decimalType struct {
	decimal format.DecimalType
	Type
}

func (t *decimalType) String() string { return t.decimal.String() }

func (t *decimalType) LogicalType() *format.LogicalType {
	return &format.LogicalType{Decimal: &t.decimal}
}

func (t *decimalType) ConvertedType() *deprecated.ConvertedType {
	return &convertedTypes[deprecated.Decimal]
}

// String constructs a leaf node of UTF8 logical type.
//
// https://github.com/apache/parquet-format/blob/master/LogicalTypes.md#string
func String() Node { return Leaf(&stringType{}) }

type stringType format.StringType

func (t *stringType) String() string { return (*format.StringType)(t).String() }

func (t *stringType) Kind() Kind { return ByteArray }

func (t *stringType) Length() int { return 0 }

func (t *stringType) EstimateSize(n int) int64 { return 10 * int64(n) }

func (t *stringType) Compare(a, b Value) int {
	return bytes.Compare(a.ByteArray(), b.ByteArray())
}

func (t *stringType) ColumnOrder() *format.ColumnOrder {
	return &typeDefinedColumnOrder
}

func (t *stringType) PhysicalType() *format.Type {
	return &physicalTypes[ByteArray]
}

func (t *stringType) LogicalType() *format.LogicalType {
	return &format.LogicalType{UTF8: (*format.StringType)(t)}
}

func (t *stringType) ConvertedType() *deprecated.ConvertedType {
	return &convertedTypes[deprecated.UTF8]
}

func (t *stringType) NewColumnIndexer(sizeLimit int) ColumnIndexer {
	return newByteArrayColumnIndexer(sizeLimit)
}

func (t *stringType) NewDictionary(columnIndex, numValues int, data encoding.Values) Dictionary {
	return newByteArrayDictionary(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t *stringType) NewColumnBuffer(columnIndex, numValues int) ColumnBuffer {
	return newByteArrayColumnBuffer(t, makeColumnIndex(columnIndex), makeNumValues(numValues))
}

func (t *stringType) NewPage(columnIndex, numValues int, data encoding.Values) Page {
	return newByteArrayPage(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t *stringType) NewValues(values []byte, offsets []uint32) encoding.Values {
	return encoding.ByteArrayValues(values, offsets)
}

func (t *stringType) Encode(dst []byte, src encoding.Values, enc encoding.Encoding) ([]byte, error) {
	return encoding.EncodeByteArray(dst, src, enc)
}

func (t *stringType) Decode(dst encoding.Values, src []byte, enc encoding.Encoding) (encoding.Values, error) {
	return encoding.DecodeByteArray(dst, src, enc)
}

// UUID constructs a leaf node of UUID logical type.
//
// https://github.com/apache/parquet-format/blob/master/LogicalTypes.md#uuid
func UUID() Node { return Leaf(&uuidType{}) }

type uuidType format.UUIDType

func (t *uuidType) String() string { return (*format.UUIDType)(t).String() }

func (t *uuidType) Kind() Kind { return FixedLenByteArray }

func (t *uuidType) Length() int { return 16 }

func (t *uuidType) EstimateSize(n int) int64 { return 16 * int64(n) }

func (t *uuidType) Compare(a, b Value) int {
	return compareBE128((*[16]byte)(a.ByteArray()), (*[16]byte)(b.ByteArray()))
}

func (t *uuidType) ColumnOrder() *format.ColumnOrder { return &typeDefinedColumnOrder }

func (t *uuidType) PhysicalType() *format.Type { return &physicalTypes[FixedLenByteArray] }

func (t *uuidType) LogicalType() *format.LogicalType {
	return &format.LogicalType{UUID: (*format.UUIDType)(t)}
}

func (t *uuidType) ConvertedType() *deprecated.ConvertedType { return nil }

func (t *uuidType) NewColumnIndexer(sizeLimit int) ColumnIndexer {
	return newBE128ColumnIndexer()
}

func (t *uuidType) NewDictionary(columnIndex, numValues int, data encoding.Values) Dictionary {
	return newBE128Dictionary(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t *uuidType) NewColumnBuffer(columnIndex, numValues int) ColumnBuffer {
	return newBE128ColumnBuffer(t, makeColumnIndex(columnIndex), makeNumValues(numValues))
}

func (t *uuidType) NewPage(columnIndex, numValues int, data encoding.Values) Page {
	return newBE128Page(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t *uuidType) NewValues(values []byte, _ []uint32) encoding.Values {
	return encoding.FixedLenByteArrayValues(values, 16)
}

func (t *uuidType) Encode(dst []byte, src encoding.Values, enc encoding.Encoding) ([]byte, error) {
	return encoding.EncodeFixedLenByteArray(dst, src, enc)
}

func (t *uuidType) Decode(dst encoding.Values, src []byte, enc encoding.Encoding) (encoding.Values, error) {
	return encoding.DecodeFixedLenByteArray(dst, src, enc)
}

// Enum constructs a leaf node with a logical type representing enumerations.
//
// https://github.com/apache/parquet-format/blob/master/LogicalTypes.md#enum
func Enum() Node { return Leaf(&enumType{}) }

type enumType format.EnumType

func (t *enumType) String() string { return (*format.EnumType)(t).String() }

func (t *enumType) Kind() Kind { return ByteArray }

func (t *enumType) Length() int { return 0 }

func (t *enumType) EstimateSize(n int) int64 { return 10 * int64(n) }

func (t *enumType) Compare(a, b Value) int {
	return bytes.Compare(a.ByteArray(), b.ByteArray())
}

func (t *enumType) ColumnOrder() *format.ColumnOrder {
	return &typeDefinedColumnOrder
}

func (t *enumType) PhysicalType() *format.Type {
	return &physicalTypes[ByteArray]
}

func (t *enumType) LogicalType() *format.LogicalType {
	return &format.LogicalType{Enum: (*format.EnumType)(t)}
}

func (t *enumType) ConvertedType() *deprecated.ConvertedType {
	return &convertedTypes[deprecated.Enum]
}

func (t *enumType) NewColumnIndexer(sizeLimit int) ColumnIndexer {
	return newByteArrayColumnIndexer(sizeLimit)
}

func (t *enumType) NewDictionary(columnIndex, numValues int, data encoding.Values) Dictionary {
	return newByteArrayDictionary(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t *enumType) NewColumnBuffer(columnIndex, numValues int) ColumnBuffer {
	return newByteArrayColumnBuffer(t, makeColumnIndex(columnIndex), makeNumValues(numValues))
}

func (t *enumType) NewPage(columnIndex, numValues int, data encoding.Values) Page {
	return newByteArrayPage(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t *enumType) NewValues(values []byte, offsets []uint32) encoding.Values {
	return encoding.ByteArrayValues(values, offsets)
}

func (t *enumType) Encode(dst []byte, src encoding.Values, enc encoding.Encoding) ([]byte, error) {
	return encoding.EncodeByteArray(dst, src, enc)
}

func (t *enumType) Decode(dst encoding.Values, src []byte, enc encoding.Encoding) (encoding.Values, error) {
	return encoding.DecodeByteArray(dst, src, enc)
}

// JSON constructs a leaf node of JSON logical type.
//
// https://github.com/apache/parquet-format/blob/master/LogicalTypes.md#json
func JSON() Node { return Leaf(&jsonType{}) }

type jsonType format.JsonType

func (t *jsonType) String() string { return (*format.JsonType)(t).String() }

func (t *jsonType) Kind() Kind { return ByteArray }

func (t *jsonType) Length() int { return 0 }

func (t *jsonType) EstimateSize(n int) int64 { return 10 * int64(n) }

func (t *jsonType) Compare(a, b Value) int {
	return bytes.Compare(a.ByteArray(), b.ByteArray())
}

func (t *jsonType) ColumnOrder() *format.ColumnOrder {
	return &typeDefinedColumnOrder
}

func (t *jsonType) PhysicalType() *format.Type {
	return &physicalTypes[ByteArray]
}

func (t *jsonType) LogicalType() *format.LogicalType {
	return &format.LogicalType{Json: (*format.JsonType)(t)}
}

func (t *jsonType) ConvertedType() *deprecated.ConvertedType {
	return &convertedTypes[deprecated.Json]
}

func (t *jsonType) NewColumnIndexer(sizeLimit int) ColumnIndexer {
	return newByteArrayColumnIndexer(sizeLimit)
}

func (t *jsonType) NewDictionary(columnIndex, numValues int, data encoding.Values) Dictionary {
	return newByteArrayDictionary(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t *jsonType) NewColumnBuffer(columnIndex, numValues int) ColumnBuffer {
	return newByteArrayColumnBuffer(t, makeColumnIndex(columnIndex), makeNumValues(numValues))
}

func (t *jsonType) NewPage(columnIndex, numValues int, data encoding.Values) Page {
	return newByteArrayPage(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t *jsonType) NewValues(values []byte, offsets []uint32) encoding.Values {
	return encoding.ByteArrayValues(values, offsets)
}

func (t *jsonType) Encode(dst []byte, src encoding.Values, enc encoding.Encoding) ([]byte, error) {
	return encoding.EncodeByteArray(dst, src, enc)
}

func (t *jsonType) Decode(dst encoding.Values, src []byte, enc encoding.Encoding) (encoding.Values, error) {
	return encoding.DecodeByteArray(dst, src, enc)
}

// BSON constructs a leaf node of BSON logical type.
//
// https://github.com/apache/parquet-format/blob/master/LogicalTypes.md#bson
func BSON() Node { return Leaf(&bsonType{}) }

type bsonType format.BsonType

func (t *bsonType) String() string { return (*format.BsonType)(t).String() }

func (t *bsonType) Kind() Kind { return ByteArray }

func (t *bsonType) Length() int { return 0 }

func (t *bsonType) EstimateSize(n int) int64 { return 10 * int64(n) }

func (t *bsonType) Compare(a, b Value) int {
	return bytes.Compare(a.ByteArray(), b.ByteArray())
}

func (t *bsonType) ColumnOrder() *format.ColumnOrder {
	return &typeDefinedColumnOrder
}

func (t *bsonType) PhysicalType() *format.Type {
	return &physicalTypes[ByteArray]
}

func (t *bsonType) LogicalType() *format.LogicalType {
	return &format.LogicalType{Bson: (*format.BsonType)(t)}
}

func (t *bsonType) ConvertedType() *deprecated.ConvertedType {
	return &convertedTypes[deprecated.Bson]
}

func (t *bsonType) NewColumnIndexer(sizeLimit int) ColumnIndexer {
	return newByteArrayColumnIndexer(sizeLimit)
}

func (t *bsonType) NewDictionary(columnIndex, numValues int, data encoding.Values) Dictionary {
	return newByteArrayDictionary(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t *bsonType) NewColumnBuffer(columnIndex, numValues int) ColumnBuffer {
	return newByteArrayColumnBuffer(t, makeColumnIndex(columnIndex), makeNumValues(numValues))
}

func (t *bsonType) NewPage(columnIndex, numValues int, data encoding.Values) Page {
	return newByteArrayPage(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t *bsonType) NewValues(values []byte, offsets []uint32) encoding.Values {
	return encoding.ByteArrayValues(values, offsets)
}

func (t *bsonType) Encode(dst []byte, src encoding.Values, enc encoding.Encoding) ([]byte, error) {
	return encoding.EncodeByteArray(dst, src, enc)
}

func (t *bsonType) Decode(dst encoding.Values, src []byte, enc encoding.Encoding) (encoding.Values, error) {
	return encoding.DecodeByteArray(dst, src, enc)
}

// Date constructs a leaf node of DATE logical type.
//
// https://github.com/apache/parquet-format/blob/master/LogicalTypes.md#date
func Date() Node { return Leaf(&dateType{}) }

type dateType format.DateType

func (t *dateType) String() string { return (*format.DateType)(t).String() }

func (t *dateType) Kind() Kind { return Int32 }

func (t *dateType) Length() int { return 32 }

func (t *dateType) EstimateSize(n int) int64 { return 4 * int64(n) }

func (t *dateType) Compare(a, b Value) int { return compareInt32(a.Int32(), b.Int32()) }

func (t *dateType) ColumnOrder() *format.ColumnOrder {
	return &typeDefinedColumnOrder
}

func (t *dateType) PhysicalType() *format.Type { return &physicalTypes[Int32] }

func (t *dateType) LogicalType() *format.LogicalType {
	return &format.LogicalType{Date: (*format.DateType)(t)}
}

func (t *dateType) ConvertedType() *deprecated.ConvertedType {
	return &convertedTypes[deprecated.Date]
}

func (t *dateType) NewColumnIndexer(sizeLimit int) ColumnIndexer {
	return newInt32ColumnIndexer()
}

func (t *dateType) NewColumnBuffer(columnIndex, numValues int) ColumnBuffer {
	return newInt32ColumnBuffer(t, makeColumnIndex(columnIndex), makeNumValues(numValues))
}

func (t *dateType) NewDictionary(columnIndex, numValues int, data encoding.Values) Dictionary {
	return newInt32Dictionary(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t *dateType) NewPage(columnIndex, numValues int, data encoding.Values) Page {
	return newInt32Page(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t *dateType) NewValues(values []byte, _ []uint32) encoding.Values {
	return encoding.Int32ValuesFromBytes(values)
}

func (t *dateType) Encode(dst []byte, src encoding.Values, enc encoding.Encoding) ([]byte, error) {
	return encoding.EncodeInt32(dst, src, enc)
}

func (t *dateType) Decode(dst encoding.Values, src []byte, enc encoding.Encoding) (encoding.Values, error) {
	return encoding.DecodeInt32(dst, src, enc)
}

// TimeUnit represents units of time in the parquet type system.
type TimeUnit interface {
	// Returns the precision of the time unit as a time.Duration value.
	Duration() time.Duration
	// Converts the TimeUnit value to its representation in the parquet thrift
	// format.
	TimeUnit() format.TimeUnit
}

var (
	Millisecond TimeUnit = &millisecond{}
	Microsecond TimeUnit = &microsecond{}
	Nanosecond  TimeUnit = &nanosecond{}
)

type millisecond format.MilliSeconds

func (u *millisecond) Duration() time.Duration { return time.Millisecond }
func (u *millisecond) TimeUnit() format.TimeUnit {
	return format.TimeUnit{Millis: (*format.MilliSeconds)(u)}
}

type microsecond format.MicroSeconds

func (u *microsecond) Duration() time.Duration { return time.Microsecond }
func (u *microsecond) TimeUnit() format.TimeUnit {
	return format.TimeUnit{Micros: (*format.MicroSeconds)(u)}
}

type nanosecond format.NanoSeconds

func (u *nanosecond) Duration() time.Duration { return time.Nanosecond }
func (u *nanosecond) TimeUnit() format.TimeUnit {
	return format.TimeUnit{Nanos: (*format.NanoSeconds)(u)}
}

// Time constructs a leaf node of TIME logical type.
//
// https://github.com/apache/parquet-format/blob/master/LogicalTypes.md#time
func Time(unit TimeUnit) Node {
	return Leaf(&timeType{IsAdjustedToUTC: true, Unit: unit.TimeUnit()})
}

type timeType format.TimeType

func (t *timeType) useInt32() bool {
	return t.Unit.Millis != nil
}

func (t *timeType) useInt64() bool {
	return t.Unit.Micros != nil
}

func (t *timeType) String() string {
	return (*format.TimeType)(t).String()
}

func (t *timeType) Kind() Kind {
	if t.useInt32() {
		return Int32
	} else {
		return Int64
	}
}

func (t *timeType) Length() int {
	if t.useInt32() {
		return 32
	} else {
		return 64
	}
}

func (t *timeType) EstimateSize(n int) int64 {
	if t.useInt32() {
		return 4 * int64(n)
	} else {
		return 8 * int64(n)
	}
}

func (t *timeType) Compare(a, b Value) int {
	if t.useInt32() {
		return compareInt32(a.Int32(), b.Int32())
	} else {
		return compareInt64(a.Int64(), b.Int64())
	}
}

func (t *timeType) ColumnOrder() *format.ColumnOrder {
	return &typeDefinedColumnOrder
}

func (t *timeType) PhysicalType() *format.Type {
	if t.useInt32() {
		return &physicalTypes[Int32]
	} else {
		return &physicalTypes[Int64]
	}
}

func (t *timeType) LogicalType() *format.LogicalType {
	return &format.LogicalType{Time: (*format.TimeType)(t)}
}

func (t *timeType) ConvertedType() *deprecated.ConvertedType {
	switch {
	case t.useInt32():
		return &convertedTypes[deprecated.TimeMillis]
	case t.useInt64():
		return &convertedTypes[deprecated.TimeMicros]
	default:
		return nil
	}
}

func (t *timeType) NewColumnIndexer(sizeLimit int) ColumnIndexer {
	if t.useInt32() {
		return newInt32ColumnIndexer()
	} else {
		return newInt64ColumnIndexer()
	}
}

func (t *timeType) NewColumnBuffer(columnIndex, numValues int) ColumnBuffer {
	if t.useInt32() {
		return newInt32ColumnBuffer(t, makeColumnIndex(columnIndex), makeNumValues(numValues))
	} else {
		return newInt64ColumnBuffer(t, makeColumnIndex(columnIndex), makeNumValues(numValues))
	}
}

func (t *timeType) NewDictionary(columnIndex, numValues int, data encoding.Values) Dictionary {
	if t.useInt32() {
		return newInt32Dictionary(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
	} else {
		return newInt64Dictionary(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
	}
}

func (t *timeType) NewPage(columnIndex, numValues int, data encoding.Values) Page {
	if t.useInt32() {
		return newInt32Page(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
	} else {
		return newInt64Page(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
	}
}

func (t *timeType) NewValues(values []byte, _ []uint32) encoding.Values {
	if t.useInt32() {
		return encoding.Int32ValuesFromBytes(values)
	} else {
		return encoding.Int64ValuesFromBytes(values)
	}
}

func (t *timeType) Encode(dst []byte, src encoding.Values, enc encoding.Encoding) ([]byte, error) {
	if t.useInt32() {
		return encoding.EncodeInt32(dst, src, enc)
	} else {
		return encoding.EncodeInt64(dst, src, enc)
	}
}

func (t *timeType) Decode(dst encoding.Values, src []byte, enc encoding.Encoding) (encoding.Values, error) {
	if t.useInt32() {
		return encoding.DecodeInt32(dst, src, enc)
	} else {
		return encoding.DecodeInt64(dst, src, enc)
	}
}

// Timestamp constructs of leaf node of TIMESTAMP logical type.
//
// https://github.com/apache/parquet-format/blob/master/LogicalTypes.md#timestamp
func Timestamp(unit TimeUnit) Node {
	return Leaf(&timestampType{IsAdjustedToUTC: true, Unit: unit.TimeUnit()})
}

type timestampType format.TimestampType

func (t *timestampType) String() string { return (*format.TimestampType)(t).String() }

func (t *timestampType) Kind() Kind { return Int64 }

func (t *timestampType) Length() int { return 64 }

func (t *timestampType) EstimateSize(n int) int64 { return 8 * int64(n) }

func (t *timestampType) Compare(a, b Value) int { return compareInt64(a.Int64(), b.Int64()) }

func (t *timestampType) ColumnOrder() *format.ColumnOrder { return &typeDefinedColumnOrder }

func (t *timestampType) PhysicalType() *format.Type { return &physicalTypes[Int64] }

func (t *timestampType) LogicalType() *format.LogicalType {
	return &format.LogicalType{Timestamp: (*format.TimestampType)(t)}
}

func (t *timestampType) ConvertedType() *deprecated.ConvertedType {
	switch {
	case t.Unit.Millis != nil:
		return &convertedTypes[deprecated.TimestampMillis]
	case t.Unit.Micros != nil:
		return &convertedTypes[deprecated.TimestampMicros]
	default:
		return nil
	}
}

func (t *timestampType) NewColumnIndexer(sizeLimit int) ColumnIndexer {
	return newInt64ColumnIndexer()
}

func (t *timestampType) NewColumnBuffer(columnIndex, numValues int) ColumnBuffer {
	return newInt64ColumnBuffer(t, makeColumnIndex(columnIndex), makeNumValues(numValues))
}

func (t *timestampType) NewDictionary(columnIndex, numValues int, data encoding.Values) Dictionary {
	return newInt64Dictionary(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t *timestampType) NewPage(columnIndex, numValues int, data encoding.Values) Page {
	return newInt64Page(t, makeColumnIndex(columnIndex), makeNumValues(numValues), data)
}

func (t *timestampType) NewValues(values []byte, _ []uint32) encoding.Values {
	return encoding.Int64ValuesFromBytes(values)
}

func (t *timestampType) Encode(dst []byte, src encoding.Values, enc encoding.Encoding) ([]byte, error) {
	return encoding.EncodeInt64(dst, src, enc)
}

func (t *timestampType) Decode(dst encoding.Values, src []byte, enc encoding.Encoding) (encoding.Values, error) {
	return encoding.DecodeInt64(dst, src, enc)
}

// List constructs a node of LIST logical type.
//
// https://github.com/apache/parquet-format/blob/master/LogicalTypes.md#lists
func List(of Node) Node {
	return listNode{Group{"list": Repeated(Group{"element": of})}}
}

type listNode struct{ Group }

func (listNode) Type() Type { return &listType{} }

type listType format.ListType

func (t *listType) String() string { return (*format.ListType)(t).String() }

func (t *listType) Kind() Kind { panic("cannot call Kind on parquet LIST type") }

func (t *listType) Length() int { return 0 }

func (t *listType) EstimateSize(int) int64 { return 0 }

func (t *listType) Compare(Value, Value) int { panic("cannot compare values on parquet LIST type") }

func (t *listType) ColumnOrder() *format.ColumnOrder { return nil }

func (t *listType) PhysicalType() *format.Type { return nil }

func (t *listType) LogicalType() *format.LogicalType {
	return &format.LogicalType{List: (*format.ListType)(t)}
}

func (t *listType) ConvertedType() *deprecated.ConvertedType {
	return &convertedTypes[deprecated.List]
}

func (t *listType) NewColumnIndexer(int) ColumnIndexer {
	panic("create create column indexer from parquet LIST type")
}

func (t *listType) NewDictionary(int, int, encoding.Values) Dictionary {
	panic("cannot create dictionary from parquet LIST type")
}

func (t *listType) NewColumnBuffer(int, int) ColumnBuffer {
	panic("cannot create column buffer from parquet LIST type")
}

func (t *listType) NewPage(int, int, encoding.Values) Page {
	panic("cannot create page from parquet LIST type")
}

func (t *listType) NewValues(values []byte, _ []uint32) encoding.Values {
	panic("cannot create values from parquet LIST type")
}

func (t *listType) Encode(_ []byte, _ encoding.Values, _ encoding.Encoding) ([]byte, error) {
	panic("cannot encode parquet LIST type")
}

func (t *listType) Decode(_ encoding.Values, _ []byte, _ encoding.Encoding) (encoding.Values, error) {
	panic("cannot decode parquet LIST type")
}

// Map constructs a node of MAP logical type.
//
// https://github.com/apache/parquet-format/blob/master/LogicalTypes.md#maps
func Map(key, value Node) Node {
	return mapNode{Group{
		"key_value": Repeated(Group{
			"key":   Required(key),
			"value": value,
		}),
	}}
}

type mapNode struct{ Group }

func (mapNode) Type() Type { return &mapType{} }

type mapType format.MapType

func (t *mapType) String() string { return (*format.MapType)(t).String() }

func (t *mapType) Kind() Kind { panic("cannot call Kind on parquet MAP type") }

func (t *mapType) Length() int { return 0 }

func (t *mapType) EstimateSize(int) int64 { return 0 }

func (t *mapType) Compare(Value, Value) int { panic("cannot compare values on parquet MAP type") }

func (t *mapType) ColumnOrder() *format.ColumnOrder { return nil }

func (t *mapType) PhysicalType() *format.Type { return nil }

func (t *mapType) LogicalType() *format.LogicalType {
	return &format.LogicalType{Map: (*format.MapType)(t)}
}

func (t *mapType) ConvertedType() *deprecated.ConvertedType {
	return &convertedTypes[deprecated.Map]
}

func (t *mapType) NewColumnIndexer(int) ColumnIndexer {
	panic("create create column indexer from parquet MAP type")
}

func (t *mapType) NewDictionary(int, int, encoding.Values) Dictionary {
	panic("cannot create dictionary from parquet MAP type")
}

func (t *mapType) NewColumnBuffer(int, int) ColumnBuffer {
	panic("cannot create column buffer from parquet MAP type")
}

func (t *mapType) NewPage(int, int, encoding.Values) Page {
	panic("cannot create page from parquet MAP type")
}

func (t *mapType) NewValues(values []byte, _ []uint32) encoding.Values {
	panic("cannot create values from parquet MAP type")
}

func (t *mapType) Encode(_ []byte, _ encoding.Values, _ encoding.Encoding) ([]byte, error) {
	panic("cannot encode parquet MAP type")
}

func (t *mapType) Decode(_ encoding.Values, _ []byte, _ encoding.Encoding) (encoding.Values, error) {
	panic("cannot decode parquet MAP type")
}

type nullType format.NullType

func (t *nullType) String() string { return (*format.NullType)(t).String() }

func (t *nullType) Kind() Kind { return -1 }

func (t *nullType) Length() int { return 0 }

func (t *nullType) EstimateSize(int) int64 { return 0 }

func (t *nullType) Compare(Value, Value) int { panic("cannot compare values on parquet NULL type") }

func (t *nullType) ColumnOrder() *format.ColumnOrder { return nil }

func (t *nullType) PhysicalType() *format.Type { return nil }

func (t *nullType) LogicalType() *format.LogicalType {
	return &format.LogicalType{Unknown: (*format.NullType)(t)}
}

func (t *nullType) ConvertedType() *deprecated.ConvertedType { return nil }

func (t *nullType) NewColumnIndexer(int) ColumnIndexer {
	panic("create create column indexer from parquet NULL type")
}

func (t *nullType) NewDictionary(int, int, encoding.Values) Dictionary {
	panic("cannot create dictionary from parquet NULL type")
}

func (t *nullType) NewColumnBuffer(int, int) ColumnBuffer {
	panic("cannot create column buffer from parquet NULL type")
}

func (t *nullType) NewPage(columnIndex, numValues int, _ encoding.Values) Page {
	return newNullPage(t, makeColumnIndex(columnIndex), makeNumValues(numValues))
}

func (t *nullType) NewValues(_ []byte, _ []uint32) encoding.Values {
	return encoding.Values{}
}

func (t *nullType) Encode(dst []byte, _ encoding.Values, _ encoding.Encoding) ([]byte, error) {
	return dst[:0], nil
}

func (t *nullType) Decode(dst encoding.Values, _ []byte, _ encoding.Encoding) (encoding.Values, error) {
	return dst, nil
}

type groupType struct{}

func (groupType) String() string { return "group" }

func (groupType) Kind() Kind {
	panic("cannot call Kind on parquet group")
}

func (groupType) Compare(Value, Value) int {
	panic("cannot compare values on parquet group")
}

func (groupType) NewColumnIndexer(int) ColumnIndexer {
	panic("cannot create column indexer from parquet group")
}

func (groupType) NewDictionary(int, int, encoding.Values) Dictionary {
	panic("cannot create dictionary from parquet group")
}

func (t groupType) NewColumnBuffer(int, int) ColumnBuffer {
	panic("cannot create column buffer from parquet group")
}

func (t groupType) NewPage(int, int, encoding.Values) Page {
	panic("cannot create page from parquet group")
}

func (t groupType) NewValues(_ []byte, _ []uint32) encoding.Values {
	panic("cannot create values from parquet group")
}

func (groupType) Encode(_ []byte, _ encoding.Values, _ encoding.Encoding) ([]byte, error) {
	panic("cannot encode parquet group")
}

func (groupType) Decode(_ encoding.Values, _ []byte, _ encoding.Encoding) (encoding.Values, error) {
	panic("cannot decode parquet group")
}

func (groupType) Length() int { return 0 }

func (groupType) EstimateSize(int) int64 { return 0 }

func (groupType) ColumnOrder() *format.ColumnOrder { return nil }

func (groupType) PhysicalType() *format.Type { return nil }

func (groupType) LogicalType() *format.LogicalType { return nil }

func (groupType) ConvertedType() *deprecated.ConvertedType { return nil }
