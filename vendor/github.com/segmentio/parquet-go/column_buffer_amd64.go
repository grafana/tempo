//go:build !purego

package parquet

import (
	"unsafe"

	"github.com/segmentio/parquet-go/internal/bytealg"
	"github.com/segmentio/parquet-go/internal/unsafecast"
	"golang.org/x/sys/cpu"
)

func broadcastValueInt32(dst []int32, src int8) {
	bytealg.Broadcast(unsafecast.Int32ToBytes(dst), byte(src))
}

//go:noescape
func broadcastRangeInt32AVX2(dst []int32, base int32)

func broadcastRangeInt32(dst []int32, base int32) {
	if len(dst) >= minLenAVX2 && cpu.X86.HasAVX2 {
		broadcastRangeInt32AVX2(dst, base)
	} else {
		for i := range dst {
			dst[i] = base + int32(i)
		}
	}
}

//go:noescape
func writeValuesBitpackAVX2(values unsafe.Pointer, rows array, size, offset uintptr)

//go:noescape
func writeValues32bitsAVX2(values unsafe.Pointer, rows array, size, offset uintptr)

//go:noescpae
func writeValues64bitsAVX2(values unsafe.Pointer, rows array, size, offset uintptr)

//go:noescape
func writeValues128bits(values unsafe.Pointer, rows array, size, offset uintptr)

func writeValuesBool(values []byte, rows array, size, offset uintptr) {
	writeValuesBitpackAVX2(*(*unsafe.Pointer)(unsafe.Pointer(&values)), rows, size, offset)
}

func writeValuesInt32(values []int32, rows array, size, offset uintptr) {
	writeValues32bitsAVX2(*(*unsafe.Pointer)(unsafe.Pointer(&values)), rows, size, offset)
}

func writeValuesInt64(values []int64, rows array, size, offset uintptr) {
	writeValues64bitsAVX2(*(*unsafe.Pointer)(unsafe.Pointer(&values)), rows, size, offset)
}

func writeValuesUint32(values []uint32, rows array, size, offset uintptr) {
	writeValues32bitsAVX2(*(*unsafe.Pointer)(unsafe.Pointer(&values)), rows, size, offset)
}

func writeValuesUint64(values []uint64, rows array, size, offset uintptr) {
	writeValues64bitsAVX2(*(*unsafe.Pointer)(unsafe.Pointer(&values)), rows, size, offset)
}

func writeValuesFloat32(values []float32, rows array, size, offset uintptr) {
	writeValues32bitsAVX2(*(*unsafe.Pointer)(unsafe.Pointer(&values)), rows, size, offset)
}

func writeValuesFloat64(values []float64, rows array, size, offset uintptr) {
	writeValues64bitsAVX2(*(*unsafe.Pointer)(unsafe.Pointer(&values)), rows, size, offset)
}

func writeValuesBE128(values [][16]byte, rows array, size, offset uintptr) {
	writeValues128bits(*(*unsafe.Pointer)(unsafe.Pointer(&values)), rows, size, offset)
}
