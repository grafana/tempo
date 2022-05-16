package delta

import (
	"fmt"
	"io"

	"github.com/segmentio/parquet-go/encoding"
)

type LengthByteArrayDecoder struct {
	encoding.NotSupportedDecoder
	binpack BinaryPackedDecoder
	lengths []int32
	index   int
}

func NewLengthByteArrayDecoder(r io.Reader) *LengthByteArrayDecoder {
	d := &LengthByteArrayDecoder{lengths: make([]int32, defaultBufferSize/4)}
	d.Reset(r)
	return d
}

func (d *LengthByteArrayDecoder) Reset(r io.Reader) {
	d.binpack.Reset(r)
	d.lengths = d.lengths[:0]
	d.index = -1
}

func (d *LengthByteArrayDecoder) DecodeByteArray(data *encoding.ByteArrayList) (n int, err error) {
	if d.index < 0 {
		if err := d.decodeLengths(); err != nil {
			return 0, err
		}
	}

	n = data.Len()
	for data.Len() < data.Cap() && d.index < len(d.lengths) {
		value := data.PushSize(int(d.lengths[d.index]))
		_, err := io.ReadFull(d.binpack.reader, value)
		if err != nil {
			err = fmt.Errorf("DELTA_LENGTH_BYTE_ARRAY: decoding byte array at index %d/%d: %w", d.index, len(d.lengths), dontExpectEOF(err))
			break
		}
		d.index++
	}

	if d.index == len(d.lengths) {
		err = io.EOF
	}

	return data.Len() - n, err
}

func (d *LengthByteArrayDecoder) decodeLengths() (err error) {
	d.lengths, err = appendDecodeInt32(&d.binpack, d.lengths[:0])
	if err != nil {
		return err
	}
	d.index = 0
	return nil
}

func (d *LengthByteArrayDecoder) readFull(b []byte) error {
	_, err := io.ReadFull(d.binpack.reader, b)
	return dontExpectEOF(err)
}
