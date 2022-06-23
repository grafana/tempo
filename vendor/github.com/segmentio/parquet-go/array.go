package parquet

import (
	"unsafe"
)

type array struct {
	ptr unsafe.Pointer
	len int
}

func makeArray(ptr unsafe.Pointer, len int) array {
	return array{ptr: ptr, len: len}
}

func makeArrayBool(values []bool) array {
	return *(*array)(unsafe.Pointer(&values))
}

func makeArrayString(values []string) array {
	return *(*array)(unsafe.Pointer(&values))
}

func makeArrayValue(values []Value) array {
	return *(*array)(unsafe.Pointer(&values))
}

func makeArrayBE128(values []*[16]byte) array {
	return *(*array)(unsafe.Pointer(&values))
}

func (a array) index(i int, size, offset uintptr) unsafe.Pointer {
	return unsafe.Add(a.ptr, uintptr(i)*size+offset)
}

func (a array) slice(i, j int, size, offset uintptr) array {
	if i < 0 || i > a.len || j < 0 || j > a.len {
		panic("slice index out of bounds")
	}
	if i > j {
		panic("negative slice length")
	}
	return array{
		ptr: a.index(i, size, offset),
		len: j - i,
	}
}
