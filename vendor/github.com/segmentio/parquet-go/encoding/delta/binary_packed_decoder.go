package delta

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/segmentio/parquet-go/encoding"
	"github.com/segmentio/parquet-go/internal/bits"
)

type BinaryPackedDecoder struct {
	encoding.NotSupportedDecoder
	reader        *bufio.Reader
	blockSize     int
	numMiniBlock  int
	miniBlockSize int
	totalValues   int
	lastValue     int64
	bitWidths     []byte
	blockValues   []int64
	valueIndex    int
	blockIndex    int
	miniBlocks    bits.Reader
}

func NewBinaryPackedDecoder(r io.Reader) *BinaryPackedDecoder {
	d := &BinaryPackedDecoder{}
	d.Reset(r)
	return d
}

func (d *BinaryPackedDecoder) Reset(r io.Reader) {
	*d = BinaryPackedDecoder{
		reader:      d.reader,
		bitWidths:   d.bitWidths[:0],
		blockValues: d.blockValues[:0],
		valueIndex:  -1,
	}

	if cap(d.blockValues) == 0 {
		d.blockValues = make([]int64, 0, blockSize32)
	}

	if rbuf, _ := r.(*bufio.Reader); rbuf != nil {
		d.reader = rbuf
	} else if d.reader != nil {
		d.reader.Reset(r)
	} else if r != nil {
		d.reader = bufio.NewReaderSize(r, defaultBufferSize)
	}

	d.miniBlocks.Reset(d.reader)
}

func (d *BinaryPackedDecoder) DecodeInt32(data []int32) (int, error) {
	decoded := 0

	for len(data) > 0 {
		if err := d.decode(); err != nil {
			if err == io.EOF && decoded > 0 {
				break
			}
			return decoded, err
		}

		i := d.blockIndex
		j := len(d.blockValues)
		remain := d.totalValues - d.valueIndex

		if (j - i) > remain {
			j = i + remain
		}

		n := j - i
		if n > len(data) {
			n = len(data)
			j = i + n
		}

		for i, v := range d.blockValues[i:j] {
			data[i] = int32(v)
		}

		data = data[n:]
		decoded += n
		d.valueIndex += n
		d.blockIndex += n
	}

	return decoded, nil
}

func (d *BinaryPackedDecoder) DecodeInt64(data []int64) (int, error) {
	decoded := 0

	for len(data) > 0 {
		if err := d.decode(); err != nil {
			if err == io.EOF && decoded > 0 {
				break
			}
			return decoded, err
		}

		n := copy(data, d.blockValues[d.blockIndex:])
		data = data[n:]
		decoded += n
		d.valueIndex += n
		d.blockIndex += n
	}

	return decoded, nil
}

func (d *BinaryPackedDecoder) decode() error {
	if d.valueIndex < 0 {
		blockSize, numMiniBlock, totalValues, firstValue, err := d.decodeHeader()
		if err != nil {
			return err
		}

		d.blockSize = blockSize
		d.numMiniBlock = numMiniBlock
		d.miniBlockSize = blockSize / numMiniBlock
		d.totalValues = totalValues
		d.lastValue = firstValue
		d.valueIndex = 0
		d.blockIndex = 0

		if d.totalValues > 0 {
			d.blockValues = append(d.blockValues[:0], firstValue)
		}

		return nil
	}

	if d.valueIndex == d.totalValues {
		return io.EOF
	}

	if d.blockIndex == 0 || d.blockIndex == len(d.blockValues) {
		if err := d.decodeBlock(); err != nil {
			return err
		}
		d.blockIndex = 0
	}

	return nil
}

func (d *BinaryPackedDecoder) decodeHeader() (blockSize, numMiniBlock, totalValues int, firstValue int64, err error) {
	var u uint64

	if u, err = binary.ReadUvarint(d.reader); err != nil {
		if err != io.EOF {
			err = fmt.Errorf("DELTA_BINARY_PACKED: reading block size: %w", err)
		}
		return
	} else {
		blockSize = int(u)
	}
	if u, err = binary.ReadUvarint(d.reader); err != nil {
		err = fmt.Errorf("DELTA_BINARY_PACKED: reading number of mini blocks: %w", dontExpectEOF(err))
		return
	} else {
		numMiniBlock = int(u)
	}
	if u, err = binary.ReadUvarint(d.reader); err != nil {
		err = fmt.Errorf("DELTA_BINARY_PACKED: reading number of values: %w", dontExpectEOF(err))
		return
	} else {
		totalValues = int(u)
	}
	if firstValue, err = binary.ReadVarint(d.reader); err != nil {
		err = fmt.Errorf("DELTA_BINARY_PACKED: reading first value: %w", dontExpectEOF(err))
		return
	}

	if numMiniBlock == 0 {
		err = fmt.Errorf("DELTA_BINARY_PACKED: invalid number of mini block (%d)", numMiniBlock)
	} else if (blockSize <= 0) || (blockSize%128) != 0 {
		err = fmt.Errorf("DELTA_BINARY_PACKED: invalid block size is not a multiple of 128 (%d)", blockSize)
	} else if miniBlockSize := blockSize / numMiniBlock; (numMiniBlock <= 0) || (miniBlockSize%32) != 0 {
		err = fmt.Errorf("DELTA_BINARY_PACKED: invalid mini block size is not a multiple of 32 (%d)", miniBlockSize)
	} else if totalValues < 0 {
		err = fmt.Errorf("DETLA_BINARY_PACKED: invalid total number of values is negative (%d)", totalValues)
	}
	return
}

func (d *BinaryPackedDecoder) decodeBlock() error {
	minDelta, err := binary.ReadVarint(d.reader)
	if err != nil {
		return fmt.Errorf("DELTA_BINARY_PACKED: reading min delta (%d): %w", minDelta, err)
	}

	if cap(d.bitWidths) < d.numMiniBlock {
		d.bitWidths = make([]byte, d.numMiniBlock)
	} else {
		d.bitWidths = d.bitWidths[:d.numMiniBlock]
	}

	if _, err := io.ReadFull(d.reader, d.bitWidths); err != nil {
		return fmt.Errorf("DELTA_BINARY_PACKED: reading bit widths: %w", err)
	}

	if cap(d.blockValues) < d.blockSize {
		d.blockValues = make([]int64, d.blockSize)
	} else {
		d.blockValues = d.blockValues[:d.blockSize]
	}

	for i := range d.blockValues {
		d.blockValues[i] = 0
	}

	i := 0
	j := d.miniBlockSize
	remain := d.totalValues - d.valueIndex

	for _, bitWidth := range d.bitWidths {
		if bitWidth != 0 {
			for k := range d.blockValues[i:j] {
				v, nbits, err := d.miniBlocks.ReadBits(uint(bitWidth))
				if err != nil {
					// In some cases, the last mini block seems to be missing
					// trailing bytes when all values have already been decoded.
					//
					// The spec is unclear on the topic, it says that no padding
					// is added for the miniblocks that contain no values, tho
					// it is not explicit on whether the last miniblock is
					// allowed to be incomplete.
					//
					// When we remove padding on the miniblock containing the
					// last value, parquet-tools sometimes fails to read the
					// column. However, if we don't handle the case where EOF
					// is reached before reading the full last miniblock, we
					// are unable to read some of the reference files from the
					// parquet-testing repository.
					if err == io.EOF && (i+k) >= remain {
						break
					}
					err = dontExpectEOF(err)
					err = fmt.Errorf("DELTA_BINARY_PACKED: reading mini blocks: %w", err)
					return err
				}
				if nbits != uint(bitWidth) {
					panic("BUG: wrong number of bits read from DELTA_BINARY_PACKED miniblock")
				}
				d.blockValues[i+k] = int64(v)
			}
		}

		if j >= remain {
			break
		}

		i += d.miniBlockSize
		j += d.miniBlockSize
	}

	if remain < len(d.blockValues) {
		d.blockValues = d.blockValues[:remain]
	}

	bits.AddInt64(d.blockValues, minDelta)
	d.blockValues[0] += d.lastValue
	for i := 1; i < len(d.blockValues); i++ {
		d.blockValues[i] += d.blockValues[i-1]
	}
	d.lastValue = d.blockValues[len(d.blockValues)-1]
	return nil
}

func dontExpectEOF(err error) error {
	if err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	return err
}
