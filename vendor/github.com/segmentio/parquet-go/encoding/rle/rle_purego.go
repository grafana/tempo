//go:build purego || !amd64

package rle

import (
	"encoding/binary"
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
