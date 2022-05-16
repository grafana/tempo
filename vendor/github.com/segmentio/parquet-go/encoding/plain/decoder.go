package plain

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/segmentio/parquet-go/deprecated"
	"github.com/segmentio/parquet-go/encoding"
	"github.com/segmentio/parquet-go/encoding/rle"
	"github.com/segmentio/parquet-go/internal/bits"
)

type Decoder struct {
	encoding.NotSupportedDecoder
	reader io.Reader
	buffer [4]byte
	rle    *rle.Decoder
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{reader: r}
}

func (d *Decoder) Reset(r io.Reader) {
	d.reader = r

	if d.rle != nil {
		d.rle.Reset(r)
	}
}

func (d *Decoder) DecodeBoolean(data []bool) (int, error) {
	if d.rle == nil {
		d.rle = rle.NewDecoder(d.reader)
	}
	return d.rle.DecodeBoolean(data)
}

func (d *Decoder) DecodeInt32(data []int32) (int, error) {
	return readFull(d.reader, 4, bits.Int32ToBytes(data))
}

func (d *Decoder) DecodeInt64(data []int64) (int, error) {
	return readFull(d.reader, 8, bits.Int64ToBytes(data))
}

func (d *Decoder) DecodeInt96(data []deprecated.Int96) (int, error) {
	return readFull(d.reader, 12, deprecated.Int96ToBytes(data))
}

func (d *Decoder) DecodeFloat(data []float32) (int, error) {
	return readFull(d.reader, 4, bits.Float32ToBytes(data))
}

func (d *Decoder) DecodeDouble(data []float64) (int, error) {
	return readFull(d.reader, 8, bits.Float64ToBytes(data))
}

func (d *Decoder) DecodeByteArray(data *encoding.ByteArrayList) (n int, err error) {
	n = data.Len()

	for data.Len() < data.Cap() {
		if _, err = io.ReadFull(d.reader, d.buffer[:4]); err != nil {
			break
		}
		if value := data.PushSize(int(binary.LittleEndian.Uint32(d.buffer[:4]))); len(value) > 0 {
			if _, err = io.ReadFull(d.reader, value); err != nil {
				if err == io.EOF {
					err = io.ErrUnexpectedEOF
				}
				break
			}
		}
	}

	return data.Len() - n, err
}

func (d *Decoder) DecodeFixedLenByteArray(size int, data []byte) (int, error) {
	if size <= 0 {
		return 0, fmt.Errorf("PLAIN: %w: size of decoded FIXED_LEN_BYTE_ARRAY must be positive", encoding.ErrInvalidArgument)
	}

	if (len(data) % size) != 0 {
		return 0, fmt.Errorf("PLAIN: %w: length of decoded FIXED_LEN_BYTE_ARRAY must be a multiple of its size: size=%d length=%d", encoding.ErrInvalidArgument, size, len(data))
	}

	return readFull(d.reader, size, data)
}

func (d *Decoder) SetBitWidth(bitWidth int) {}

func readFull(r io.Reader, scale int, data []byte) (int, error) {
	n, err := io.ReadFull(r, data)
	if err == io.ErrUnexpectedEOF && (n%scale) == 0 {
		err = io.EOF
	}
	return n / scale, err
}

func prepend(dst, src []byte) (ret []byte) {
	if (cap(dst) - len(dst)) < len(src) {
		ret = make([]byte, len(src)+len(dst))
	} else {
		ret = dst[:len(src)+len(dst)]
	}
	copy(ret[len(src):], dst)
	copy(ret, src)
	return ret
}
