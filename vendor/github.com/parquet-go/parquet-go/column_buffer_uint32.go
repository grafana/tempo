package parquet

import (
	"fmt"
	"io"
	"slices"

	"github.com/parquet-go/bitpack/unsafecast"
	"github.com/parquet-go/parquet-go/sparse"
)

type uint32ColumnBuffer struct{ uint32Page }

func newUint32ColumnBuffer(typ Type, columnIndex int16, numValues int32) *uint32ColumnBuffer {
	return &uint32ColumnBuffer{
		uint32Page: uint32Page{
			typ:         typ,
			values:      make([]uint32, 0, numValues),
			columnIndex: ^columnIndex,
		},
	}
}

func (col *uint32ColumnBuffer) Clone() ColumnBuffer {
	return &uint32ColumnBuffer{
		uint32Page: uint32Page{
			typ:         col.typ,
			values:      slices.Clone(col.values),
			columnIndex: col.columnIndex,
		},
	}
}

func (col *uint32ColumnBuffer) ColumnIndex() (ColumnIndex, error) {
	return uint32ColumnIndex{&col.uint32Page}, nil
}

func (col *uint32ColumnBuffer) OffsetIndex() (OffsetIndex, error) {
	return uint32OffsetIndex{&col.uint32Page}, nil
}

func (col *uint32ColumnBuffer) BloomFilter() BloomFilter { return nil }

func (col *uint32ColumnBuffer) Dictionary() Dictionary { return nil }

func (col *uint32ColumnBuffer) Pages() Pages { return onePage(col.Page()) }

func (col *uint32ColumnBuffer) Page() Page { return &col.uint32Page }

func (col *uint32ColumnBuffer) Reset() { col.values = col.values[:0] }

func (col *uint32ColumnBuffer) Cap() int { return cap(col.values) }

func (col *uint32ColumnBuffer) Len() int { return len(col.values) }

func (col *uint32ColumnBuffer) Less(i, j int) bool { return col.values[i] < col.values[j] }

func (col *uint32ColumnBuffer) Swap(i, j int) {
	col.values[i], col.values[j] = col.values[j], col.values[i]
}

func (col *uint32ColumnBuffer) Write(b []byte) (int, error) {
	if (len(b) % 4) != 0 {
		return 0, fmt.Errorf("cannot write INT32 values from input of size %d", len(b))
	}
	col.values = append(col.values, unsafecast.Slice[uint32](b)...)
	return len(b), nil
}

func (col *uint32ColumnBuffer) WriteUint32s(values []uint32) (int, error) {
	col.values = append(col.values, values...)
	return len(values), nil
}

func (col *uint32ColumnBuffer) WriteValues(values []Value) (int, error) {
	col.writeValues(makeArrayValue(values, offsetOfU32), columnLevels{})
	return len(values), nil
}

func (col *uint32ColumnBuffer) writeValues(rows sparse.Array, _ columnLevels) {
	if n := len(col.values) + rows.Len(); n > cap(col.values) {
		col.values = append(make([]uint32, 0, max(n, 2*cap(col.values))), col.values...)
	}
	n := len(col.values)
	col.values = col.values[:n+rows.Len()]
	sparse.GatherUint32(col.values[n:], rows.Uint32Array())
}

func (col *uint32ColumnBuffer) ReadValuesAt(values []Value, offset int64) (n int, err error) {
	i := int(offset)
	switch {
	case i < 0:
		return 0, errRowIndexOutOfBounds(offset, int64(len(col.values)))
	case i >= len(col.values):
		return 0, io.EOF
	default:
		for n < len(values) && i < len(col.values) {
			values[n] = col.makeValue(col.values[i])
			n++
			i++
		}
		if n < len(values) {
			err = io.EOF
		}
		return n, err
	}
}
