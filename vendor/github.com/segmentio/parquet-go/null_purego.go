//go:build go1.18 && (purego || !amd64)

package parquet

func nullIndexBool(bits []uint64, rows array, size, offset uintptr) {
	nullIndex[bool](bits, rows, size, offset)
}

func nullIndexInt(bits []uint64, rows array, size, offset uintptr) {
	nullIndex[int](bits, rows, size, offset)
}

func nullIndexInt32(bits []uint64, rows array, size, offset uintptr) {
	nullIndex[int32](bits, rows, size, offset)
}

func nullIndexInt64(bits []uint64, rows array, size, offset uintptr) {
	nullIndex[int64](bits, rows, size, offset)
}

func nullIndexUint(bits []uint64, rows array, size, offset uintptr) {
	nullIndex[uint](bits, rows, size, offset)
}

func nullIndexUint32(bits []uint64, rows array, size, offset uintptr) {
	nullIndex[uint32](bits, rows, size, offset)
}

func nullIndexUint64(bits []uint64, rows array, size, offset uintptr) {
	nullIndex[uint64](bits, rows, size, offset)
}

func nullIndexUint128(bits []uint64, rows array, size, offset uintptr) {
	nullIndex[[16]byte](bits, rows, size, offset)
}

func nullIndexFloat32(bits []uint64, rows array, size, offset uintptr) {
	nullIndex[float32](bits, rows, size, offset)
}

func nullIndexFloat64(bits []uint64, rows array, size, offset uintptr) {
	nullIndex[float64](bits, rows, size, offset)
}

func nullIndexString(bits []uint64, rows array, size, offset uintptr) {
	nullIndex[string](bits, rows, size, offset)
}

func nullIndexSlice(bits []uint64, rows array, size, offset uintptr) {
	for i := 0; i < rows.len; i++ {
		p := *(**struct{})(rows.index(i, size, offset))
		b := uint64(0)
		if p != nil {
			b = 1
		}
		bits[uint(i)/64] |= b << (uint(i) % 64)
	}
}

func nullIndexPointer(bits []uint64, rows array, size, offset uintptr) {
	nullIndex[*struct{}](bits, rows, size, offset)
}
