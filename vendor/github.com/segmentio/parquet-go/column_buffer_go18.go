//go:build go1.18

package parquet

import (
	"fmt"
	"io"

	"github.com/segmentio/parquet-go/internal/cast"
)

type columnBuffer[T primitive] struct {
	page[T]
	typ Type
}

func newUint32ColumnBuffer(typ Type, columnIndex int16, bufferSize int) *columnBuffer[uint32] {
	return newColumnBuffer(typ, columnIndex, bufferSize, &uint32Class)
}

func newUint64ColumnBuffer(typ Type, columnIndex int16, bufferSize int) *columnBuffer[uint64] {
	return newColumnBuffer(typ, columnIndex, bufferSize, &uint64Class)
}

func newColumnBuffer[T primitive](typ Type, columnIndex int16, bufferSize int, class *class[T]) *columnBuffer[T] {
	return &columnBuffer[T]{
		page: page[T]{
			class:       class,
			values:      make([]T, 0, bufferSize/sizeof[T]()),
			columnIndex: ^columnIndex,
		},
		typ: typ,
	}
}

func (col *columnBuffer[T]) Clone() ColumnBuffer {
	return &columnBuffer[T]{
		page: page[T]{
			class:       col.page.class,
			values:      append([]T{}, col.values...),
			columnIndex: col.columnIndex,
		},
		typ: col.typ,
	}
}

func (col *columnBuffer[T]) Type() Type { return col.typ }

func (col *columnBuffer[T]) ColumnIndex() ColumnIndex { return columnIndex[T]{&col.page} }

func (col *columnBuffer[T]) OffsetIndex() OffsetIndex { return offsetIndex[T]{&col.page} }

func (col *columnBuffer[T]) BloomFilter() BloomFilter { return nil }

func (col *columnBuffer[T]) Dictionary() Dictionary { return nil }

func (col *columnBuffer[T]) Pages() Pages { return onePage(col.Page()) }

func (col *columnBuffer[T]) Page() BufferedPage { return &col.page }

func (col *columnBuffer[T]) Reset() { col.values = col.values[:0] }

func (col *columnBuffer[T]) Cap() int { return cap(col.values) }

func (col *columnBuffer[T]) Len() int { return len(col.values) }

func (col *columnBuffer[T]) Less(i, j int) bool {
	return col.class.less(col.values[i], col.values[j])
}

func (col *columnBuffer[T]) Swap(i, j int) {
	col.values[i], col.values[j] = col.values[j], col.values[i]
}

func (col *columnBuffer[T]) Write(b []byte) (int, error) {
	if (len(b) % sizeof[T]()) != 0 {
		return 0, fmt.Errorf("cannot write %s values from input of size %d", col.class.name, len(b))
	}
	n, err := col.WriteRequired(cast.BytesToSlice[T](b))
	return sizeof[T]() * n, err
}

func (col *columnBuffer[T]) WriteRequired(values []T) (int, error) {
	col.values = append(col.values, values...)
	return len(values), nil
}

func (col *columnBuffer[T]) WriteValues(values []Value) (int, error) {
	value := col.class.value
	for _, v := range values {
		col.values = append(col.values, value(v))
	}
	return len(values), nil
}

func (col *columnBuffer[T]) ReadValuesAt(values []Value, offset int64) (n int, err error) {
	i := int(offset)
	switch {
	case i < 0:
		return 0, errRowIndexOutOfBounds(offset, int64(len(col.values)))
	case i >= len(col.values):
		return 0, io.EOF
	default:
		for n < len(values) && i < len(col.values) {
			values[n] = col.class.makeValue(col.values[i])
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
