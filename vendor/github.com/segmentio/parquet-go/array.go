package parquet

import (
	"unsafe"

	"github.com/segmentio/parquet-go/sparse"
)

func makeArrayValue(values []Value, offset uintptr) sparse.Array {
	ptr := *(*unsafe.Pointer)(unsafe.Pointer(&values))
	return sparse.UnsafeArray(unsafe.Add(ptr, offset), len(values), unsafe.Sizeof(Value{}))
}

func makeArrayString(values []string) sparse.Array {
	str := ""
	ptr := *(*unsafe.Pointer)(unsafe.Pointer(&values))
	return sparse.UnsafeArray(ptr, len(values), unsafe.Sizeof(str))
}

func makeArrayBE128(values []*[16]byte) sparse.Array {
	ptr := *(*unsafe.Pointer)(unsafe.Pointer(&values))
	return sparse.UnsafeArray(ptr, len(values), unsafe.Sizeof((*[16]byte)(nil)))
}
