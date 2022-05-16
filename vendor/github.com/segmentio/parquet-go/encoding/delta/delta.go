package delta

import (
	"io"

	"github.com/segmentio/parquet-go/encoding"
)

const (
	defaultBufferSize = 4096
)

func appendDecodeInt32(d encoding.Decoder, data []int32) ([]int32, error) {
	for {
		if len(data) == cap(data) {
			if cap(data) == 0 {
				data = make([]int32, 0, blockSize32)
			} else {
				newData := make([]int32, len(data), 2*cap(data))
				copy(newData, data)
				data = newData
			}
		}

		n, err := d.DecodeInt32(data[len(data):cap(data)])
		data = data[:len(data)+n]
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return data, err
		}
	}
}
