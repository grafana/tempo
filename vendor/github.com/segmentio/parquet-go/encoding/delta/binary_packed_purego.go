//go:build purego || !amd64

package delta

import (
	"encoding/binary"
)

func miniBlockPackInt32(dst []byte, src *[miniBlockSize]int32, bitWidth uint) {
	bitMask := uint32(1<<bitWidth) - 1
	bitOffset := uint(0)

	for _, value := range src {
		i := bitOffset / 32
		j := bitOffset % 32

		lo := binary.LittleEndian.Uint32(dst[(i+0)*4:])
		hi := binary.LittleEndian.Uint32(dst[(i+1)*4:])

		lo |= (uint32(value) & bitMask) << j
		hi |= (uint32(value) >> (32 - j))

		binary.LittleEndian.PutUint32(dst[(i+0)*4:], lo)
		binary.LittleEndian.PutUint32(dst[(i+1)*4:], hi)

		bitOffset += bitWidth
	}
}

func miniBlockPackInt64(dst []byte, src *[miniBlockSize]int64, bitWidth uint) {
	bitMask := uint64(1<<bitWidth) - 1
	bitOffset := uint(0)

	for _, value := range src {
		i := bitOffset / 64
		j := bitOffset % 64

		lo := binary.LittleEndian.Uint64(dst[(i+0)*8:])
		hi := binary.LittleEndian.Uint64(dst[(i+1)*8:])

		lo |= (uint64(value) & bitMask) << j
		hi |= (uint64(value) >> (64 - j))

		binary.LittleEndian.PutUint64(dst[(i+0)*8:], lo)
		binary.LittleEndian.PutUint64(dst[(i+1)*8:], hi)

		bitOffset += bitWidth
	}
}
