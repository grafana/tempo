package delta

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/segmentio/parquet-go/encoding"
	"github.com/segmentio/parquet-go/encoding/plain"
	"github.com/segmentio/parquet-go/format"
)

const (
	maxLinearSearchPrefixLength = 64 // arbitrary
)

type ByteArrayEncoding struct {
	encoding.NotSupported
}

func (e *ByteArrayEncoding) String() string {
	return "DELTA_BYTE_ARRAY"
}

func (e *ByteArrayEncoding) Encoding() format.Encoding {
	return format.DeltaByteArray
}

func (e *ByteArrayEncoding) EncodeByteArray(dst, src []byte) ([]byte, error) {
	prefix := getInt32Buffer()
	defer putInt32Buffer(prefix)

	length := getInt32Buffer()
	defer putInt32Buffer(length)

	totalSize := 0
	lastValue := ([]byte)(nil)

	for i := 0; i < len(src); {
		r := len(src) - i
		if r < plain.ByteArrayLengthSize {
			return dst[:0], plain.ErrTooShort(r)
		}
		n := plain.ByteArrayLength(src[i:])
		i += plain.ByteArrayLengthSize
		r -= plain.ByteArrayLengthSize
		if n > r {
			return dst[:0], plain.ErrTooShort(n)
		}
		if n > plain.MaxByteArrayLength {
			return dst[:0], plain.ErrTooLarge(n)
		}
		v := src[i : i+n : i+n]
		p := 0

		if len(v) <= maxLinearSearchPrefixLength {
			p = linearSearchPrefixLength(lastValue, v)
		} else {
			p = binarySearchPrefixLength(lastValue, v)
		}

		prefix.values = append(prefix.values, int32(p))
		length.values = append(length.values, int32(n-p))
		lastValue = v
		totalSize += n - p
		i += n
	}

	dst = dst[:0]
	dst = encodeInt32(dst, prefix.values)
	dst = encodeInt32(dst, length.values)
	dst = resize(dst, len(dst)+totalSize)

	b := dst[len(dst)-totalSize:]
	i := plain.ByteArrayLengthSize
	j := 0

	_ = length.values[:len(prefix.values)]

	for k, p := range prefix.values {
		n := p + length.values[k]
		j += copy(b[j:], src[i+int(p):i+int(n)])
		i += plain.ByteArrayLengthSize
		i += int(n)
	}

	return dst, nil
}

func (e *ByteArrayEncoding) EncodeFixedLenByteArray(dst, src []byte, size int) ([]byte, error) {
	// The parquet specs say that this encoding is only supported for BYTE_ARRAY
	// values, but the reference Java implementation appears to support
	// FIXED_LEN_BYTE_ARRAY as well:
	// https://github.com/apache/parquet-mr/blob/5608695f5777de1eb0899d9075ec9411cfdf31d3/parquet-column/src/main/java/org/apache/parquet/column/Encoding.java#L211
	if size < 0 || size > encoding.MaxFixedLenByteArraySize {
		return dst[:0], encoding.Error(e, encoding.ErrInvalidArgument)
	}
	if (len(src) % size) != 0 {
		return dst[:0], encoding.ErrEncodeInvalidInputSize(e, "FIXED_LEN_BYTE_ARRAY", len(src))
	}

	prefix := getInt32Buffer()
	defer putInt32Buffer(prefix)

	length := getInt32Buffer()
	defer putInt32Buffer(length)

	totalSize := 0
	lastValue := ([]byte)(nil)

	for i := size; i <= len(src); i += size {
		v := src[i-size : i : i]
		p := linearSearchPrefixLength(lastValue, v)
		n := size - p
		prefix.values = append(prefix.values, int32(p))
		length.values = append(length.values, int32(n))
		lastValue = v
		totalSize += n
	}

	dst = dst[:0]
	dst = encodeInt32(dst, prefix.values)
	dst = encodeInt32(dst, length.values)
	dst = resize(dst, len(dst)+totalSize)

	b := dst[len(dst)-totalSize:]
	i := 0
	j := 0

	for _, p := range prefix.values {
		j += copy(b[j:], src[i+int(p):i+size])
		i += size
	}

	return dst, nil
}

func (e *ByteArrayEncoding) DecodeByteArray(dst, src []byte) ([]byte, error) {
	dst = dst[:0]

	prefix := getInt32Buffer()
	defer putInt32Buffer(prefix)

	suffix := getInt32Buffer()
	defer putInt32Buffer(suffix)

	var err error
	src, err = prefix.decode(src)
	if err != nil {
		return dst, encoding.Errorf(e, "decoding prefix lengths: %w", err)
	}
	src, err = suffix.decode(src)
	if err != nil {
		return dst, encoding.Errorf(e, "decoding suffix lengths: %w", err)
	}
	if len(prefix.values) != len(suffix.values) {
		return dst, encoding.Error(e, errPrefixAndSuffixLengthMismatch(len(prefix.values), len(suffix.values)))
	}
	return decodeByteArray(dst, src, prefix.values, suffix.values)
}

func (e *ByteArrayEncoding) DecodeFixedLenByteArray(dst, src []byte, size int) ([]byte, error) {
	dst = dst[:0]

	if size < 0 || size > encoding.MaxFixedLenByteArraySize {
		return dst, encoding.Error(e, encoding.ErrInvalidArgument)
	}

	prefix := getInt32Buffer()
	defer putInt32Buffer(prefix)

	suffix := getInt32Buffer()
	defer putInt32Buffer(suffix)

	var err error
	src, err = prefix.decode(src)
	if err != nil {
		return dst, fmt.Errorf("decoding prefix lengths: %w", err)
	}
	src, err = suffix.decode(src)
	if err != nil {
		return dst, fmt.Errorf("decoding suffix lengths: %w", err)
	}
	if len(prefix.values) != len(suffix.values) {
		return dst, errPrefixAndSuffixLengthMismatch(len(prefix.values), len(suffix.values))
	}
	return decodeFixedLenByteArray(dst, src, size, prefix.values, suffix.values)
}

func linearSearchPrefixLength(base, data []byte) (n int) {
	for n < len(base) && n < len(data) && base[n] == data[n] {
		n++
	}
	return n
}

func binarySearchPrefixLength(base, data []byte) int {
	n := len(base)
	if n > len(data) {
		n = len(data)
	}
	return sort.Search(n, func(i int) bool {
		return !bytes.Equal(base[:i+1], data[:i+1])
	})
}
