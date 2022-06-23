//go:build go1.18

package parquet

import (
	"unsafe"

	"github.com/segmentio/parquet-go/internal/unsafecast"
)

func makeArrayOf[T any](s []T) array {
	return makeArray(unsafecast.PointerOf(s), len(s))
}

func makeSlice[T any](a array) []T {
	return slice[T](a.ptr, a.len)
}

func slice[T any](p unsafe.Pointer, n int) []T {
	return unsafe.Slice((*T)(p), n)
}
