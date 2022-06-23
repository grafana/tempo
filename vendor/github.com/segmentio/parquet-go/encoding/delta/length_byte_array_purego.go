//go:build purego || !amd64

package delta

import (
	"github.com/segmentio/parquet-go/encoding/plain"
)

func decodeLengthByteArray(dst, src []byte, lengths []int32) ([]byte, error) {
	for i := range lengths {
		n := int(lengths[i])
		if n < 0 {
			return dst, errInvalidNegativeValueLength(n)
		}
		if n > len(src) {
			return dst, errValueLengthOutOfBounds(n, len(src))
		}
		dst = plain.AppendByteArray(dst, src[:n])
		src = src[n:]
	}
	return dst, nil
}
