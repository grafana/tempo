//go:build go1.18

package cast

import "unsafe"

type Int96 = [3]uint32

type Uint128 = [16]byte

func Slice[To, From any](data []From) []To {
	var zf From
	var zt To
	return unsafe.Slice(*(**To)(unsafe.Pointer(&data)), (uintptr(len(data))*unsafe.Sizeof(zf))/unsafe.Sizeof(zt))
}

func SliceToBytes[T any](data []T) []byte {
	return Slice[byte](data)
}

func BytesToSlice[T any](data []byte) []T {
	return Slice[T](data)
}

func BytesToString(data []byte) string {
	return *(*string)(unsafe.Pointer(&data))
}
