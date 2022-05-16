//go:build go1.18

package parquet

import (
	"github.com/segmentio/parquet-go/format"
	"github.com/segmentio/parquet-go/internal/cast"
)

type columnIndex[T primitive] struct{ page *page[T] }

func (i columnIndex[T]) NumPages() int       { return 1 }
func (i columnIndex[T]) NullCount(int) int64 { return 0 }
func (i columnIndex[T]) NullPage(int) bool   { return false }
func (i columnIndex[T]) MinValue(int) Value  { return i.page.class.makeValue(i.page.min()) }
func (i columnIndex[T]) MaxValue(int) Value  { return i.page.class.makeValue(i.page.max()) }
func (i columnIndex[T]) IsAscending() bool   { return false }
func (i columnIndex[T]) IsDescending() bool  { return false }

type columnIndexer[T primitive] struct {
	class      *class[T]
	nullPages  []bool
	nullCounts []int64
	minValues  []T
	maxValues  []T
}

func newColumnIndexer[T primitive](class *class[T]) *columnIndexer[T] {
	return &columnIndexer[T]{class: class}
}

func (i *columnIndexer[T]) Reset() {
	i.nullPages = i.nullPages[:0]
	i.nullCounts = i.nullCounts[:0]
	i.minValues = i.minValues[:0]
	i.maxValues = i.maxValues[:0]
}

func (i *columnIndexer[T]) IndexPage(numValues, numNulls int64, min, max Value) {
	i.nullPages = append(i.nullPages, numValues == numNulls)
	i.nullCounts = append(i.nullCounts, numNulls)
	i.minValues = append(i.minValues, i.class.value(min))
	i.maxValues = append(i.maxValues, i.class.value(max))
}

func (i *columnIndexer[T]) ColumnIndex() format.ColumnIndex {
	minValues := splitFixedLenByteArrayList(sizeof[T](), cast.SliceToBytes(i.minValues))
	maxValues := splitFixedLenByteArrayList(sizeof[T](), cast.SliceToBytes(i.maxValues))
	minOrder := i.class.order(i.minValues)
	maxOrder := i.class.order(i.maxValues)
	return format.ColumnIndex{
		NullPages:     i.nullPages,
		NullCounts:    i.nullCounts,
		MinValues:     minValues,
		MaxValues:     maxValues,
		BoundaryOrder: boundaryOrderOf(minOrder, maxOrder),
	}
}
