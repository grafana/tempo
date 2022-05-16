//go:build !go1.18

package parquet

import (
	"github.com/segmentio/parquet-go/deprecated"
	"github.com/segmentio/parquet-go/format"
	"github.com/segmentio/parquet-go/internal/bits"
)

type booleanColumnIndex struct{ page *booleanPage }

func (i booleanColumnIndex) NumPages() int       { return 1 }
func (i booleanColumnIndex) NullCount(int) int64 { return 0 }
func (i booleanColumnIndex) NullPage(int) bool   { return false }
func (i booleanColumnIndex) MinValue(int) Value  { return makeValueBoolean(i.page.min()) }
func (i booleanColumnIndex) MaxValue(int) Value  { return makeValueBoolean(i.page.max()) }
func (i booleanColumnIndex) IsAscending() bool   { return false }
func (i booleanColumnIndex) IsDescending() bool  { return false }

type int32ColumnIndex struct{ page *int32Page }

func (i int32ColumnIndex) NumPages() int       { return 1 }
func (i int32ColumnIndex) NullCount(int) int64 { return 0 }
func (i int32ColumnIndex) NullPage(int) bool   { return false }
func (i int32ColumnIndex) MinValue(int) Value  { return makeValueInt32(i.page.min()) }
func (i int32ColumnIndex) MaxValue(int) Value  { return makeValueInt32(i.page.max()) }
func (i int32ColumnIndex) IsAscending() bool   { return false }
func (i int32ColumnIndex) IsDescending() bool  { return false }

type int64ColumnIndex struct{ page *int64Page }

func (i int64ColumnIndex) NumPages() int       { return 1 }
func (i int64ColumnIndex) NullCount(int) int64 { return 0 }
func (i int64ColumnIndex) NullPage(int) bool   { return false }
func (i int64ColumnIndex) MinValue(int) Value  { return makeValueInt64(i.page.min()) }
func (i int64ColumnIndex) MaxValue(int) Value  { return makeValueInt64(i.page.max()) }
func (i int64ColumnIndex) IsAscending() bool   { return false }
func (i int64ColumnIndex) IsDescending() bool  { return false }

type int96ColumnIndex struct{ page *int96Page }

func (i int96ColumnIndex) NumPages() int       { return 1 }
func (i int96ColumnIndex) NullCount(int) int64 { return 0 }
func (i int96ColumnIndex) NullPage(int) bool   { return false }
func (i int96ColumnIndex) MinValue(int) Value  { return makeValueInt96(i.page.min()) }
func (i int96ColumnIndex) MaxValue(int) Value  { return makeValueInt96(i.page.max()) }
func (i int96ColumnIndex) IsAscending() bool   { return false }
func (i int96ColumnIndex) IsDescending() bool  { return false }

type floatColumnIndex struct{ page *floatPage }

func (i floatColumnIndex) NumPages() int       { return 1 }
func (i floatColumnIndex) NullCount(int) int64 { return 0 }
func (i floatColumnIndex) NullPage(int) bool   { return false }
func (i floatColumnIndex) MinValue(int) Value  { return makeValueFloat(i.page.min()) }
func (i floatColumnIndex) MaxValue(int) Value  { return makeValueFloat(i.page.max()) }
func (i floatColumnIndex) IsAscending() bool   { return false }
func (i floatColumnIndex) IsDescending() bool  { return false }

type doubleColumnIndex struct{ page *doublePage }

func (i doubleColumnIndex) NumPages() int       { return 1 }
func (i doubleColumnIndex) NullCount(int) int64 { return 0 }
func (i doubleColumnIndex) NullPage(int) bool   { return false }
func (i doubleColumnIndex) MinValue(int) Value  { return makeValueDouble(i.page.min()) }
func (i doubleColumnIndex) MaxValue(int) Value  { return makeValueDouble(i.page.max()) }
func (i doubleColumnIndex) IsAscending() bool   { return false }
func (i doubleColumnIndex) IsDescending() bool  { return false }

type uint32ColumnIndex struct{ page uint32Page }

func (i uint32ColumnIndex) NumPages() int       { return 1 }
func (i uint32ColumnIndex) NullCount(int) int64 { return 0 }
func (i uint32ColumnIndex) NullPage(int) bool   { return false }
func (i uint32ColumnIndex) MinValue(int) Value  { return makeValueInt32(int32(i.page.min())) }
func (i uint32ColumnIndex) MaxValue(int) Value  { return makeValueInt32(int32(i.page.max())) }
func (i uint32ColumnIndex) IsAscending() bool   { return false }
func (i uint32ColumnIndex) IsDescending() bool  { return false }

type uint64ColumnIndex struct{ page uint64Page }

func (i uint64ColumnIndex) NumPages() int       { return 1 }
func (i uint64ColumnIndex) NullCount(int) int64 { return 0 }
func (i uint64ColumnIndex) NullPage(int) bool   { return false }
func (i uint64ColumnIndex) MinValue(int) Value  { return makeValueInt64(int64(i.page.min())) }
func (i uint64ColumnIndex) MaxValue(int) Value  { return makeValueInt64(int64(i.page.max())) }
func (i uint64ColumnIndex) IsAscending() bool   { return false }
func (i uint64ColumnIndex) IsDescending() bool  { return false }

type booleanColumnIndexer struct {
	baseColumnIndexer
	minValues []bool
	maxValues []bool
}

func newBooleanColumnIndexer() *booleanColumnIndexer {
	return new(booleanColumnIndexer)
}

func (i *booleanColumnIndexer) Reset() {
	i.reset()
	i.minValues = i.minValues[:0]
	i.maxValues = i.maxValues[:0]
}

func (i *booleanColumnIndexer) IndexPage(numValues, numNulls int64, min, max Value) {
	i.observe(numValues, numNulls)
	i.minValues = append(i.minValues, min.Boolean())
	i.maxValues = append(i.maxValues, max.Boolean())
}

func (i *booleanColumnIndexer) ColumnIndex() format.ColumnIndex {
	return i.columnIndex(
		splitFixedLenByteArrayList(1, bits.BoolToBytes(i.minValues)),
		splitFixedLenByteArrayList(1, bits.BoolToBytes(i.maxValues)),
		bits.OrderOfBool(i.minValues),
		bits.OrderOfBool(i.maxValues),
	)
}

type int32ColumnIndexer struct {
	baseColumnIndexer
	minValues []int32
	maxValues []int32
}

func newInt32ColumnIndexer() *int32ColumnIndexer {
	return new(int32ColumnIndexer)
}

func (i *int32ColumnIndexer) Reset() {
	i.reset()
	i.minValues = i.minValues[:0]
	i.maxValues = i.maxValues[:0]
}

func (i *int32ColumnIndexer) IndexPage(numValues, numNulls int64, min, max Value) {
	i.observe(numValues, numNulls)
	i.minValues = append(i.minValues, min.Int32())
	i.maxValues = append(i.maxValues, max.Int32())
}

func (i *int32ColumnIndexer) ColumnIndex() format.ColumnIndex {
	return i.columnIndex(
		splitFixedLenByteArrayList(4, bits.Int32ToBytes(i.minValues)),
		splitFixedLenByteArrayList(4, bits.Int32ToBytes(i.maxValues)),
		bits.OrderOfInt32(i.minValues),
		bits.OrderOfInt32(i.maxValues),
	)
}

type int64ColumnIndexer struct {
	baseColumnIndexer
	minValues []int64
	maxValues []int64
}

func newInt64ColumnIndexer() *int64ColumnIndexer {
	return new(int64ColumnIndexer)
}

func (i *int64ColumnIndexer) Reset() {
	i.reset()
	i.minValues = i.minValues[:0]
	i.maxValues = i.maxValues[:0]
}

func (i *int64ColumnIndexer) IndexPage(numValues, numNulls int64, min, max Value) {
	i.observe(numValues, numNulls)
	i.minValues = append(i.minValues, min.Int64())
	i.maxValues = append(i.maxValues, max.Int64())
}

func (i *int64ColumnIndexer) ColumnIndex() format.ColumnIndex {
	return i.columnIndex(
		splitFixedLenByteArrayList(8, bits.Int64ToBytes(i.minValues)),
		splitFixedLenByteArrayList(8, bits.Int64ToBytes(i.maxValues)),
		bits.OrderOfInt64(i.minValues),
		bits.OrderOfInt64(i.maxValues),
	)
}

type int96ColumnIndexer struct {
	baseColumnIndexer
	minValues []deprecated.Int96
	maxValues []deprecated.Int96
}

func newInt96ColumnIndexer() *int96ColumnIndexer {
	return new(int96ColumnIndexer)
}

func (i *int96ColumnIndexer) Reset() {
	i.reset()
	i.minValues = i.minValues[:0]
	i.maxValues = i.maxValues[:0]
}

func (i *int96ColumnIndexer) IndexPage(numValues, numNulls int64, min, max Value) {
	i.observe(numValues, numNulls)
	i.minValues = append(i.minValues, min.Int96())
	i.maxValues = append(i.maxValues, max.Int96())
}

func (i *int96ColumnIndexer) ColumnIndex() format.ColumnIndex {
	return i.columnIndex(
		splitFixedLenByteArrayList(12, deprecated.Int96ToBytes(i.minValues)),
		splitFixedLenByteArrayList(12, deprecated.Int96ToBytes(i.maxValues)),
		deprecated.OrderOfInt96(i.minValues),
		deprecated.OrderOfInt96(i.maxValues),
	)
}

type floatColumnIndexer struct {
	baseColumnIndexer
	minValues []float32
	maxValues []float32
}

func newFloatColumnIndexer() *floatColumnIndexer {
	return new(floatColumnIndexer)
}

func (i *floatColumnIndexer) Reset() {
	i.reset()
	i.minValues = i.minValues[:0]
	i.maxValues = i.maxValues[:0]
}

func (i *floatColumnIndexer) IndexPage(numValues, numNulls int64, min, max Value) {
	i.observe(numValues, numNulls)
	i.minValues = append(i.minValues, min.Float())
	i.maxValues = append(i.maxValues, max.Float())
}

func (i *floatColumnIndexer) ColumnIndex() format.ColumnIndex {
	return i.columnIndex(
		splitFixedLenByteArrayList(4, bits.Float32ToBytes(i.minValues)),
		splitFixedLenByteArrayList(4, bits.Float32ToBytes(i.maxValues)),
		bits.OrderOfFloat32(i.minValues),
		bits.OrderOfFloat32(i.maxValues),
	)
}

type doubleColumnIndexer struct {
	baseColumnIndexer
	minValues []float64
	maxValues []float64
}

func newDoubleColumnIndexer() *doubleColumnIndexer {
	return new(doubleColumnIndexer)
}

func (i *doubleColumnIndexer) Reset() {
	i.reset()
	i.minValues = i.minValues[:0]
	i.maxValues = i.maxValues[:0]
}

func (i *doubleColumnIndexer) IndexPage(numValues, numNulls int64, min, max Value) {
	i.observe(numValues, numNulls)
	i.minValues = append(i.minValues, min.Double())
	i.maxValues = append(i.maxValues, max.Double())
}

func (i *doubleColumnIndexer) ColumnIndex() format.ColumnIndex {
	return i.columnIndex(
		splitFixedLenByteArrayList(8, bits.Float64ToBytes(i.minValues)),
		splitFixedLenByteArrayList(8, bits.Float64ToBytes(i.maxValues)),
		bits.OrderOfFloat64(i.minValues),
		bits.OrderOfFloat64(i.maxValues),
	)
}

type uint32ColumnIndexer struct{ *int32ColumnIndexer }

func newUint32ColumnIndexer() uint32ColumnIndexer {
	return uint32ColumnIndexer{newInt32ColumnIndexer()}
}

func (i uint32ColumnIndexer) ColumnIndex() format.ColumnIndex {
	minValues := bits.Int32ToUint32(i.minValues)
	maxValues := bits.Int32ToUint32(i.maxValues)
	return i.columnIndex(
		splitFixedLenByteArrayList(4, bits.Uint32ToBytes(minValues)),
		splitFixedLenByteArrayList(4, bits.Uint32ToBytes(maxValues)),
		bits.OrderOfUint32(minValues),
		bits.OrderOfUint32(maxValues),
	)
}

type uint64ColumnIndexer struct{ *int64ColumnIndexer }

func newUint64ColumnIndexer() uint64ColumnIndexer {
	return uint64ColumnIndexer{newInt64ColumnIndexer()}
}

func (i uint64ColumnIndexer) ColumnIndex() format.ColumnIndex {
	minValues := bits.Int64ToUint64(i.minValues)
	maxValues := bits.Int64ToUint64(i.maxValues)
	return i.columnIndex(
		splitFixedLenByteArrayList(8, bits.Uint64ToBytes(minValues)),
		splitFixedLenByteArrayList(8, bits.Uint64ToBytes(maxValues)),
		bits.OrderOfUint64(minValues),
		bits.OrderOfUint64(maxValues),
	)
}
