//go:build go1.18

package parquet

import (
	"unsafe"

	"github.com/segmentio/parquet-go/internal/unsafecast"
	"github.com/segmentio/parquet-go/sparse"
)

func makeArray(base unsafe.Pointer, length int, offset uintptr) sparse.Array {
	return sparse.UnsafeArray(base, length, offset)
}

func makeArrayOf[T any](s []T) sparse.Array {
	var model T
	return makeArray(unsafecast.PointerOf(s), len(s), unsafe.Sizeof(model))
}

func makeSlice[T any](a sparse.Array) []T {
	return slice[T](a.Index(0), a.Len())
}

func slice[T any](p unsafe.Pointer, n int) []T {
	return unsafe.Slice((*T)(p), n)
}

type sliceHeader struct {
	base unsafe.Pointer
	len  int
	cap  int
}
