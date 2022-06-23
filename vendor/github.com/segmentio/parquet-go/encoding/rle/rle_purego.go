//go:build purego || !amd64

package rle

import (
	"encoding/binary"

	"github.com/segmentio/parquet-go/internal/bitpack"
)

func encodeBytesBitpack(dst []byte, src []uint64, bitWidth uint) int {
	bitMask := uint64(1<<bitWidth) - 1
	n := 0

	for _, word := range src {
		word = (word & bitMask) |
			(((word >> 8) & bitMask) << (1 * bitWidth)) |
			(((word >> 16) & bitMask) << (2 * bitWidth)) |
			(((word >> 24) & bitMask) << (3 * bitWidth)) |
			(((word >> 32) & bitMask) << (4 * bitWidth)) |
			(((word >> 40) & bitMask) << (5 * bitWidth)) |
			(((word >> 48) & bitMask) << (6 * bitWidth)) |
			(((word >> 56) & bitMask) << (7 * bitWidth))
		binary.LittleEndian.PutUint64(dst[n:], word)
		n += int(bitWidth)
	}

	return n
}

func encodeInt32IndexEqual8Contiguous(words [][8]int32) (n int) {
	for n < len(words) && words[n] != broadcast8x4(words[n][0]) {
		n++
	}
	return n
}

func encodeInt32Bitpack(dst []byte, src [][8]int32, bitWidth uint) int {
	return encodeInt32BitpackDefault(dst, src, bitWidth)
}

func decodeBytesBitpack(dst, src []byte, count, bitWidth uint) {
	dst = dst[:0]

	bitMask := uint64(1<<bitWidth) - 1
	byteCount := bitpack.ByteCount(8 * bitWidth)

	for i := 0; count > 0; count -= 8 {
		j := i + byteCount

		bits := [8]byte{}
		copy(bits[:], src[i:j])
		word := binary.LittleEndian.Uint64(bits[:])

		dst = append(dst,
			byte((word>>(0*bitWidth))&bitMask),
			byte((word>>(1*bitWidth))&bitMask),
			byte((word>>(2*bitWidth))&bitMask),
			byte((word>>(3*bitWidth))&bitMask),
			byte((word>>(4*bitWidth))&bitMask),
			byte((word>>(5*bitWidth))&bitMask),
			byte((word>>(6*bitWidth))&bitMask),
			byte((word>>(7*bitWidth))&bitMask),
		)

		i = j
	}
}
