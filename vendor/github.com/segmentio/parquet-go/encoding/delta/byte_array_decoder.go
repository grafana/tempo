package delta

import (
	"bufio"
	"fmt"
	"io"

	"github.com/segmentio/parquet-go/encoding"
)

type ByteArrayDecoder struct {
	encoding.NotSupportedDecoder
	deltas   BinaryPackedDecoder
	arrays   LengthByteArrayDecoder
	previous []byte
	prefixes []int32
}

func NewByteArrayDecoder(r io.Reader) *ByteArrayDecoder {
	d := &ByteArrayDecoder{prefixes: make([]int32, defaultBufferSize/4)}
	d.Reset(r)
	return d
}

func (d *ByteArrayDecoder) Reset(r io.Reader) {
	if _, ok := r.(*bufio.Reader); !ok {
		r = bufio.NewReaderSize(r, defaultBufferSize)
	}
	d.deltas.Reset(r)
	d.arrays.Reset(r)
	d.previous = d.previous[:0]
	d.prefixes = d.prefixes[:0]
}

func (d *ByteArrayDecoder) DecodeByteArray(data *encoding.ByteArrayList) (int, error) {
	return d.decode(data.Cap()-data.Len(), func(n int) ([]byte, error) { return data.PushSize(n), nil })
}

func (d *ByteArrayDecoder) DecodeFixedLenByteArray(size int, data []byte) (int, error) {
	if size <= 0 {
		return 0, fmt.Errorf("DELTA_BYTE_ARRAY: %w: size of decoded FIXED_LEN_BYTE_ARRAY must be positive", encoding.ErrInvalidArgument)
	}

	i := 0
	return d.decode(len(data)/size, func(n int) ([]byte, error) {
		if n != size {
			return nil, fmt.Errorf("decoding fixed length byte array of size %d but a value of length %d was found", size, n)
		}
		v := data[i : i+n]
		i += n
		return v, nil
	})
}

func (d *ByteArrayDecoder) decode(limit int, push func(int) ([]byte, error)) (int, error) {
	if d.arrays.index < 0 {
		if err := d.decodePrefixes(); err != nil {
			return 0, fmt.Errorf("DELTA_BYTE_ARRAY: decoding prefix lengths: %w", err)
		}
		if err := d.arrays.decodeLengths(); err != nil {
			return 0, fmt.Errorf("DELTA_BYTE_ARRAY: decoding byte array lengths: %w", err)
		}
	}

	if d.arrays.index == len(d.arrays.lengths) {
		return 0, io.EOF
	}

	decoded := 0
	for d.arrays.index < len(d.arrays.lengths) && decoded < limit {
		prefixLength := len(d.previous)
		suffixLength := int(d.arrays.lengths[d.arrays.index])
		length := prefixLength + suffixLength

		value, err := push(length)
		if err != nil {
			return decoded, fmt.Errorf("DELTA_BYTE_ARRAY: %w", err)
		}

		copy(value, d.previous[:prefixLength])
		if err := d.arrays.readFull(value[prefixLength:]); err != nil {
			return decoded, fmt.Errorf("DELTA_BYTE_ARRAY: decoding byte array at index %d/%d: %w", d.arrays.index, len(d.arrays.lengths), err)
		}

		if i := d.arrays.index + 1; i < len(d.prefixes) {
			j := int(d.prefixes[i])
			k := len(value)
			if j > k {
				return decoded, fmt.Errorf("DELTA_BYTE_ARRAY: next prefix is longer than the last decoded byte array (%d>%d)", j, k)
			}
			d.previous = append(d.previous[:0], value[:j]...)
		}

		decoded++
		d.arrays.index++
	}

	return decoded, nil
}

func (d *ByteArrayDecoder) decodePrefixes() (err error) {
	d.prefixes, err = appendDecodeInt32(&d.deltas, d.prefixes[:0])
	return err
}
