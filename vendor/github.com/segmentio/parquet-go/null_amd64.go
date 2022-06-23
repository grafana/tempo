//go:build go1.18 && !purego

package parquet

//go:noescape
func nullIndex8bits(bits *uint64, rows array, size, offset uintptr)

//go:noescape
func nullIndex32bits(bits *uint64, rows array, size, offset uintptr)

//go:noescape
func nullIndex64bits(bits *uint64, rows array, size, offset uintptr)

//go:noescape
func nullIndex128bits(bits *uint64, rows array, size, offset uintptr)

func nullIndexBool(bits []uint64, rows array, size, offset uintptr) {
	nullIndex8bits(&bits[0], rows, size, offset)
}

func nullIndexInt(bits []uint64, rows array, size, offset uintptr) {
	nullIndex64bits(&bits[0], rows, size, offset)
}

func nullIndexInt32(bits []uint64, rows array, size, offset uintptr) {
	nullIndex32bits(&bits[0], rows, size, offset)
}

func nullIndexInt64(bits []uint64, rows array, size, offset uintptr) {
	nullIndex64bits(&bits[0], rows, size, offset)
}

func nullIndexUint(bits []uint64, rows array, size, offset uintptr) {
	nullIndex64bits(&bits[0], rows, size, offset)
}

func nullIndexUint32(bits []uint64, rows array, size, offset uintptr) {
	nullIndex32bits(&bits[0], rows, size, offset)
}

func nullIndexUint64(bits []uint64, rows array, size, offset uintptr) {
	nullIndex64bits(&bits[0], rows, size, offset)
}

func nullIndexUint128(bits []uint64, rows array, size, offset uintptr) {
	nullIndex128bits(&bits[0], rows, size, offset)
}

func nullIndexFloat32(bits []uint64, rows array, size, offset uintptr) {
	nullIndex32bits(&bits[0], rows, size, offset)
}

func nullIndexFloat64(bits []uint64, rows array, size, offset uintptr) {
	nullIndex64bits(&bits[0], rows, size, offset)
}

func nullIndexString(bits []uint64, rows array, size, offset uintptr) {
	// We offset by an extra 8 bytes to test the lengths of string values where
	// the first field is the pointer and the second is the length which we want
	// to test.
	nullIndex64bits(&bits[0], rows, size, offset+8)
}

func nullIndexSlice(bits []uint64, rows array, size, offset uintptr) {
	// Slice values are null if their pointer is nil, which is held in the first
	// 8 bytes of the object so we can simply test 64 bits words.
	nullIndex64bits(&bits[0], rows, size, offset)
}

func nullIndexPointer(bits []uint64, rows array, size, offset uintptr) {
	nullIndex64bits(&bits[0], rows, size, offset)
}
