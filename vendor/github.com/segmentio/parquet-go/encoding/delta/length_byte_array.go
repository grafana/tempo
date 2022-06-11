package delta

import (
	"github.com/segmentio/parquet-go/encoding"
	"github.com/segmentio/parquet-go/encoding/plain"
	"github.com/segmentio/parquet-go/format"
)

type LengthByteArrayEncoding struct {
	encoding.NotSupported
}

func (e *LengthByteArrayEncoding) String() string {
	return "DELTA_LENGTH_BYTE_ARRAY"
}

func (e *LengthByteArrayEncoding) Encoding() format.Encoding {
	return format.DeltaLengthByteArray
}

func (e *LengthByteArrayEncoding) EncodeByteArray(dst, src []byte) ([]byte, error) {
	dst, err := e.encodeByteArray(dst[:0], src)
	if err != nil {
		err = encoding.Error(e, err)
	}
	return dst, err
}

func (e *LengthByteArrayEncoding) encodeByteArray(dst, src []byte) ([]byte, error) {
	length := getInt32Buffer()
	defer putInt32Buffer(length)

	totalSize := 0

	for i := 0; i < len(src); {
		r := len(src) - i
		if r < plain.ByteArrayLengthSize {
			return dst, plain.ErrTooShort(r)
		}
		n := plain.ByteArrayLength(src[i:])
		i += plain.ByteArrayLengthSize
		r -= plain.ByteArrayLengthSize
		if n > r {
			return dst, plain.ErrTooShort(n)
		}
		if n > plain.MaxByteArrayLength {
			return dst, plain.ErrTooLarge(n)
		}
		length.values = append(length.values, int32(n))
		totalSize += n
		i += n
	}

	dst = encodeInt32(dst, length.values)
	dst = resize(dst, len(dst)+totalSize)

	b := dst[len(dst)-totalSize:]
	i := plain.ByteArrayLengthSize
	j := 0

	for _, n := range length.values {
		j += copy(b[j:], src[i:i+int(n)])
		i += plain.ByteArrayLengthSize
		i += int(n)
	}

	return dst, nil
}

func (e *LengthByteArrayEncoding) DecodeByteArray(dst, src []byte) ([]byte, error) {
	return e.decodeByteArray(dst[:0], src)
}

func (e *LengthByteArrayEncoding) decodeByteArray(dst, src []byte) ([]byte, error) {
	length := getInt32Buffer()
	defer putInt32Buffer(length)

	var err error
	src, err = length.decode(src)
	if err != nil {
		return dst, err
	}

	for _, n := range length.values {
		if int(n) < 0 {
			return dst, encoding.Errorf(e, "invalid negative value length: %d", n)
		}
		if int(n) > len(src) {
			return dst, encoding.Errorf(e, "value length is larger than the input size: %d > %d", n, len(src))
		}
		dst = plain.AppendByteArray(dst, src[:n])
		src = src[n:]
	}

	return dst, nil
}
