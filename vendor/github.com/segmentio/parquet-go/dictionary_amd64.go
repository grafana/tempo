//go:build !purego

package parquet

import (
	"github.com/segmentio/parquet-go/internal/unsafecast"
	"github.com/segmentio/parquet-go/sparse"
)

//go:noescape
func dictionaryBoundsInt32(dict []int32, indexes []int32) (min, max int32, err errno)

//go:noescape
func dictionaryBoundsInt64(dict []int64, indexes []int32) (min, max int64, err errno)

//go:noescape
func dictionaryBoundsFloat32(dict []float32, indexes []int32) (min, max float32, err errno)

//go:noescape
func dictionaryBoundsFloat64(dict []float64, indexes []int32) (min, max float64, err errno)

//go:noescape
func dictionaryBoundsUint32(dict []uint32, indexes []int32) (min, max uint32, err errno)

//go:noescape
func dictionaryBoundsUint64(dict []uint64, indexes []int32) (min, max uint64, err errno)

//go:noescape
func dictionaryBoundsBE128(dict [][16]byte, indexes []int32) (min, max *[16]byte, err errno)

//go:noescape
func dictionaryLookup32(dict []uint32, indexes []int32, rows sparse.Array) errno

//go:noescape
func dictionaryLookup64(dict []uint64, indexes []int32, rows sparse.Array) errno

//go:noescape
func dictionaryLookupByteArrayString(dict []uint32, page []byte, indexes []int32, rows sparse.Array) errno

//go:noescape
func dictionaryLookupFixedLenByteArrayString(dict []byte, len int, indexes []int32, rows sparse.Array) errno

//go:noescape
func dictionaryLookupFixedLenByteArrayPointer(dict []byte, len int, indexes []int32, rows sparse.Array) errno

func (d *int32Dictionary) lookup(indexes []int32, rows sparse.Array) {
	checkLookupIndexBounds(indexes, rows)
	dict := unsafecast.Int32ToUint32(d.values)
	dictionaryLookup32(dict, indexes, rows).check()
}

func (d *int64Dictionary) lookup(indexes []int32, rows sparse.Array) {
	checkLookupIndexBounds(indexes, rows)
	dict := unsafecast.Int64ToUint64(d.values)
	dictionaryLookup64(dict, indexes, rows).check()
}

func (d *floatDictionary) lookup(indexes []int32, rows sparse.Array) {
	checkLookupIndexBounds(indexes, rows)
	dict := unsafecast.Float32ToUint32(d.values)
	dictionaryLookup32(dict, indexes, rows).check()
}

func (d *doubleDictionary) lookup(indexes []int32, rows sparse.Array) {
	checkLookupIndexBounds(indexes, rows)
	dict := unsafecast.Float64ToUint64(d.values)
	dictionaryLookup64(dict, indexes, rows).check()
}

func (d *byteArrayDictionary) lookupString(indexes []int32, rows sparse.Array) {
	checkLookupIndexBounds(indexes, rows)
	dictionaryLookupByteArrayString(d.offsets, d.values, indexes, rows).check()
}

func (d *fixedLenByteArrayDictionary) lookupString(indexes []int32, rows sparse.Array) {
	checkLookupIndexBounds(indexes, rows)
	dictionaryLookupFixedLenByteArrayString(d.data, d.size, indexes, rows).check()
}

func (d *uint32Dictionary) lookup(indexes []int32, rows sparse.Array) {
	checkLookupIndexBounds(indexes, rows)
	dictionaryLookup32(d.values, indexes, rows).check()
}

func (d *uint64Dictionary) lookup(indexes []int32, rows sparse.Array) {
	checkLookupIndexBounds(indexes, rows)
	dictionaryLookup64(d.values, indexes, rows).check()
}

func (d *be128Dictionary) lookupString(indexes []int32, rows sparse.Array) {
	checkLookupIndexBounds(indexes, rows)
	dict := unsafecast.Uint128ToBytes(d.values)
	dictionaryLookupFixedLenByteArrayString(dict, 16, indexes, rows).check()
}

func (d *be128Dictionary) lookupPointer(indexes []int32, rows sparse.Array) {
	checkLookupIndexBounds(indexes, rows)
	dict := unsafecast.Uint128ToBytes(d.values)
	dictionaryLookupFixedLenByteArrayPointer(dict, 16, indexes, rows).check()
}

func (d *int32Dictionary) bounds(indexes []int32) (min, max int32) {
	min, max, err := dictionaryBoundsInt32(d.values, indexes)
	err.check()
	return min, max
}

func (d *int64Dictionary) bounds(indexes []int32) (min, max int64) {
	min, max, err := dictionaryBoundsInt64(d.values, indexes)
	err.check()
	return min, max
}

func (d *floatDictionary) bounds(indexes []int32) (min, max float32) {
	min, max, err := dictionaryBoundsFloat32(d.values, indexes)
	err.check()
	return min, max
}

func (d *doubleDictionary) bounds(indexes []int32) (min, max float64) {
	min, max, err := dictionaryBoundsFloat64(d.values, indexes)
	err.check()
	return min, max
}

func (d *uint32Dictionary) bounds(indexes []int32) (min, max uint32) {
	min, max, err := dictionaryBoundsUint32(d.values, indexes)
	err.check()
	return min, max
}

func (d *uint64Dictionary) bounds(indexes []int32) (min, max uint64) {
	min, max, err := dictionaryBoundsUint64(d.values, indexes)
	err.check()
	return min, max
}

func (d *be128Dictionary) bounds(indexes []int32) (min, max *[16]byte) {
	min, max, err := dictionaryBoundsBE128(d.values, indexes)
	err.check()
	return min, max
}
