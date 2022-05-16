//go:build !go1.18

package parquet

import (
	"io"

	"github.com/segmentio/parquet-go/deprecated"
	"github.com/segmentio/parquet-go/encoding"
)

// The types below are implementations of the ColumnReader interface for each
// primitive type supported by parquet.
//
// The readers use an in-memory intermediary buffer to support decoding arrays
// of values from the underlying decoder, which are then boxed into the []Value
// buffer passed to ReadValues. When the program converts type checks the
// readers for more specific interfaces (e.g. parquet.Int32Reader), the values
// can be decoded directly from the underlying decoder. There is no need for
// the intermediary buffers so they are lazily allocated only if the ReadValues
// methods are called.

type booleanColumnReader struct {
	typ         Type
	decoder     encoding.Decoder
	buffer      []bool
	offset      int
	remain      int
	bufferSize  int
	columnIndex int16
}

func newBooleanColumnReader(typ Type, columnIndex int16, bufferSize int) *booleanColumnReader {
	return &booleanColumnReader{
		typ:         typ,
		bufferSize:  bufferSize,
		columnIndex: ^columnIndex,
	}
}

func (r *booleanColumnReader) Type() Type { return r.typ }

func (r *booleanColumnReader) Column() int { return int(^r.columnIndex) }

func (r *booleanColumnReader) ReadBooleans(values []bool) (n int, err error) {
	if r.offset < len(r.buffer) {
		n = copy(values, r.buffer[r.offset:])
		r.offset += n
		r.remain -= n
		values = values[n:]
	}
	if r.remain == 0 || r.decoder == nil {
		return n, io.EOF
	}
	values = values[:min(r.remain, len(values))]
	d, err := r.decoder.DecodeBoolean(values)
	r.remain -= d
	if r.remain == 0 && err == nil {
		err = io.EOF
	}
	return n + d, err
}

func (r *booleanColumnReader) ReadValues(values []Value) (n int, err error) {
	if cap(r.buffer) == 0 {
		r.buffer = make([]bool, 0, atLeastOne(r.bufferSize))
	}

	for {
		for r.offset < len(r.buffer) && n < len(values) {
			values[n] = makeValueBoolean(r.buffer[r.offset])
			values[n].columnIndex = r.columnIndex
			r.offset++
			r.remain--
			n++
		}

		if r.remain == 0 || r.decoder == nil {
			return n, io.EOF
		}
		if n == len(values) {
			return n, nil
		}

		length := min(r.remain, cap(r.buffer))
		buffer := r.buffer[:length]
		d, err := r.decoder.DecodeBoolean(buffer)
		if d == 0 {
			return n, err
		}

		r.buffer = buffer[:d]
		r.offset = 0
	}
}

func (r *booleanColumnReader) Reset(numValues int, decoder encoding.Decoder) {
	r.decoder = decoder
	r.buffer = r.buffer[:0]
	r.offset = 0
	r.remain = numValues
}

type int32ColumnReader struct {
	typ         Type
	decoder     encoding.Decoder
	buffer      []int32
	offset      int
	remain      int
	bufferSize  int
	columnIndex int16
}

func newInt32ColumnReader(typ Type, columnIndex int16, bufferSize int) *int32ColumnReader {
	return &int32ColumnReader{
		typ:         typ,
		bufferSize:  bufferSize,
		columnIndex: ^columnIndex,
	}
}

func (r *int32ColumnReader) Type() Type { return r.typ }

func (r *int32ColumnReader) Column() int { return int(^r.columnIndex) }

func (r *int32ColumnReader) ReadInt32s(values []int32) (n int, err error) {
	if r.offset < len(r.buffer) {
		n = copy(values, r.buffer[r.offset:])
		r.offset += n
		r.remain -= n
		values = values[n:]
	}
	if r.remain == 0 || r.decoder == nil {
		err = io.EOF
	} else {
		var d int
		values = values[:min(r.remain, len(values))]
		d, err = r.decoder.DecodeInt32(values)
		n += d
		r.remain -= d
		if r.remain == 0 && err == nil {
			err = io.EOF
		}
	}
	return n, err
}

func (r *int32ColumnReader) ReadValues(values []Value) (n int, err error) {
	if cap(r.buffer) == 0 {
		r.buffer = make([]int32, 0, atLeastOne(r.bufferSize))
	}

	for {
		for r.offset < len(r.buffer) && n < len(values) {
			values[n] = makeValueInt32(r.buffer[r.offset])
			values[n].columnIndex = r.columnIndex
			r.offset++
			r.remain--
			n++
		}

		if r.remain == 0 || r.decoder == nil {
			return n, io.EOF
		}
		if n == len(values) {
			return n, nil
		}

		length := min(r.remain, cap(r.buffer))
		buffer := r.buffer[:length]
		d, err := r.decoder.DecodeInt32(buffer)
		if d == 0 {
			return n, err
		}

		r.buffer = buffer[:d]
		r.offset = 0
	}
}

func (r *int32ColumnReader) Reset(numValues int, decoder encoding.Decoder) {
	r.decoder = decoder
	r.buffer = r.buffer[:0]
	r.offset = 0
	r.remain = numValues
}

type int64ColumnReader struct {
	typ         Type
	decoder     encoding.Decoder
	buffer      []int64
	offset      int
	remain      int
	bufferSize  int
	columnIndex int16
}

func newInt64ColumnReader(typ Type, columnIndex int16, bufferSize int) *int64ColumnReader {
	return &int64ColumnReader{
		typ:         typ,
		bufferSize:  bufferSize,
		columnIndex: ^columnIndex,
	}
}

func (r *int64ColumnReader) Type() Type { return r.typ }

func (r *int64ColumnReader) Column() int { return int(^r.columnIndex) }

func (r *int64ColumnReader) ReadInt64s(values []int64) (n int, err error) {
	if r.offset < len(r.buffer) {
		n = copy(values, r.buffer[r.offset:])
		r.offset += n
		r.remain -= n
		values = values[n:]
	}
	if r.remain == 0 || r.decoder == nil {
		err = io.EOF
	} else {
		var d int
		values = values[:min(r.remain, len(values))]
		d, err = r.decoder.DecodeInt64(values)
		n += d
		r.remain -= d
		if r.remain == 0 && err == nil {
			err = io.EOF
		}
	}
	return n, err
}

func (r *int64ColumnReader) ReadValues(values []Value) (n int, err error) {
	if cap(r.buffer) == 0 {
		r.buffer = make([]int64, 0, atLeastOne(r.bufferSize))
	}

	for {
		for r.offset < len(r.buffer) && n < len(values) {
			values[n] = makeValueInt64(r.buffer[r.offset])
			values[n].columnIndex = r.columnIndex
			r.offset++
			r.remain--
			n++
		}

		if r.remain == 0 || r.decoder == nil {
			return n, io.EOF
		}
		if n == len(values) {
			return n, nil
		}

		length := min(r.remain, cap(r.buffer))
		buffer := r.buffer[:length]
		d, err := r.decoder.DecodeInt64(buffer)
		if d == 0 {
			return n, err
		}

		r.buffer = buffer[:d]
		r.offset = 0
	}
}

func (r *int64ColumnReader) Reset(numValues int, decoder encoding.Decoder) {
	r.decoder = decoder
	r.buffer = r.buffer[:0]
	r.offset = 0
	r.remain = numValues
}

type int96ColumnReader struct {
	typ         Type
	decoder     encoding.Decoder
	buffer      []deprecated.Int96
	offset      int
	remain      int
	bufferSize  int
	columnIndex int16
}

func newInt96ColumnReader(typ Type, columnIndex int16, bufferSize int) *int96ColumnReader {
	return &int96ColumnReader{
		typ:         typ,
		bufferSize:  bufferSize,
		columnIndex: ^columnIndex,
	}
}

func (r *int96ColumnReader) Type() Type { return r.typ }

func (r *int96ColumnReader) Column() int { return int(^r.columnIndex) }

func (r *int96ColumnReader) ReadInt96s(values []deprecated.Int96) (n int, err error) {
	if r.offset < len(r.buffer) {
		n = copy(values, r.buffer[r.offset:])
		r.offset += n
		r.remain -= n
		values = values[n:]
	}
	if r.remain == 0 || r.decoder == nil {
		err = io.EOF
	} else {
		var d int
		values = values[:min(r.remain, len(values))]
		d, err = r.decoder.DecodeInt96(values)
		n += d
		r.remain -= d
		if r.remain == 0 && err == nil {
			err = io.EOF
		}
	}
	return n, err
}

func (r *int96ColumnReader) ReadValues(values []Value) (n int, err error) {
	if cap(r.buffer) == 0 {
		r.buffer = make([]deprecated.Int96, 0, atLeastOne(r.bufferSize))
	}

	for {
		for r.offset < len(r.buffer) && n < len(values) {
			values[n] = makeValueInt96(r.buffer[r.offset])
			values[n].columnIndex = r.columnIndex
			r.offset++
			r.remain--
			n++
		}

		if r.remain == 0 || r.decoder == nil {
			return n, io.EOF
		}
		if n == len(values) {
			return n, nil
		}

		length := min(r.remain, cap(r.buffer))
		buffer := r.buffer[:length]
		d, err := r.decoder.DecodeInt96(buffer)
		if d == 0 {
			return n, err
		}

		r.buffer = buffer[:d]
		r.offset = 0
	}
}

func (r *int96ColumnReader) Reset(numValues int, decoder encoding.Decoder) {
	r.decoder = decoder
	r.buffer = r.buffer[:0]
	r.offset = 0
	r.remain = numValues
}

type floatColumnReader struct {
	typ         Type
	decoder     encoding.Decoder
	buffer      []float32
	offset      int
	remain      int
	bufferSize  int
	columnIndex int16
}

func newFloatColumnReader(typ Type, columnIndex int16, bufferSize int) *floatColumnReader {
	return &floatColumnReader{
		typ:         typ,
		bufferSize:  bufferSize,
		columnIndex: ^columnIndex,
	}
}

func (r *floatColumnReader) Type() Type { return r.typ }

func (r *floatColumnReader) Column() int { return int(^r.columnIndex) }

func (r *floatColumnReader) ReadFloats(values []float32) (n int, err error) {
	if r.offset < len(r.buffer) {
		n = copy(values, r.buffer[r.offset:])
		r.offset += n
		r.remain -= n
		values = values[n:]
	}
	if r.remain == 0 || r.decoder == nil {
		err = io.EOF
	} else {
		var d int
		values = values[:min(r.remain, len(values))]
		d, err = r.decoder.DecodeFloat(values)
		n += d
		r.remain -= d
		if r.remain == 0 && err == nil {
			err = io.EOF
		}
	}
	return n, err
}

func (r *floatColumnReader) ReadValues(values []Value) (n int, err error) {
	if cap(r.buffer) == 0 {
		r.buffer = make([]float32, 0, atLeastOne(r.bufferSize))
	}

	for {
		for r.offset < len(r.buffer) && n < len(values) {
			values[n] = makeValueFloat(r.buffer[r.offset])
			values[n].columnIndex = r.columnIndex
			r.offset++
			r.remain--
			n++
		}

		if r.remain == 0 || r.decoder == nil {
			return n, io.EOF
		}
		if n == len(values) {
			return n, nil
		}

		length := min(r.remain, cap(r.buffer))
		buffer := r.buffer[:length]
		d, err := r.decoder.DecodeFloat(buffer)
		if d == 0 {
			return n, err
		}

		r.buffer = buffer[:d]
		r.offset = 0
	}
}

func (r *floatColumnReader) Reset(numValues int, decoder encoding.Decoder) {
	r.decoder = decoder
	r.buffer = r.buffer[:0]
	r.offset = 0
	r.remain = numValues
}

type doubleColumnReader struct {
	typ         Type
	decoder     encoding.Decoder
	buffer      []float64
	offset      int
	remain      int
	bufferSize  int
	columnIndex int16
}

func newDoubleColumnReader(typ Type, columnIndex int16, bufferSize int) *doubleColumnReader {
	return &doubleColumnReader{
		typ:         typ,
		bufferSize:  bufferSize,
		columnIndex: ^columnIndex,
	}
}

func (r *doubleColumnReader) Type() Type { return r.typ }

func (r *doubleColumnReader) Column() int { return int(^r.columnIndex) }

func (r *doubleColumnReader) ReadDoubles(values []float64) (n int, err error) {
	if r.offset < len(r.buffer) {
		n = copy(values, r.buffer[r.offset:])
		r.offset += n
		r.remain -= n
		values = values[n:]
	}
	if r.remain == 0 || r.decoder == nil {
		err = io.EOF
	} else {
		var d int
		values = values[:min(r.remain, len(values))]
		d, err = r.decoder.DecodeDouble(values)
		n += d
		r.remain -= d
		if r.remain == 0 && err == nil {
			err = io.EOF
		}
	}
	return n, err
}

func (r *doubleColumnReader) ReadValues(values []Value) (n int, err error) {
	if cap(r.buffer) == 0 {
		r.buffer = make([]float64, 0, atLeastOne(r.bufferSize))
	}

	for {
		for r.offset < len(r.buffer) && n < len(values) {
			values[n] = makeValueDouble(r.buffer[r.offset])
			values[n].columnIndex = r.columnIndex
			r.offset++
			r.remain--
			n++
		}

		if r.remain == 0 || r.decoder == nil {
			return n, io.EOF
		}
		if n == len(values) {
			return n, nil
		}

		length := min(r.remain, cap(r.buffer))
		buffer := r.buffer[:length]
		d, err := r.decoder.DecodeDouble(buffer)
		if d == 0 {
			return n, err
		}

		r.buffer = buffer[:d]
		r.offset = 0
	}
}

func (r *doubleColumnReader) Reset(numValues int, decoder encoding.Decoder) {
	r.decoder = decoder
	r.buffer = r.buffer[:0]
	r.offset = 0
	r.remain = numValues
}

var (
	_ BooleanReader           = (*booleanColumnReader)(nil)
	_ Int32Reader             = (*int32ColumnReader)(nil)
	_ Int64Reader             = (*int64ColumnReader)(nil)
	_ Int96Reader             = (*int96ColumnReader)(nil)
	_ FloatReader             = (*floatColumnReader)(nil)
	_ DoubleReader            = (*doubleColumnReader)(nil)
	_ ByteArrayReader         = (*byteArrayColumnReader)(nil)
	_ FixedLenByteArrayReader = (*fixedLenByteArrayColumnReader)(nil)
)
