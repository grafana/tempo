//go:build !go1.18

package parquet

import (
	"bytes"
	"fmt"

	"github.com/segmentio/parquet-go/deprecated"
	"github.com/segmentio/parquet-go/encoding"
	"github.com/segmentio/parquet-go/format"
)

var (
	BooleanType   Type = booleanType{}
	Int32Type     Type = int32Type{}
	Int64Type     Type = int64Type{}
	Int96Type     Type = int96Type{}
	FloatType     Type = floatType{}
	DoubleType    Type = doubleType{}
	ByteArrayType Type = byteArrayType{}
)

type primitiveType struct{}

func (t primitiveType) ColumnOrder() *format.ColumnOrder { return &typeDefinedColumnOrder }

func (t primitiveType) LogicalType() *format.LogicalType { return nil }

func (t primitiveType) ConvertedType() *deprecated.ConvertedType { return nil }

type booleanType struct{ primitiveType }

func (t booleanType) String() string { return "BOOLEAN" }

func (t booleanType) Kind() Kind { return Boolean }

func (t booleanType) Length() int { return 1 }

func (t booleanType) Compare(a, b Value) int {
	return compareBool(a.Boolean(), b.Boolean())
}

func (t booleanType) PhysicalType() *format.Type {
	return &physicalTypes[Boolean]
}

func (t booleanType) NewColumnIndexer(sizeLimit int) ColumnIndexer {
	return newBooleanColumnIndexer()
}

func (t booleanType) NewDictionary(columnIndex, bufferSize int) Dictionary {
	return newBooleanDictionary(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t booleanType) NewColumnBuffer(columnIndex, bufferSize int) ColumnBuffer {
	return newBooleanColumnBuffer(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t booleanType) NewColumnReader(columnIndex, bufferSize int) ColumnReader {
	return newBooleanColumnReader(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t booleanType) ReadDictionary(columnIndex, numValues int, decoder encoding.Decoder) (Dictionary, error) {
	return readBooleanDictionary(t, makeColumnIndex(columnIndex), numValues, decoder)
}

type int32Type struct{ primitiveType }

func (t int32Type) String() string { return "INT32" }

func (t int32Type) Kind() Kind { return Int32 }

func (t int32Type) Length() int { return 32 }

func (t int32Type) Compare(a, b Value) int {
	return compareInt32(a.Int32(), b.Int32())
}

func (t int32Type) PhysicalType() *format.Type {
	return &physicalTypes[Int32]
}

func (t int32Type) NewColumnIndexer(sizeLimit int) ColumnIndexer {
	return newInt32ColumnIndexer()
}

func (t int32Type) NewDictionary(columnIndex, bufferSize int) Dictionary {
	return newInt32Dictionary(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t int32Type) NewColumnBuffer(columnIndex, bufferSize int) ColumnBuffer {
	return newInt32ColumnBuffer(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t int32Type) NewColumnReader(columnIndex, bufferSize int) ColumnReader {
	return newInt32ColumnReader(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t int32Type) ReadDictionary(columnIndex, numValues int, decoder encoding.Decoder) (Dictionary, error) {
	return readInt32Dictionary(t, makeColumnIndex(columnIndex), numValues, decoder)
}

type int64Type struct{ primitiveType }

func (t int64Type) String() string { return "INT64" }

func (t int64Type) Kind() Kind { return Int64 }

func (t int64Type) Length() int { return 64 }

func (t int64Type) Compare(a, b Value) int {
	return compareInt64(a.Int64(), b.Int64())
}

func (t int64Type) PhysicalType() *format.Type {
	return &physicalTypes[Int64]
}

func (t int64Type) NewColumnIndexer(sizeLimit int) ColumnIndexer {
	return newInt64ColumnIndexer()
}

func (t int64Type) NewDictionary(columnIndex, bufferSize int) Dictionary {
	return newInt64Dictionary(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t int64Type) NewColumnBuffer(columnIndex, bufferSize int) ColumnBuffer {
	return newInt64ColumnBuffer(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t int64Type) NewColumnReader(columnIndex, bufferSize int) ColumnReader {
	return newInt64ColumnReader(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t int64Type) ReadDictionary(columnIndex, numValues int, decoder encoding.Decoder) (Dictionary, error) {
	return readInt64Dictionary(t, makeColumnIndex(columnIndex), numValues, decoder)
}

type int96Type struct{ primitiveType }

func (t int96Type) String() string { return "INT96" }

func (t int96Type) Kind() Kind { return Int96 }

func (t int96Type) Length() int { return 96 }

func (t int96Type) Compare(a, b Value) int {
	return compareInt96(a.Int96(), b.Int96())
}

func (t int96Type) PhysicalType() *format.Type {
	return &physicalTypes[Int96]
}

func (t int96Type) NewColumnIndexer(sizeLimit int) ColumnIndexer {
	return newInt96ColumnIndexer()
}

func (t int96Type) NewDictionary(columnIndex, bufferSize int) Dictionary {
	return newInt96Dictionary(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t int96Type) NewColumnBuffer(columnIndex, bufferSize int) ColumnBuffer {
	return newInt96ColumnBuffer(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t int96Type) NewColumnReader(columnIndex, bufferSize int) ColumnReader {
	return newInt96ColumnReader(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t int96Type) ReadDictionary(columnIndex, numValues int, decoder encoding.Decoder) (Dictionary, error) {
	return readInt96Dictionary(t, makeColumnIndex(columnIndex), numValues, decoder)
}

type floatType struct{ primitiveType }

func (t floatType) String() string { return "FLOAT" }

func (t floatType) Kind() Kind { return Float }

func (t floatType) Length() int { return 32 }

func (t floatType) Compare(a, b Value) int {
	return compareFloat32(a.Float(), b.Float())
}

func (t floatType) PhysicalType() *format.Type {
	return &physicalTypes[Float]
}

func (t floatType) NewColumnIndexer(sizeLimit int) ColumnIndexer {
	return newFloatColumnIndexer()
}

func (t floatType) NewDictionary(columnIndex, bufferSize int) Dictionary {
	return newFloatDictionary(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t floatType) NewColumnBuffer(columnIndex, bufferSize int) ColumnBuffer {
	return newFloatColumnBuffer(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t floatType) NewColumnReader(columnIndex, bufferSize int) ColumnReader {
	return newFloatColumnReader(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t floatType) ReadDictionary(columnIndex, numValues int, decoder encoding.Decoder) (Dictionary, error) {
	return readFloatDictionary(t, makeColumnIndex(columnIndex), numValues, decoder)
}

type doubleType struct{ primitiveType }

func (t doubleType) String() string { return "DOUBLE" }

func (t doubleType) Kind() Kind { return Double }

func (t doubleType) Length() int { return 64 }

func (t doubleType) Compare(a, b Value) int {
	return compareFloat64(a.Double(), b.Double())
}

func (t doubleType) PhysicalType() *format.Type { return &physicalTypes[Double] }

func (t doubleType) NewColumnIndexer(sizeLimit int) ColumnIndexer {
	return newDoubleColumnIndexer()
}

func (t doubleType) NewDictionary(columnIndex, bufferSize int) Dictionary {
	return newDoubleDictionary(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t doubleType) NewColumnBuffer(columnIndex, bufferSize int) ColumnBuffer {
	return newDoubleColumnBuffer(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t doubleType) NewColumnReader(columnIndex, bufferSize int) ColumnReader {
	return newDoubleColumnReader(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t doubleType) ReadDictionary(columnIndex, numValues int, decoder encoding.Decoder) (Dictionary, error) {
	return readDoubleDictionary(t, makeColumnIndex(columnIndex), numValues, decoder)
}

type byteArrayType struct{ primitiveType }

func (t byteArrayType) String() string { return "BYTE_ARRAY" }

func (t byteArrayType) Kind() Kind { return ByteArray }

func (t byteArrayType) Length() int { return 0 }

func (t byteArrayType) Compare(a, b Value) int {
	return bytes.Compare(a.ByteArray(), b.ByteArray())
}

func (t byteArrayType) PhysicalType() *format.Type {
	return &physicalTypes[ByteArray]
}

func (t byteArrayType) NewColumnIndexer(sizeLimit int) ColumnIndexer {
	return newByteArrayColumnIndexer(sizeLimit)
}

func (t byteArrayType) NewDictionary(columnIndex, bufferSize int) Dictionary {
	return newByteArrayDictionary(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t byteArrayType) NewColumnBuffer(columnIndex, bufferSize int) ColumnBuffer {
	return newByteArrayColumnBuffer(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t byteArrayType) NewColumnReader(columnIndex, bufferSize int) ColumnReader {
	return newByteArrayColumnReader(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t byteArrayType) ReadDictionary(columnIndex, numValues int, decoder encoding.Decoder) (Dictionary, error) {
	return readByteArrayDictionary(t, makeColumnIndex(columnIndex), numValues, decoder)
}

type fixedLenByteArrayType struct {
	primitiveType
	length int
}

func (t *fixedLenByteArrayType) String() string {
	return fmt.Sprintf("FIXED_LEN_BYTE_ARRAY(%d)", t.length)
}

func (t *fixedLenByteArrayType) Kind() Kind { return FixedLenByteArray }

func (t *fixedLenByteArrayType) Length() int { return t.length }

func (t *fixedLenByteArrayType) Compare(a, b Value) int {
	return bytes.Compare(a.ByteArray(), b.ByteArray())
}

func (t *fixedLenByteArrayType) PhysicalType() *format.Type {
	return &physicalTypes[FixedLenByteArray]
}

func (t *fixedLenByteArrayType) NewColumnIndexer(sizeLimit int) ColumnIndexer {
	return newFixedLenByteArrayColumnIndexer(t.length, sizeLimit)
}

func (t *fixedLenByteArrayType) NewDictionary(columnIndex, bufferSize int) Dictionary {
	return newFixedLenByteArrayDictionary(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t *fixedLenByteArrayType) NewColumnBuffer(columnIndex, bufferSize int) ColumnBuffer {
	return newFixedLenByteArrayColumnBuffer(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t *fixedLenByteArrayType) NewColumnReader(columnIndex, bufferSize int) ColumnReader {
	return newFixedLenByteArrayColumnReader(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t *fixedLenByteArrayType) ReadDictionary(columnIndex, numValues int, decoder encoding.Decoder) (Dictionary, error) {
	return readFixedLenByteArrayDictionary(t, makeColumnIndex(columnIndex), numValues, decoder)
}

// FixedLenByteArrayType constructs a type for fixed-length values of the given
// size (in bytes).
func FixedLenByteArrayType(length int) Type {
	return &fixedLenByteArrayType{length: length}
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

func (t *intType) NewDictionary(columnIndex, bufferSize int) Dictionary {
	if t.IsSigned {
		if t.BitWidth == 64 {
			return newInt64Dictionary(t, makeColumnIndex(columnIndex), bufferSize)
		} else {
			return newInt32Dictionary(t, makeColumnIndex(columnIndex), bufferSize)
		}
	} else {
		if t.BitWidth == 64 {
			return newUint64Dictionary(t, makeColumnIndex(columnIndex), bufferSize)
		} else {
			return newUint32Dictionary(t, makeColumnIndex(columnIndex), bufferSize)
		}
	}
}

func (t *intType) NewColumnBuffer(columnIndex, bufferSize int) ColumnBuffer {
	if t.IsSigned {
		if t.BitWidth == 64 {
			return newInt64ColumnBuffer(t, makeColumnIndex(columnIndex), bufferSize)
		} else {
			return newInt32ColumnBuffer(t, makeColumnIndex(columnIndex), bufferSize)
		}
	} else {
		if t.BitWidth == 64 {
			return newUint64ColumnBuffer(t, makeColumnIndex(columnIndex), bufferSize)
		} else {
			return newUint32ColumnBuffer(t, makeColumnIndex(columnIndex), bufferSize)
		}
	}
}

func (t *intType) NewColumnReader(columnIndex, bufferSize int) ColumnReader {
	if t.BitWidth == 64 {
		return newInt64ColumnReader(t, makeColumnIndex(columnIndex), bufferSize)
	} else {
		return newInt32ColumnReader(t, makeColumnIndex(columnIndex), bufferSize)
	}
}

func (t *intType) ReadDictionary(columnIndex, numValues int, decoder encoding.Decoder) (Dictionary, error) {
	if t.IsSigned {
		if t.BitWidth == 64 {
			return readInt64Dictionary(t, makeColumnIndex(columnIndex), numValues, decoder)
		} else {
			return readInt32Dictionary(t, makeColumnIndex(columnIndex), numValues, decoder)
		}
	} else {
		if t.BitWidth == 64 {
			return readUint64Dictionary(t, makeColumnIndex(columnIndex), numValues, decoder)
		} else {
			return readUint32Dictionary(t, makeColumnIndex(columnIndex), numValues, decoder)
		}
	}
}

func (t *dateType) NewColumnIndexer(sizeLimit int) ColumnIndexer {
	return newInt32ColumnIndexer()
}

func (t *dateType) NewDictionary(columnIndex, bufferSize int) Dictionary {
	return newInt32Dictionary(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t *dateType) NewColumnBuffer(columnIndex, bufferSize int) ColumnBuffer {
	return newInt32ColumnBuffer(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t *dateType) NewColumnReader(columnIndex, bufferSize int) ColumnReader {
	return newInt32ColumnReader(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t *dateType) ReadDictionary(columnIndex, numValues int, decoder encoding.Decoder) (Dictionary, error) {
	return readInt32Dictionary(t, makeColumnIndex(columnIndex), numValues, decoder)
}

func (t *timeType) NewColumnIndexer(sizeLimit int) ColumnIndexer {
	if t.Unit.Millis != nil {
		return newInt32ColumnIndexer()
	} else {
		return newInt64ColumnIndexer()
	}
}

func (t *timeType) NewDictionary(columnIndex, bufferSize int) Dictionary {
	if t.Unit.Millis != nil {
		return newInt32Dictionary(t, makeColumnIndex(columnIndex), bufferSize)
	} else {
		return newInt64Dictionary(t, makeColumnIndex(columnIndex), bufferSize)
	}
}

func (t *timeType) NewColumnBuffer(columnIndex, bufferSize int) ColumnBuffer {
	if t.Unit.Millis != nil {
		return newInt32ColumnBuffer(t, makeColumnIndex(columnIndex), bufferSize)
	} else {
		return newInt64ColumnBuffer(t, makeColumnIndex(columnIndex), bufferSize)
	}
}

func (t *timeType) NewColumnReader(columnIndex, bufferSize int) ColumnReader {
	if t.Unit.Millis != nil {
		return newInt32ColumnReader(t, makeColumnIndex(columnIndex), bufferSize)
	} else {
		return newInt64ColumnReader(t, makeColumnIndex(columnIndex), bufferSize)
	}
}

func (t *timeType) ReadDictionary(columnIndex, numValues int, decoder encoding.Decoder) (Dictionary, error) {
	if t.Unit.Millis != nil {
		return readInt32Dictionary(t, makeColumnIndex(columnIndex), numValues, decoder)
	} else {
		return readInt64Dictionary(t, makeColumnIndex(columnIndex), numValues, decoder)
	}
}

func (t *timestampType) NewColumnIndexer(sizeLimit int) ColumnIndexer {
	return newInt64ColumnIndexer()
}

func (t *timestampType) NewDictionary(columnIndex, bufferSize int) Dictionary {
	return newInt64Dictionary(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t *timestampType) NewColumnBuffer(columnIndex, bufferSize int) ColumnBuffer {
	return newInt64ColumnBuffer(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t *timestampType) NewColumnReader(columnIndex, bufferSize int) ColumnReader {
	return newInt64ColumnReader(t, makeColumnIndex(columnIndex), bufferSize)
}

func (t *timestampType) ReadDictionary(columnIndex, numValues int, decoder encoding.Decoder) (Dictionary, error) {
	return readInt64Dictionary(t, makeColumnIndex(columnIndex), numValues, decoder)
}
