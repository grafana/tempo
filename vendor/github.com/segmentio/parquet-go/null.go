//go:build go1.18

package parquet

import (
	"reflect"
	"unsafe"

	"github.com/segmentio/parquet-go/deprecated"
	"github.com/segmentio/parquet-go/internal/bytealg"
	"github.com/segmentio/parquet-go/internal/unsafecast"
)

// nullIndexFunc is the type of functions used to detect null values in rows.
//
// For each value of the rows array, the bitmap passed as first argument is
// populated to indicate whether the values were null (0) or not (1).
//
// The function writes one bit to the output buffer for each row in the input,
// the buffer must be sized accordingly.
type nullIndexFunc func(bits []uint64, rows array, size, offset uintptr)

func nullIndex[T comparable](bits []uint64, rows array, size, offset uintptr) {
	var zero T
	for i := 0; i < rows.len; i++ {
		v := *(*T)(rows.index(i, size, offset))
		if v != zero {
			x := uint(i) / 64
			y := uint(i) % 64
			bits[x] |= 1 << y
		}
	}
}

func nullIndexStruct(bits []uint64, rows array, size, offset uintptr) {
	bytealg.Broadcast(unsafecast.Slice[byte](bits), 0xFF)
}

func nullIndexFuncOf(t reflect.Type) nullIndexFunc {
	switch t {
	case reflect.TypeOf(deprecated.Int96{}):
		return nullIndex[deprecated.Int96]
	}

	switch t.Kind() {
	case reflect.Bool:
		return nullIndexBool

	case reflect.Int:
		return nullIndexInt

	case reflect.Int32:
		return nullIndexInt32

	case reflect.Int64:
		return nullIndexInt64

	case reflect.Uint:
		return nullIndexUint

	case reflect.Uint32:
		return nullIndexUint32

	case reflect.Uint64:
		return nullIndexUint64

	case reflect.Float32:
		return nullIndexFloat32

	case reflect.Float64:
		return nullIndexFloat64

	case reflect.String:
		return nullIndexString

	case reflect.Slice:
		return nullIndexSlice

	case reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			switch size := t.Len(); size {
			case 16:
				return nullIndexUint128
			default:
				return nullIndexFuncOfByteArray(size)
			}
		}

	case reflect.Pointer:
		return nullIndexPointer

	case reflect.Struct:
		return nullIndexStruct
	}

	panic("cannot convert Go values of type " + t.String() + " to parquet value")
}

func nullIndexFuncOfByteArray(n int) nullIndexFunc {
	return func(bits []uint64, rows array, size, offset uintptr) {
		for i := 0; i < rows.len; i++ {
			p := (*byte)(rows.index(i, size, offset))
			b := unsafe.Slice(p, n)
			if !isZero(b) {
				x := uint(i) / 64
				y := uint(i) % 64
				bits[x] |= 1 << y
			}
		}
	}
}

func isZero(b []byte) bool {
	for _, c := range b {
		if c != 0 {
			return false
		}
	}
	return true
}
