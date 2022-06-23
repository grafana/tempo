//go:build !amd64 || purego

package parquet

func broadcastValueInt32(dst []int32, src int8) {
	value := 0x01010101 * int32(src)
	for i := range dst {
		dst[i] = value
	}
}

func broadcastRangeInt32(dst []int32, base int32) {
	for i := range dst {
		dst[i] = base + int32(i)
	}
}

func writeValuesBool(values []byte, rows array, size, offset uintptr) {
	panic("unreachable")
}

func writeValuesInt32(values []int32, rows array, size, offset uintptr) {
	panic("unreachable")
}

func writeValuesInt64(values []int64, rows array, size, offset uintptr) {
	panic("unreachable")
}

func writeValuesUint32(values []uint32, rows array, size, offset uintptr) {
	panic("unreachable")
}

func writeValuesUint64(values []uint64, rows array, size, offset uintptr) {
	panic("unreachable")
}

func writeValuesFloat32(values []float32, rows array, size, offset uintptr) {
	panic("unreachable")
}

func writeValuesFloat64(values []float64, rows array, size, offset uintptr) {
	panic("unreachable")
}

func writeValuesBE128(values [][16]byte, rows array, size, offset uintptr) {
	for i := range values {
		values[i] = *(*[16]byte)(rows.index(i, size, offset))
	}
}
