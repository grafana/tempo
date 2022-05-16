//go:build go1.18

package parquet

import "github.com/segmentio/parquet-go/encoding/plain"

// RequiredReader is a parameterized interface implemented by ValueReader
// instances which exposes the content of a column as array of Go values of the
// type parameter T.
type RequiredReader[T plain.Type] interface {
	// Read values into the data slice, returning the number of values read, or
	// an error if less than len(data) values could be read, or io.EOF if the
	// end of the sequence was reached.
	//
	// For columns of type BYTE_ARRAY and FIXED_LEN_BYTE_ARRAY, T is byte and
	// the data is PLAIN encoded.
	//
	// If the column is of type FIXED_LEN_BYTE_ARRAY, the data slice length must
	// be a multiple of the column size.
	ReadRequired(data []T) (int, error)
}

// RequiredWriter is a parameterized interface implemented by ValueWriter
// instances which allows writing arrays of Go values of the type parameter T.
type RequiredWriter[T plain.Type] interface {
	// Write values from the data slice, returning the number of values written,
	// or an error if less than len(data) values were written.
	//
	// For columns of type BYTE_ARRAY and FIXED_LEN_BYTE_ARRAY, T is byte and
	// the data is PLAIN encoded.
	//
	// If the column is of type FIXED_LEN_BYTE_ARRAY, the data slice length must
	// be a multiple of the column size.
	WriteRequired(data []T) (int, error)
}
