//go:build !go1.18

package parquet

import (
	"fmt"
	"io"

	"github.com/segmentio/parquet-go/deprecated"
	"github.com/segmentio/parquet-go/internal/bits"
)

// =============================================================================
// The types below are in-memory implementations of the ColumnBuffer interface
// for each parquet type.
//
// These column buffers are created by calling NewColumnBuffer on parquet.Type
// instances; each parquet type manages to construct column buffers of the
// appropriate type, which ensures that we are packing as many values as we
// can in memory.
//
// See Type.NewColumnBuffer for details about how these types get created.
// =============================================================================

type booleanColumnBuffer struct {
	booleanPage
	typ Type
}

func newBooleanColumnBuffer(typ Type, columnIndex int16, bufferSize int) *booleanColumnBuffer {
	return &booleanColumnBuffer{
		booleanPage: booleanPage{
			values:      make([]bool, 0, bufferSize),
			columnIndex: ^columnIndex,
		},
		typ: typ,
	}
}

func (col *booleanColumnBuffer) Clone() ColumnBuffer {
	return &booleanColumnBuffer{
		booleanPage: booleanPage{
			values:      append([]bool{}, col.values...),
			columnIndex: col.columnIndex,
		},
		typ: col.typ,
	}
}

func (col *booleanColumnBuffer) Type() Type { return col.typ }

func (col *booleanColumnBuffer) ColumnIndex() ColumnIndex {
	return booleanColumnIndex{&col.booleanPage}
}

func (col *booleanColumnBuffer) OffsetIndex() OffsetIndex {
	return booleanOffsetIndex{&col.booleanPage}
}

func (col *booleanColumnBuffer) BloomFilter() BloomFilter { return nil }

func (col *booleanColumnBuffer) Dictionary() Dictionary { return nil }

func (col *booleanColumnBuffer) Pages() Pages { return onePage(col.Page()) }

func (col *booleanColumnBuffer) Page() BufferedPage { return &col.booleanPage }

func (col *booleanColumnBuffer) Reset() { col.values = col.values[:0] }

func (col *booleanColumnBuffer) Cap() int { return cap(col.values) }

func (col *booleanColumnBuffer) Len() int { return len(col.values) }

func (col *booleanColumnBuffer) Less(i, j int) bool {
	return col.values[i] != col.values[j] && !col.values[i]
}

func (col *booleanColumnBuffer) Swap(i, j int) {
	col.values[i], col.values[j] = col.values[j], col.values[i]
}

func (col *booleanColumnBuffer) Write(b []byte) (int, error) {
	return col.WriteBooleans(bits.BytesToBool(b))
}

func (col *booleanColumnBuffer) WriteBooleans(values []bool) (int, error) {
	col.values = append(col.values, values...)
	return len(values), nil
}

func (col *booleanColumnBuffer) WriteValues(values []Value) (int, error) {
	for _, v := range values {
		col.values = append(col.values, v.Boolean())
	}
	return len(values), nil
}

func (col *booleanColumnBuffer) ReadValuesAt(values []Value, offset int64) (n int, err error) {
	i := int(offset)
	switch {
	case i < 0:
		return 0, errRowIndexOutOfBounds(offset, int64(len(col.values)))
	case i >= len(col.values):
		return 0, io.EOF
	default:
		for n < len(values) && i < len(col.values) {
			values[n] = makeValueBoolean(col.values[i])
			values[n].columnIndex = col.columnIndex
			n++
			i++
		}
		if n < len(values) {
			err = io.EOF
		}
		return n, err
	}
}

type int32ColumnBuffer struct {
	int32Page
	typ Type
}

func newInt32ColumnBuffer(typ Type, columnIndex int16, bufferSize int) *int32ColumnBuffer {
	return &int32ColumnBuffer{
		int32Page: int32Page{
			values:      make([]int32, 0, bufferSize/4),
			columnIndex: ^columnIndex,
		},
		typ: typ,
	}
}

func (col *int32ColumnBuffer) Clone() ColumnBuffer {
	return &int32ColumnBuffer{
		int32Page: int32Page{
			values:      append([]int32{}, col.values...),
			columnIndex: col.columnIndex,
		},
		typ: col.typ,
	}
}

func (col *int32ColumnBuffer) Type() Type { return col.typ }

func (col *int32ColumnBuffer) ColumnIndex() ColumnIndex { return int32ColumnIndex{&col.int32Page} }

func (col *int32ColumnBuffer) OffsetIndex() OffsetIndex { return int32OffsetIndex{&col.int32Page} }

func (col *int32ColumnBuffer) BloomFilter() BloomFilter { return nil }

func (col *int32ColumnBuffer) Dictionary() Dictionary { return nil }

func (col *int32ColumnBuffer) Pages() Pages { return onePage(col.Page()) }

func (col *int32ColumnBuffer) Page() BufferedPage { return &col.int32Page }

func (col *int32ColumnBuffer) Reset() { col.values = col.values[:0] }

func (col *int32ColumnBuffer) Cap() int { return cap(col.values) }

func (col *int32ColumnBuffer) Len() int { return len(col.values) }

func (col *int32ColumnBuffer) Less(i, j int) bool { return col.values[i] < col.values[j] }

func (col *int32ColumnBuffer) Swap(i, j int) {
	col.values[i], col.values[j] = col.values[j], col.values[i]
}

func (col *int32ColumnBuffer) Write(b []byte) (int, error) {
	if (len(b) % 4) != 0 {
		return 0, fmt.Errorf("cannot write INT32 values from input of size %d", len(b))
	}
	col.values = append(col.values, bits.BytesToInt32(b)...)
	return len(b), nil
}

func (col *int32ColumnBuffer) WriteInt32s(values []int32) (int, error) {
	col.values = append(col.values, values...)
	return len(values), nil
}

func (col *int32ColumnBuffer) WriteValues(values []Value) (int, error) {
	for _, v := range values {
		col.values = append(col.values, v.Int32())
	}
	return len(values), nil
}

func (col *int32ColumnBuffer) ReadValuesAt(values []Value, offset int64) (n int, err error) {
	i := int(offset)
	switch {
	case i < 0:
		return 0, errRowIndexOutOfBounds(offset, int64(len(col.values)))
	case i >= len(col.values):
		return 0, io.EOF
	default:
		for n < len(values) && i < len(col.values) {
			values[n] = makeValueInt32(col.values[i])
			values[n].columnIndex = col.columnIndex
			n++
			i++
		}
		if n < len(values) {
			err = io.EOF
		}
		return n, err
	}
}

type int64ColumnBuffer struct {
	int64Page
	typ Type
}

func newInt64ColumnBuffer(typ Type, columnIndex int16, bufferSize int) *int64ColumnBuffer {
	return &int64ColumnBuffer{
		int64Page: int64Page{
			values:      make([]int64, 0, bufferSize/8),
			columnIndex: ^columnIndex,
		},
		typ: typ,
	}
}

func (col *int64ColumnBuffer) Clone() ColumnBuffer {
	return &int64ColumnBuffer{
		int64Page: int64Page{
			values:      append([]int64{}, col.values...),
			columnIndex: col.columnIndex,
		},
		typ: col.typ,
	}
}

func (col *int64ColumnBuffer) Type() Type { return col.typ }

func (col *int64ColumnBuffer) ColumnIndex() ColumnIndex { return int64ColumnIndex{&col.int64Page} }

func (col *int64ColumnBuffer) OffsetIndex() OffsetIndex { return int64OffsetIndex{&col.int64Page} }

func (col *int64ColumnBuffer) BloomFilter() BloomFilter { return nil }

func (col *int64ColumnBuffer) Dictionary() Dictionary { return nil }

func (col *int64ColumnBuffer) Pages() Pages { return onePage(col.Page()) }

func (col *int64ColumnBuffer) Page() BufferedPage { return &col.int64Page }

func (col *int64ColumnBuffer) Reset() { col.values = col.values[:0] }

func (col *int64ColumnBuffer) Cap() int { return cap(col.values) }

func (col *int64ColumnBuffer) Len() int { return len(col.values) }

func (col *int64ColumnBuffer) Less(i, j int) bool { return col.values[i] < col.values[j] }

func (col *int64ColumnBuffer) Swap(i, j int) {
	col.values[i], col.values[j] = col.values[j], col.values[i]
}

func (col *int64ColumnBuffer) Write(b []byte) (int, error) {
	if (len(b) % 8) != 0 {
		return 0, fmt.Errorf("cannot write INT64 values from input of size %d", len(b))
	}
	col.values = append(col.values, bits.BytesToInt64(b)...)
	return len(b), nil
}

func (col *int64ColumnBuffer) WriteInt64s(values []int64) (int, error) {
	col.values = append(col.values, values...)
	return len(values), nil
}

func (col *int64ColumnBuffer) WriteValues(values []Value) (int, error) {
	for _, v := range values {
		col.values = append(col.values, v.Int64())
	}
	return len(values), nil
}

func (col *int64ColumnBuffer) ReadValuesAt(values []Value, offset int64) (n int, err error) {
	i := int(offset)
	switch {
	case i < 0:
		return 0, errRowIndexOutOfBounds(offset, int64(len(col.values)))
	case i >= len(col.values):
		return 0, io.EOF
	default:
		for n < len(values) && i < len(col.values) {
			values[n] = makeValueInt64(col.values[i])
			values[n].columnIndex = col.columnIndex
			n++
			i++
		}
		if n < len(values) {
			err = io.EOF
		}
		return n, err
	}
}

type int96ColumnBuffer struct {
	int96Page
	typ Type
}

func newInt96ColumnBuffer(typ Type, columnIndex int16, bufferSize int) *int96ColumnBuffer {
	return &int96ColumnBuffer{
		int96Page: int96Page{
			values:      make([]deprecated.Int96, 0, bufferSize/12),
			columnIndex: ^columnIndex,
		},
		typ: typ,
	}
}

func (col *int96ColumnBuffer) Clone() ColumnBuffer {
	return &int96ColumnBuffer{
		int96Page: int96Page{
			values:      append([]deprecated.Int96{}, col.values...),
			columnIndex: col.columnIndex,
		},
		typ: col.typ,
	}
}

func (col *int96ColumnBuffer) Type() Type { return col.typ }

func (col *int96ColumnBuffer) ColumnIndex() ColumnIndex { return int96ColumnIndex{&col.int96Page} }

func (col *int96ColumnBuffer) OffsetIndex() OffsetIndex { return int96OffsetIndex{&col.int96Page} }

func (col *int96ColumnBuffer) BloomFilter() BloomFilter { return nil }

func (col *int96ColumnBuffer) Dictionary() Dictionary { return nil }

func (col *int96ColumnBuffer) Pages() Pages { return onePage(col.Page()) }

func (col *int96ColumnBuffer) Page() BufferedPage { return &col.int96Page }

func (col *int96ColumnBuffer) Reset() { col.values = col.values[:0] }

func (col *int96ColumnBuffer) Cap() int { return cap(col.values) }

func (col *int96ColumnBuffer) Len() int { return len(col.values) }

func (col *int96ColumnBuffer) Less(i, j int) bool { return col.values[i].Less(col.values[j]) }

func (col *int96ColumnBuffer) Swap(i, j int) {
	col.values[i], col.values[j] = col.values[j], col.values[i]
}

func (col *int96ColumnBuffer) Write(b []byte) (int, error) {
	if (len(b) % 12) != 0 {
		return 0, fmt.Errorf("cannot write INT96 values from input of size %d", len(b))
	}
	col.values = append(col.values, deprecated.BytesToInt96(b)...)
	return len(b), nil
}

func (col *int96ColumnBuffer) WriteInt96s(values []deprecated.Int96) (int, error) {
	col.values = append(col.values, values...)
	return len(values), nil
}

func (col *int96ColumnBuffer) WriteValues(values []Value) (int, error) {
	for _, v := range values {
		col.values = append(col.values, v.Int96())
	}
	return len(values), nil
}

func (col *int96ColumnBuffer) ReadValuesAt(values []Value, offset int64) (n int, err error) {
	i := int(offset)
	switch {
	case i < 0:
		return 0, errRowIndexOutOfBounds(offset, int64(len(col.values)))
	case i >= len(col.values):
		return 0, io.EOF
	default:
		for n < len(values) && i < len(col.values) {
			values[n] = makeValueInt96(col.values[i])
			values[n].columnIndex = col.columnIndex
			n++
			i++
		}
		if n < len(values) {
			err = io.EOF
		}
		return n, err
	}
}

type floatColumnBuffer struct {
	floatPage
	typ Type
}

func newFloatColumnBuffer(typ Type, columnIndex int16, bufferSize int) *floatColumnBuffer {
	return &floatColumnBuffer{
		floatPage: floatPage{
			values:      make([]float32, 0, bufferSize/4),
			columnIndex: ^columnIndex,
		},
		typ: typ,
	}
}

func (col *floatColumnBuffer) Clone() ColumnBuffer {
	return &floatColumnBuffer{
		floatPage: floatPage{
			values:      append([]float32{}, col.values...),
			columnIndex: col.columnIndex,
		},
		typ: col.typ,
	}
}

func (col *floatColumnBuffer) Type() Type { return col.typ }

func (col *floatColumnBuffer) ColumnIndex() ColumnIndex { return floatColumnIndex{&col.floatPage} }

func (col *floatColumnBuffer) OffsetIndex() OffsetIndex { return floatOffsetIndex{&col.floatPage} }

func (col *floatColumnBuffer) BloomFilter() BloomFilter { return nil }

func (col *floatColumnBuffer) Dictionary() Dictionary { return nil }

func (col *floatColumnBuffer) Pages() Pages { return onePage(col.Page()) }

func (col *floatColumnBuffer) Page() BufferedPage { return &col.floatPage }

func (col *floatColumnBuffer) Reset() { col.values = col.values[:0] }

func (col *floatColumnBuffer) Cap() int { return cap(col.values) }

func (col *floatColumnBuffer) Len() int { return len(col.values) }

func (col *floatColumnBuffer) Less(i, j int) bool { return col.values[i] < col.values[j] }

func (col *floatColumnBuffer) Swap(i, j int) {
	col.values[i], col.values[j] = col.values[j], col.values[i]
}

func (col *floatColumnBuffer) Write(b []byte) (int, error) {
	if (len(b) % 4) != 0 {
		return 0, fmt.Errorf("cannot write FLOAT values from input of size %d", len(b))
	}
	col.values = append(col.values, bits.BytesToFloat32(b)...)
	return len(b), nil
}

func (col *floatColumnBuffer) WriteFloats(values []float32) (int, error) {
	col.values = append(col.values, values...)
	return len(values), nil
}

func (col *floatColumnBuffer) WriteValues(values []Value) (int, error) {
	for _, v := range values {
		col.values = append(col.values, v.Float())
	}
	return len(values), nil
}

func (col *floatColumnBuffer) ReadValuesAt(values []Value, offset int64) (n int, err error) {
	i := int(offset)
	switch {
	case i < 0:
		return 0, errRowIndexOutOfBounds(offset, int64(len(col.values)))
	case i >= len(col.values):
		return 0, io.EOF
	default:
		for n < len(values) && i < len(col.values) {
			values[n] = makeValueFloat(col.values[i])
			values[n].columnIndex = col.columnIndex
			n++
			i++
		}
		if n < len(values) {
			err = io.EOF
		}
		return n, err
	}
}

type doubleColumnBuffer struct {
	doublePage
	typ Type
}

func newDoubleColumnBuffer(typ Type, columnIndex int16, bufferSize int) *doubleColumnBuffer {
	return &doubleColumnBuffer{
		doublePage: doublePage{
			values:      make([]float64, 0, bufferSize/8),
			columnIndex: ^columnIndex,
		},
		typ: typ,
	}
}

func (col *doubleColumnBuffer) Clone() ColumnBuffer {
	return &doubleColumnBuffer{
		doublePage: doublePage{
			values:      append([]float64{}, col.values...),
			columnIndex: col.columnIndex,
		},
		typ: col.typ,
	}
}

func (col *doubleColumnBuffer) Type() Type { return col.typ }

func (col *doubleColumnBuffer) ColumnIndex() ColumnIndex { return doubleColumnIndex{&col.doublePage} }

func (col *doubleColumnBuffer) OffsetIndex() OffsetIndex { return doubleOffsetIndex{&col.doublePage} }

func (col *doubleColumnBuffer) BloomFilter() BloomFilter { return nil }

func (col *doubleColumnBuffer) Dictionary() Dictionary { return nil }

func (col *doubleColumnBuffer) Pages() Pages { return onePage(col.Page()) }

func (col *doubleColumnBuffer) Page() BufferedPage { return &col.doublePage }

func (col *doubleColumnBuffer) Reset() { col.values = col.values[:0] }

func (col *doubleColumnBuffer) Cap() int { return cap(col.values) }

func (col *doubleColumnBuffer) Len() int { return len(col.values) }

func (col *doubleColumnBuffer) Less(i, j int) bool { return col.values[i] < col.values[j] }

func (col *doubleColumnBuffer) Swap(i, j int) {
	col.values[i], col.values[j] = col.values[j], col.values[i]
}

func (col *doubleColumnBuffer) Write(b []byte) (int, error) {
	if (len(b) % 8) != 0 {
		return 0, fmt.Errorf("cannot write DOUBLE values from input of size %d", len(b))
	}
	col.values = append(col.values, bits.BytesToFloat64(b)...)
	return len(b), nil
}

func (col *doubleColumnBuffer) WriteDoubles(values []float64) (int, error) {
	col.values = append(col.values, values...)
	return len(values), nil
}

func (col *doubleColumnBuffer) WriteValues(values []Value) (int, error) {
	for _, v := range values {
		col.values = append(col.values, v.Double())
	}
	return len(values), nil
}

func (col *doubleColumnBuffer) ReadValuesAt(values []Value, offset int64) (n int, err error) {
	i := int(offset)
	switch {
	case i < 0:
		return 0, errRowIndexOutOfBounds(offset, int64(len(col.values)))
	case i >= len(col.values):
		return 0, io.EOF
	default:
		for n < len(values) && i < len(col.values) {
			values[n] = makeValueDouble(col.values[i])
			values[n].columnIndex = col.columnIndex
			n++
			i++
		}
		if n < len(values) {
			err = io.EOF
		}
		return n, err
	}
}

type uint32ColumnBuffer struct{ *int32ColumnBuffer }

func newUint32ColumnBuffer(typ Type, columnIndex int16, bufferSize int) uint32ColumnBuffer {
	return uint32ColumnBuffer{newInt32ColumnBuffer(typ, columnIndex, bufferSize)}
}

func (col uint32ColumnBuffer) ColumnIndex() ColumnIndex { return uint32ColumnIndex{col.page()} }

func (col uint32ColumnBuffer) OffsetIndex() OffsetIndex { return uint32OffsetIndex{col.page()} }

func (col uint32ColumnBuffer) page() uint32Page { return uint32Page{&col.int32Page} }

func (col uint32ColumnBuffer) Page() BufferedPage { return col.page() }

func (col uint32ColumnBuffer) Pages() Pages { return onePage(col.page()) }

func (col uint32ColumnBuffer) Clone() ColumnBuffer {
	return uint32ColumnBuffer{col.int32ColumnBuffer.Clone().(*int32ColumnBuffer)}
}

func (col uint32ColumnBuffer) Less(i, j int) bool {
	return uint32(col.values[i]) < uint32(col.values[j])
}

type uint64ColumnBuffer struct{ *int64ColumnBuffer }

func newUint64ColumnBuffer(typ Type, columnIndex int16, bufferSize int) uint64ColumnBuffer {
	return uint64ColumnBuffer{newInt64ColumnBuffer(typ, columnIndex, bufferSize)}
}

func (col uint64ColumnBuffer) ColumnIndex() ColumnIndex { return uint64ColumnIndex{col.page()} }

func (col uint64ColumnBuffer) OffsetIndex() OffsetIndex { return uint64OffsetIndex{col.page()} }

func (col uint64ColumnBuffer) page() uint64Page { return uint64Page{&col.int64Page} }

func (col uint64ColumnBuffer) Page() BufferedPage { return col.page() }

func (col uint64ColumnBuffer) Pages() Pages { return onePage(col.page()) }

func (col uint64ColumnBuffer) Clone() ColumnBuffer {
	return uint64ColumnBuffer{col.int64ColumnBuffer.Clone().(*int64ColumnBuffer)}
}

func (col uint64ColumnBuffer) Less(i, j int) bool {
	return uint64(col.values[i]) < uint64(col.values[j])
}

var (
	_ io.Writer = (*booleanColumnBuffer)(nil)
	_ io.Writer = (*int32ColumnBuffer)(nil)
	_ io.Writer = (*int64ColumnBuffer)(nil)
	_ io.Writer = (*int96ColumnBuffer)(nil)
	_ io.Writer = (*floatColumnBuffer)(nil)
	_ io.Writer = (*doubleColumnBuffer)(nil)

	_ BooleanWriter           = (*booleanColumnBuffer)(nil)
	_ Int32Writer             = (*int32ColumnBuffer)(nil)
	_ Int64Writer             = (*int64ColumnBuffer)(nil)
	_ Int96Writer             = (*int96ColumnBuffer)(nil)
	_ FloatWriter             = (*floatColumnBuffer)(nil)
	_ DoubleWriter            = (*doubleColumnBuffer)(nil)
	_ ByteArrayWriter         = (*byteArrayColumnBuffer)(nil)
	_ FixedLenByteArrayWriter = (*fixedLenByteArrayColumnBuffer)(nil)
)
