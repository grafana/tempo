//go:build !go1.18

package parquet

import "github.com/segmentio/parquet-go/deprecated"

// BooleanReader is an interface implemented by ValueReader instances which
// expose the content of a column of boolean values.
type BooleanReader interface {
	// Read boolean values into the buffer passed as argument.
	//
	// The method returns io.EOF when all values have been read.
	ReadBooleans(values []bool) (int, error)
}

// BooleanWriter is an interface implemented by ValueWriter instances which
// support writing columns of boolean values.
type BooleanWriter interface {
	// Write boolean values.
	//
	// The method returns the number of values written, and any error that
	// occured while writing the values.
	WriteBooleans(values []bool) (int, error)
}

// Int32Reader is an interface implemented by ValueReader instances which expose
// the content of a column of int32 values.
type Int32Reader interface {
	// Read 32 bits integer values into the buffer passed as argument.
	//
	// The method returns io.EOF when all values have been read.
	ReadInt32s(values []int32) (int, error)
}

// Int32Writer is an interface implemented by ValueWriter instances which
// support writing columns of 32 bits signed integer values.
type Int32Writer interface {
	// Write 32 bits signed integer values.
	//
	// The method returns the number of values written, and any error that
	// occured while writing the values.
	WriteInt32s(values []int32) (int, error)
}

// Int64Reader is an interface implemented by ValueReader instances which expose
// the content of a column of int64 values.
type Int64Reader interface {
	// Read 64 bits integer values into the buffer passed as argument.
	//
	// The method returns io.EOF when all values have been read.
	ReadInt64s(values []int64) (int, error)
}

// Int64Writer is an interface implemented by ValueWriter instances which
// support writing columns of 64 bits signed integer values.
type Int64Writer interface {
	// Write 64 bits signed integer values.
	//
	// The method returns the number of values written, and any error that
	// occured while writing the values.
	WriteInt64s(values []int64) (int, error)
}

// Int96Reader is an interface implemented by ValueReader instances which expose
// the content of a column of int96 values.
type Int96Reader interface {
	// Read 96 bits integer values into the buffer passed as argument.
	//
	// The method returns io.EOF when all values have been read.
	ReadInt96s(values []deprecated.Int96) (int, error)
}

// Int96Writer is an interface implemented by ValueWriter instances which
// support writing columns of 96 bits signed integer values.
type Int96Writer interface {
	// Write 96 bits signed integer values.
	//
	// The method returns the number of values written, and any error that
	// occured while writing the values.
	WriteInt96s(values []deprecated.Int96) (int, error)
}

// FloatReader is an interface implemented by ValueReader instances which expose
// the content of a column of single-precision floating point values.
type FloatReader interface {
	// Read single-precision floating point values into the buffer passed as
	// argument.
	//
	// The method returns io.EOF when all values have been read.
	ReadFloats(values []float32) (int, error)
}

// FloatWriter is an interface implemented by ValueWriter instances which
// support writing columns of single-precision floating point values.
type FloatWriter interface {
	// Write single-precision floating point values.
	//
	// The method returns the number of values written, and any error that
	// occured while writing the values.
	WriteFloats(values []float32) (int, error)
}

// DoubleReader is an interface implemented by ValueReader instances which
// expose the content of a column of double-precision float point values.
type DoubleReader interface {
	// Read double-precision floating point values into the buffer passed as
	// argument.
	//
	// The method returns io.EOF when all values have been read.
	ReadDoubles(values []float64) (int, error)
}

// DoubleWriter is an interface implemented by ValueWriter instances which
// support writing columns of double-precision floating point values.
type DoubleWriter interface {
	// Write double-precision floating point values.
	//
	// The method returns the number of values written, and any error that
	// occured while writing the values.
	WriteDoubles(values []float64) (int, error)
}

// ByteArrayReader is an interface implemented by ValueReader instances which
// expose the content of a column of variable length byte array values.
type ByteArrayReader interface {
	// Read values into the byte buffer passed as argument, returning the number
	// of values written to the buffer (not the number of bytes). Values are
	// written using the PLAIN encoding, each byte array prefixed with its
	// length encoded as a 4 bytes little endian unsigned integer.
	//
	// The method returns io.EOF when all values have been read.
	//
	// If the buffer was not empty, but too small to hold at least one value,
	// io.ErrShortBuffer is returned.
	ReadByteArrays(values []byte) (int, error)
}

// ByteArrayWriter is an interface implemented by ValueWriter instances which
// support writing columns of variable length byte array values.
type ByteArrayWriter interface {
	// Write variable length byte array values.
	//
	// The values passed as input must be laid out using the PLAIN encoding,
	// with each byte array prefixed with the four bytes little endian unsigned
	// integer length.
	//
	// The method returns the number of values written to the underlying column
	// (not the number of bytes), or any error that occured while attempting to
	// write the values.
	WriteByteArrays(values []byte) (int, error)
}

// FixedLenByteArrayReader is an interface implemented by ValueReader instances
// which expose the content of a column of fixed length byte array values.
type FixedLenByteArrayReader interface {
	// Read values into the byte buffer passed as argument, returning the number
	// of values written to the buffer (not the number of bytes).
	//
	// The method returns io.EOF when all values have been read.
	//
	// If the buffer was not empty, but too small to hold at least one value,
	// io.ErrShortBuffer is returned.
	ReadFixedLenByteArrays(values []byte) (int, error)
}

// FixedLenByteArrayWriter is an interface implemented by ValueWriter instances
// which support writing columns of fixed length byte array values.
type FixedLenByteArrayWriter interface {
	// Writes the fixed length byte array values.
	//
	// The size of the values is assumed to be the same as the expected size of
	// items in the column. The method errors if the length of the input values
	// is not a multiple of the expected item size.
	WriteFixedLenByteArrays(values []byte) (int, error)
}
