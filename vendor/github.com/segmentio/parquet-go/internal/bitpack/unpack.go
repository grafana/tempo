package bitpack

// Padding is the padding expected to exist after the end of input buffers for
// the unpacking algorithms to avoid reading beyond the end of the input.
const Padding = 16

// UnpackInt32 unpacks 32 bit integers from src to dst.
//
// The function unpacked len(dst) integers, it panics if src is too short to
// contain len(dst) values of the given bit width.
func UnpackInt32(dst []int32, src []byte, bitWidth uint) {
	assertUnpack(src, len(dst), bitWidth)
	unpackInt32(dst, src, bitWidth)
}

// UnpackInt64 unpacks 64 bit integers from src to dst.
//
// The function unpacked len(dst) integers, it panics if src is too short to
// contain len(dst) values of the given bit width.
func UnpackInt64(dst []int64, src []byte, bitWidth uint) {
	assertUnpack(src, len(dst), bitWidth)
	unpackInt64(dst, src, bitWidth)
}

func assertUnpack(src []byte, count int, bitWidth uint) {
	_ = src[:ByteCount(bitWidth*uint(count)+8*Padding)]
}
