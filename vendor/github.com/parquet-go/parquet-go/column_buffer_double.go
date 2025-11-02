package parquet

import (
	"fmt"
	"io"
	"slices"

	"github.com/parquet-go/bitpack/unsafecast"
	"github.com/parquet-go/parquet-go/sparse"
)

type doubleColumnBuffer struct{ doublePage }

func newDoubleColumnBuffer(typ Type, columnIndex int16, numValues int32) *doubleColumnBuffer {
	return &doubleColumnBuffer{
		doublePage: doublePage{
			typ:         typ,
			values:      make([]float64, 0, numValues),
			columnIndex: ^columnIndex,
		},
	}
}

func (col *doubleColumnBuffer) Clone() ColumnBuffer {
	return &doubleColumnBuffer{
		doublePage: doublePage{
			typ:         col.typ,
			values:      slices.Clone(col.values),
			columnIndex: col.columnIndex,
		},
	}
}

func (col *doubleColumnBuffer) ColumnIndex() (ColumnIndex, error) {
	return doubleColumnIndex{&col.doublePage}, nil
}

func (col *doubleColumnBuffer) OffsetIndex() (OffsetIndex, error) {
	return doubleOffsetIndex{&col.doublePage}, nil
}

func (col *doubleColumnBuffer) BloomFilter() BloomFilter { return nil }

func (col *doubleColumnBuffer) Dictionary() Dictionary { return nil }

func (col *doubleColumnBuffer) Pages() Pages { return onePage(col.Page()) }

func (col *doubleColumnBuffer) Page() Page { return &col.doublePage }

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
	col.values = append(col.values, unsafecast.Slice[float64](b)...)
	return len(b), nil
}

func (col *doubleColumnBuffer) WriteDoubles(values []float64) (int, error) {
	col.values = append(col.values, values...)
	return len(values), nil
}

func (col *doubleColumnBuffer) WriteValues(values []Value) (int, error) {
	col.writeValues(makeArrayValue(values, offsetOfU64), columnLevels{})
	return len(values), nil
}

func (col *doubleColumnBuffer) writeValues(rows sparse.Array, _ columnLevels) {
	if n := len(col.values) + rows.Len(); n > cap(col.values) {
		col.values = append(make([]float64, 0, max(n, 2*cap(col.values))), col.values...)
	}
	n := len(col.values)
	col.values = col.values[:n+rows.Len()]
	sparse.GatherFloat64(col.values[n:], rows.Float64Array())
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
