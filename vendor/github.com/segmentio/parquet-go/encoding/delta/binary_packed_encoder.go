package delta

import (
	"encoding/binary"
	"io"

	"github.com/segmentio/parquet-go/encoding"
	"github.com/segmentio/parquet-go/internal/bits"
)

// TODO: figure out better heuristics to determine those values,
// right now they are optimized for keeping the memory footprint
// of the encoder/decoder at ~8KB.
const (
	blockSize64     = 128
	numMiniBlock64  = 4 // (blockSize64 / numMiniBlock64) % 32 == 0
	miniBlockSize64 = blockSize64 / numMiniBlock64

	blockSize32     = 2 * blockSize64
	numMiniBlock32  = 2 * numMiniBlock64
	miniBlockSize32 = blockSize32 / numMiniBlock32

	headerBufferSize    = 32
	blockBufferSize     = 8 * blockSize64
	bitWidthsBufferSize = 2 * numMiniBlock64
)

type BinaryPackedEncoder struct {
	encoding.NotSupportedEncoder
	writer    io.Writer
	header    [headerBufferSize]byte
	block     [blockBufferSize]byte
	bitWidths [bitWidthsBufferSize]byte
	miniBlock bits.Writer
}

func NewBinaryPackedEncoder(w io.Writer) *BinaryPackedEncoder {
	e := &BinaryPackedEncoder{}
	e.Reset(w)
	return e
}

func (e *BinaryPackedEncoder) Reset(w io.Writer) {
	e.writer = w
	e.miniBlock.Reset(w)
}

func (e *BinaryPackedEncoder) EncodeInt32(data []int32) error {
	firstValue := int32(0)
	if len(data) > 0 {
		firstValue = data[0]
	}

	if err := e.encodeHeader(blockSize32, numMiniBlock32, len(data), int64(firstValue)); err != nil {
		return err
	}

	if len(data) <= 1 {
		return nil
	}

	data = data[1:]
	lastValue := firstValue

	for len(data) > 0 {
		block := bits.BytesToInt32(e.block[:])
		for i := range block {
			block[i] = 0
		}

		n := copy(block, data)
		data = data[n:]

		for i, v := range block[:n] {
			block[i], lastValue = v-lastValue, v
		}

		minDelta := bits.MinInt32(block[:n])
		bits.SubInt32(block[:n], minDelta)

		bitWidths := e.bitWidths[:numMiniBlock32]
		for i := range bitWidths {
			j := (i + 0) * miniBlockSize32
			k := (i + 1) * miniBlockSize32
			bitWidths[i] = byte(bits.MaxLen32(block[j:k]))
		}

		if err := e.encodeBlock(int64(minDelta), bitWidths); err != nil {
			return err
		}

		for i, bitWidth := range bitWidths {
			j := (i + 0) * miniBlockSize32
			k := (i + 1) * miniBlockSize32
			if bitWidth != 0 {
				for _, bits := range block[j:k] {
					e.miniBlock.WriteBits(uint64(bits), uint(bitWidth))
				}
			}
			if k >= n {
				break
			}
		}

		if err := e.miniBlock.Flush(); err != nil {
			return err
		}
	}

	return nil
}

func (e *BinaryPackedEncoder) EncodeInt64(data []int64) error {
	firstValue := int64(0)
	if len(data) > 0 {
		firstValue = data[0]
	}

	if err := e.encodeHeader(blockSize64, numMiniBlock64, len(data), firstValue); err != nil {
		return err
	}

	if len(data) <= 1 {
		return nil
	}

	data = data[1:]
	lastValue := firstValue

	for len(data) > 0 {
		block := bits.BytesToInt64(e.block[:])
		for i := range block {
			block[i] = 0
		}

		n := copy(block, data)
		data = data[n:]

		for i, v := range block[:n] {
			block[i], lastValue = v-lastValue, v
		}

		minDelta := bits.MinInt64(block)
		bits.SubInt64(block, minDelta)

		bitWidths := e.bitWidths[:numMiniBlock64]
		for i := range bitWidths {
			j := (i + 0) * miniBlockSize64
			k := (i + 1) * miniBlockSize64
			bitWidths[i] = byte(bits.MaxLen64(block[j:k]))
		}

		if err := e.encodeBlock(minDelta, bitWidths); err != nil {
			return err
		}

		for i, bitWidth := range bitWidths {
			j := (i + 0) * miniBlockSize64
			k := (i + 1) * miniBlockSize64
			if bitWidth != 0 {
				for _, bits := range block[j:k] {
					e.miniBlock.WriteBits(uint64(bits), uint(bitWidth))
				}
			}
			if k >= n {
				break
			}
		}

		if err := e.miniBlock.Flush(); err != nil {
			return err
		}
	}

	return nil
}

func (e *BinaryPackedEncoder) encodeHeader(blockSize, numMiniBlock, totalValues int, firstValue int64) error {
	b := e.header[:]
	n := 0
	n += binary.PutUvarint(b[n:], uint64(blockSize))
	n += binary.PutUvarint(b[n:], uint64(numMiniBlock))
	n += binary.PutUvarint(b[n:], uint64(totalValues))
	n += binary.PutVarint(b[n:], firstValue)
	_, err := e.writer.Write(b[:n])
	return err
}

func (e *BinaryPackedEncoder) encodeBlock(minDelta int64, bitWidths []byte) error {
	b := e.header[:]
	n := binary.PutVarint(b, minDelta)
	if _, err := e.writer.Write(b[:n]); err != nil {
		return err
	}
	_, err := e.writer.Write(bitWidths)
	return err
}
