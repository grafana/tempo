package delta

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"math/bits"

	"github.com/segmentio/parquet-go/encoding"
	"github.com/segmentio/parquet-go/format"
)

const (
	blockSize     = 128
	numMiniBlocks = 4
	miniBlockSize = blockSize / numMiniBlocks
	// The parquet spec does not enforce a limit to the block size, but we need
	// one otherwise invalid inputs may result in unbounded memory allocations.
	//
	// 65K+ values should be enough for any valid use case.
	maxSupportedBlockSize = 65536

	maxHeaderLength32    = 4 * binary.MaxVarintLen64
	maxMiniBlockLength32 = binary.MaxVarintLen64 + numMiniBlocks + (4 * blockSize)

	maxHeaderLength64    = 8 * binary.MaxVarintLen64
	maxMiniBlockLength64 = binary.MaxVarintLen64 + numMiniBlocks + (8 * blockSize)
)

var (
	encodeInt32 = encodeInt32Default
	encodeInt64 = encodeInt64Default
)

func encodeInt32Default(dst []byte, src []int32) []byte {
	totalValues := len(src)
	firstValue := int32(0)
	if totalValues > 0 {
		firstValue = src[0]
	}

	n := len(dst)
	dst = resize(dst, n+maxHeaderLength32)
	dst = dst[:n+encodeBinaryPackedHeader(dst[n:], blockSize, numMiniBlocks, totalValues, int64(firstValue))]

	if totalValues < 2 {
		return dst
	}

	lastValue := firstValue
	for i := 1; i < len(src); i += blockSize {
		block := [blockSize]int32{}
		blockLength := copy(block[:], src[i:])

		lastValue = blockDeltaInt32(&block, lastValue)
		minDelta := blockMinInt32(&block)
		blockSubInt32(&block, minDelta)
		blockClearInt32(&block, blockLength)

		bitWidths := [numMiniBlocks]byte{}
		blockBitWidthsInt32(&bitWidths, &block)

		n := len(dst)
		dst = resize(dst, n+maxMiniBlockLength32+4)
		n += encodeBlockHeader(dst[n:], int64(minDelta), bitWidths)

		for i, bitWidth := range bitWidths {
			if bitWidth != 0 {
				miniBlock := (*[miniBlockSize]int32)(block[i*miniBlockSize:])
				miniBlockPackInt32(dst[n:], miniBlock, uint(bitWidth))
				n += (miniBlockSize * int(bitWidth)) / 8
			}
		}

		dst = dst[:n]
	}

	return dst
}

func encodeInt64Default(dst []byte, src []int64) []byte {
	totalValues := len(src)
	firstValue := int64(0)
	if totalValues > 0 {
		firstValue = src[0]
	}

	n := len(dst)
	dst = resize(dst, n+maxHeaderLength64)
	dst = dst[:n+encodeBinaryPackedHeader(dst[n:], blockSize, numMiniBlocks, totalValues, firstValue)]

	if totalValues < 2 {
		return dst
	}

	lastValue := firstValue
	for i := 1; i < len(src); i += blockSize {
		block := [blockSize]int64{}
		blockLength := copy(block[:], src[i:])

		lastValue = blockDeltaInt64(&block, lastValue)
		minDelta := blockMinInt64(&block)
		blockSubInt64(&block, minDelta)
		blockClearInt64(&block, blockLength)

		bitWidths := [numMiniBlocks]byte{}
		blockBitWidthsInt64(&bitWidths, &block)

		n := len(dst)
		dst = resize(dst, n+maxMiniBlockLength64+8)
		n += encodeBlockHeader(dst[n:], minDelta, bitWidths)

		for i, bitWidth := range bitWidths {
			if bitWidth != 0 {
				miniBlock := (*[miniBlockSize]int64)(block[i*miniBlockSize:])
				miniBlockPackInt64(dst[n:], miniBlock, uint(bitWidth))
				n += (miniBlockSize * int(bitWidth)) / 8
			}
		}

		dst = dst[:n]
	}

	return dst
}

func encodeBinaryPackedHeader(dst []byte, blockSize, numMiniBlocks, totalValues int, firstValue int64) (n int) {
	n += binary.PutUvarint(dst[n:], uint64(blockSize))
	n += binary.PutUvarint(dst[n:], uint64(numMiniBlocks))
	n += binary.PutUvarint(dst[n:], uint64(totalValues))
	n += binary.PutVarint(dst[n:], firstValue)
	return n
}

func encodeBlockHeader(dst []byte, minDelta int64, bitWidths [numMiniBlocks]byte) (n int) {
	n += binary.PutVarint(dst, int64(minDelta))
	n += copy(dst[n:], bitWidths[:])
	return n
}

func blockClearInt32(block *[blockSize]int32, blockLength int) {
	if blockLength < blockSize {
		clear := block[blockLength:]
		for i := range clear {
			clear[i] = 0
		}
	}
}

func blockDeltaInt32(block *[blockSize]int32, lastValue int32) int32 {
	for i, v := range block {
		block[i], lastValue = v-lastValue, v
	}
	return lastValue
}

func blockMinInt32(block *[blockSize]int32) int32 {
	min := block[0]
	for _, v := range block[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func blockSubInt32(block *[blockSize]int32, value int32) {
	for i := range block {
		block[i] -= value
	}
}

func blockBitWidthsInt32(bitWidths *[numMiniBlocks]byte, block *[blockSize]int32) {
	for i := range bitWidths {
		j := (i + 0) * miniBlockSize
		k := (i + 1) * miniBlockSize
		bitWidth := 0

		for _, v := range block[j:k] {
			if n := bits.Len32(uint32(v)); n > bitWidth {
				bitWidth = n
			}
		}

		bitWidths[i] = byte(bitWidth)
	}
}

func blockClearInt64(block *[blockSize]int64, blockLength int) {
	if blockLength < blockSize {
		clear := block[blockLength:]
		for i := range clear {
			clear[i] = 0
		}
	}
}

func blockDeltaInt64(block *[blockSize]int64, lastValue int64) int64 {
	for i, v := range block {
		block[i], lastValue = v-lastValue, v
	}
	return lastValue
}

func blockMinInt64(block *[blockSize]int64) int64 {
	min := block[0]
	for _, v := range block[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func blockSubInt64(block *[blockSize]int64, value int64) {
	for i := range block {
		block[i] -= value
	}
}

func blockBitWidthsInt64(bitWidths *[numMiniBlocks]byte, block *[blockSize]int64) {
	for i := range bitWidths {
		j := (i + 0) * miniBlockSize
		k := (i + 1) * miniBlockSize
		bitWidth := 0

		for _, v := range block[j:k] {
			if n := bits.Len64(uint64(v)); n > bitWidth {
				bitWidth = n
			}
		}

		bitWidths[i] = byte(bitWidth)
	}
}

func resize(buf []byte, size int) []byte {
	if cap(buf) < size {
		return grow(buf, size)
	}
	if size > len(buf) {
		clear := buf[len(buf):size]
		for i := range clear {
			clear[i] = 0
		}
	}
	return buf[:size]
}

func grow(buf []byte, size int) []byte {
	newCap := 2 * cap(buf)
	if newCap < size {
		newCap = size
	}
	newBuf := make([]byte, size, newCap)
	copy(newBuf, buf)
	return newBuf
}

type BinaryPackedEncoding struct {
	encoding.NotSupported
}

func (e *BinaryPackedEncoding) String() string {
	return "DELTA_BINARY_PACKED"
}

func (e *BinaryPackedEncoding) Encoding() format.Encoding {
	return format.DeltaBinaryPacked
}

func (e *BinaryPackedEncoding) EncodeInt32(dst, src []byte) ([]byte, error) {
	if (len(src) % 4) != 0 {
		return dst[:0], encoding.ErrEncodeInvalidInputSize(e, "INT64", len(src))
	}
	return encodeInt32(dst[:0], bytesToInt32(src)), nil
}

func (e *BinaryPackedEncoding) EncodeInt64(dst, src []byte) ([]byte, error) {
	if (len(src) % 8) != 0 {
		return dst[:0], encoding.ErrEncodeInvalidInputSize(e, "INT64", len(src))
	}
	return encodeInt64(dst[:0], bytesToInt64(src)), nil
}

func (e *BinaryPackedEncoding) DecodeInt32(dst, src []byte) ([]byte, error) {
	dst, _, err := e.decodeInt32(dst[:0], src)
	return dst, e.wrap(err)
}

func (e *BinaryPackedEncoding) DecodeInt64(dst, src []byte) ([]byte, error) {
	dst, _, err := e.decodeInt64(dst[:0], src)
	return dst, e.wrap(err)
}

func (e *BinaryPackedEncoding) decodeInt32(dst, src []byte) ([]byte, []byte, error) {
	src, err := e.decode(src, func(value int64) {
		buf := [4]byte{}
		binary.LittleEndian.PutUint32(buf[:], uint32(value))
		dst = append(dst, buf[:]...)
	})
	return dst, src, err
}

func (e *BinaryPackedEncoding) decodeInt64(dst, src []byte) ([]byte, []byte, error) {
	src, err := e.decode(src, func(value int64) {
		buf := [8]byte{}
		binary.LittleEndian.PutUint64(buf[:], uint64(value))
		dst = append(dst, buf[:]...)
	})
	return dst, src, err
}

func (e *BinaryPackedEncoding) decode(src []byte, observe func(int64)) ([]byte, error) {
	blockSize, numMiniBlocks, totalValues, firstValue, src, err := decodeBinaryPackedHeader(src)
	if err != nil {
		return src, err
	}
	if totalValues == 0 {
		return src, nil
	}

	observe(firstValue)
	totalValues--
	lastValue := firstValue
	numValuesInMiniBlock := blockSize / numMiniBlocks

	block := make([]int64, 128)
	if cap(block) < blockSize {
		block = make([]int64, blockSize)
	} else {
		block = block[:blockSize]
	}

	miniBlockData := make([]byte, 256)

	for totalValues > 0 && len(src) > 0 {
		var minDelta int64
		var bitWidths []byte
		minDelta, bitWidths, src, err = decodeBinaryPackedBlock(src, numMiniBlocks)
		if err != nil {
			return src, err
		}

		blockOffset := 0
		for i := range block {
			block[i] = 0
		}

		for _, bitWidth := range bitWidths {
			if bitWidth == 0 {
				n := numValuesInMiniBlock
				if n > totalValues {
					n = totalValues
				}
				blockOffset += n
				totalValues -= n
			} else {
				miniBlockSize := (numValuesInMiniBlock * int(bitWidth)) / 8
				if cap(miniBlockData) < miniBlockSize {
					miniBlockData = make([]byte, miniBlockSize, miniBlockSize)
				} else {
					miniBlockData = miniBlockData[:miniBlockSize]
				}

				n := copy(miniBlockData, src)
				src = src[n:]
				bitOffset := uint(0)

				for count := numValuesInMiniBlock; count > 0 && totalValues > 0; count-- {
					delta := int64(0)

					for b := uint(0); b < uint(bitWidth); b++ {
						x := (bitOffset + b) / 8
						y := (bitOffset + b) % 8
						delta |= int64((miniBlockData[x]>>y)&1) << b
					}

					block[blockOffset] = delta
					blockOffset++
					totalValues--
					bitOffset += uint(bitWidth)
				}
			}

			if totalValues == 0 {
				break
			}
		}

		for i := range block {
			block[i] += minDelta
		}

		block[0] += lastValue
		for i := 1; i < len(block); i++ {
			block[i] += block[i-1]
		}
		if values := block[:blockOffset]; len(values) > 0 {
			for _, v := range values {
				observe(v)
			}
			lastValue = values[len(values)-1]
		}
	}

	if totalValues > 0 {
		return src, fmt.Errorf("%d missing values: %w", totalValues, io.ErrUnexpectedEOF)
	}

	return src, nil
}

func (e *BinaryPackedEncoding) wrap(err error) error {
	if err != nil {
		err = encoding.Error(e, err)
	}
	return err
}

func decodeBinaryPackedHeader(src []byte) (blockSize, numMiniBlocks, totalValues int, firstValue int64, next []byte, err error) {
	u := uint64(0)
	n := 0
	i := 0

	if u, n, err = decodeUvarint(src[i:], "block size"); err != nil {
		return
	}
	i += n
	blockSize = int(u)

	if u, n, err = decodeUvarint(src[i:], "number of mini-blocks"); err != nil {
		return
	}
	i += n
	numMiniBlocks = int(u)

	if u, n, err = decodeUvarint(src[i:], "total values"); err != nil {
		return
	}
	i += n
	totalValues = int(u)

	if firstValue, n, err = decodeVarint(src[i:], "first value"); err != nil {
		return
	}
	i += n

	if numMiniBlocks == 0 {
		err = fmt.Errorf("invalid number of mini block (%d)", numMiniBlocks)
	} else if (blockSize <= 0) || (blockSize%128) != 0 {
		err = fmt.Errorf("invalid block size is not a multiple of 128 (%d)", blockSize)
	} else if blockSize > maxSupportedBlockSize {
		err = fmt.Errorf("invalid block size is too large (%d)", blockSize)
	} else if miniBlockSize := blockSize / numMiniBlocks; (numMiniBlocks <= 0) || (miniBlockSize%32) != 0 {
		err = fmt.Errorf("invalid mini block size is not a multiple of 32 (%d)", miniBlockSize)
	} else if totalValues < 0 {
		err = fmt.Errorf("invalid total number of values is negative (%d)", totalValues)
	} else if totalValues > math.MaxInt32 {
		err = fmt.Errorf("too many values: %d", totalValues)
	}

	return blockSize, numMiniBlocks, totalValues, firstValue, src[i:], err
}

func decodeBinaryPackedBlock(src []byte, numMiniBlocks int) (minDelta int64, bitWidths, next []byte, err error) {
	minDelta, n, err := decodeVarint(src, "min delta")
	if err != nil {
		return 0, nil, src, err
	}
	src = src[n:]
	if len(src) < numMiniBlocks {
		bitWidths, next = src, nil
	} else {
		bitWidths, next = src[:numMiniBlocks], src[numMiniBlocks:]
	}
	return minDelta, bitWidths, next, nil
}

func decodeUvarint(buf []byte, what string) (u uint64, n int, err error) {
	u, n = binary.Uvarint(buf)
	if n == 0 {
		return 0, 0, fmt.Errorf("decoding %s: %w", what, io.ErrUnexpectedEOF)
	}
	if n < 0 {
		return 0, 0, fmt.Errorf("overflow decoding %s (read %d/%d bytes)", what, -n, len(buf))
	}
	return u, n, nil
}

func decodeVarint(buf []byte, what string) (v int64, n int, err error) {
	v, n = binary.Varint(buf)
	if n == 0 {
		return 0, 0, fmt.Errorf("decoding %s: %w", what, io.ErrUnexpectedEOF)
	}
	if n < 0 {
		return 0, 0, fmt.Errorf("overflow decoding %s (read %d/%d bytes)", what, -n, len(buf))
	}
	return v, n, nil
}
