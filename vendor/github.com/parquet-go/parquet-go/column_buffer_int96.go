package parquet

import (
	"fmt"
	"io"
	"slices"

	"github.com/parquet-go/bitpack/unsafecast"
	"github.com/parquet-go/parquet-go/deprecated"
	"github.com/parquet-go/parquet-go/sparse"
)

type int96ColumnBuffer struct{ int96Page }

func newInt96ColumnBuffer(typ Type, columnIndex int16, numValues int32) *int96ColumnBuffer {
	return &int96ColumnBuffer{
		int96Page: int96Page{
			typ:         typ,
			values:      make([]deprecated.Int96, 0, numValues),
			columnIndex: ^columnIndex,
		},
	}
}

func (col *int96ColumnBuffer) Clone() ColumnBuffer {
	return &int96ColumnBuffer{
		int96Page: int96Page{
			typ:         col.typ,
			values:      slices.Clone(col.values),
			columnIndex: col.columnIndex,
		},
	}
}

func (col *int96ColumnBuffer) ColumnIndex() (ColumnIndex, error) {
	return int96ColumnIndex{&col.int96Page}, nil
}

func (col *int96ColumnBuffer) OffsetIndex() (OffsetIndex, error) {
	return int96OffsetIndex{&col.int96Page}, nil
}

func (col *int96ColumnBuffer) BloomFilter() BloomFilter { return nil }

func (col *int96ColumnBuffer) Dictionary() Dictionary { return nil }

func (col *int96ColumnBuffer) Pages() Pages { return onePage(col.Page()) }

func (col *int96ColumnBuffer) Page() Page { return &col.int96Page }

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
	col.values = append(col.values, unsafecast.Slice[deprecated.Int96](b)...)
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

func (col *int96ColumnBuffer) writeValues(rows sparse.Array, _ columnLevels) {
	for i := range rows.Len() {
		p := rows.Index(i)
		col.values = append(col.values, *(*deprecated.Int96)(p))
	}
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
