//go:build purego || !amd64

package bitpack

import (
	"encoding/binary"

	"golang.org/x/sys/cpu"

	"github.com/parquet-go/parquet-go/internal/unsafecast"
)

func unpackInt64(dst []int64, src []byte, bitWidth uint) {
	var bits []uint32
	if cpu.IsBigEndian {
		srcLen := (len(src) / 4)
		bits = make([]uint32, srcLen)
		idx := 0
		for k := range srcLen {
			bits[k] = binary.LittleEndian.Uint32((src)[idx:(4 + idx)])
			idx += 4
		}
	} else {
		bits = unsafecast.Slice[uint32](src)
	}

	bitMask := uint64(1<<bitWidth) - 1
	bitOffset := uint(0)

	for n := range dst {
		i := bitOffset / 32
		j := bitOffset % 32
		d := (uint64(bits[i]) & (bitMask << j)) >> j
		if j+bitWidth > 32 {
			k := 32 - j
			d |= (uint64(bits[i+1]) & (bitMask >> k)) << k
			if j+bitWidth > 64 {
				k := 64 - j
				d |= (uint64(bits[i+2]) & (bitMask >> k)) << k
			}
		}
		dst[n] = int64(d)
		bitOffset += bitWidth
	}
}
