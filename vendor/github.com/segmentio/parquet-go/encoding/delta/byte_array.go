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
	err := e.decode(src, func(prefix, suffix []byte) ([]byte, error) {
		n := len(prefix) + len(suffix)
		b := [4]byte{}
		plain.PutByteArrayLength(b[:], n)
		dst = append(dst, b[:]...)
		i := len(dst)
		dst = append(dst, prefix...)
		dst = append(dst, suffix...)
		return dst[i:len(dst):len(dst)], nil
	})
	if err != nil {
		err = encoding.Error(e, err)
	}
	return dst, err
}

func (e *ByteArrayEncoding) DecodeFixedLenByteArray(dst, src []byte, size int) ([]byte, error) {
	if size < 0 || size > encoding.MaxFixedLenByteArraySize {
		return dst[:0], encoding.Error(e, encoding.ErrInvalidArgument)
	}
	dst = dst[:0]
	err := e.decode(src, func(prefix, suffix []byte) ([]byte, error) {
		n := len(prefix) + len(suffix)
		if n != size {
			return nil, fmt.Errorf("cannot decode value of size %d into fixed-length byte array of size %d", n, size)
		}
		i := len(dst)
		dst = append(dst, prefix...)
		dst = append(dst, suffix...)
		return dst[i:len(dst):len(dst)], nil
	})
	if err != nil {
		err = encoding.Error(e, err)
	}
	return dst, err
}

func (e *ByteArrayEncoding) decode(src []byte, observe func(prefix, suffix []byte) ([]byte, error)) error {
	prefix := getInt32Buffer()
	defer putInt32Buffer(prefix)

	length := getInt32Buffer()
	defer putInt32Buffer(length)

	var err error
	src, err = prefix.decode(src)
	if err != nil {
		return err
	}
	src, err = length.decode(src)
	if err != nil {
		return err
	}
	if len(prefix.values) != len(length.values) {
		return fmt.Errorf("number of prefix and lengths mismatch: %d != %d", len(prefix.values), len(length.values))
	}

	var lastValue []byte
	for i, n := range length.values {
		if int(n) < 0 {
			return fmt.Errorf("invalid negative value length: %d", n)
		}
		if int(n) > len(src) {
			return fmt.Errorf("value length is larger than the input size: %d > %d", n, len(src))
		}

		p := prefix.values[i]
		if int(p) < 0 {
			return fmt.Errorf("invalid negative prefix length: %d", p)
		}
		if int(p) > len(lastValue) {
			return fmt.Errorf("prefix length %d is larger than the last value of size %d", p, len(lastValue))
		}

		prefix := lastValue[:p:p]
		suffix := src[:n:n]
		src = src[n:len(src):len(src)]

		if lastValue, err = observe(prefix, suffix); err != nil {
			return err
		}
	}

	return nil
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
