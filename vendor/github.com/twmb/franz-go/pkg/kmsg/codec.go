package kmsg

import "github.com/twmb/franz-go/pkg/kmsg/internal/kbin"

// Generated code reads and appends fixed size fields with these. A call is
// much smaller than the inlined kbin function at every field.

//go:noinline
func readBool(b *kbin.Reader) bool { return b.Bool() }

//go:noinline
func readInt8(b *kbin.Reader) int8 { return b.Int8() }

//go:noinline
func readInt16(b *kbin.Reader) int16 { return b.Int16() }

//go:noinline
func readUint16(b *kbin.Reader) uint16 { return b.Uint16() }

//go:noinline
func readInt32(b *kbin.Reader) int32 { return b.Int32() }

//go:noinline
func readInt64(b *kbin.Reader) int64 { return b.Int64() }

//go:noinline
func readUuid(b *kbin.Reader) [16]byte { return b.Uuid() }

//go:noinline
func appendBool(dst []byte, v bool) []byte { return kbin.AppendBool(dst, v) }

//go:noinline
func appendInt8(dst []byte, v int8) []byte { return kbin.AppendInt8(dst, v) }

//go:noinline
func appendInt16(dst []byte, v int16) []byte { return kbin.AppendInt16(dst, v) }

//go:noinline
func appendUint16(dst []byte, v uint16) []byte { return kbin.AppendUint16(dst, v) }

//go:noinline
func appendInt32(dst []byte, v int32) []byte { return kbin.AppendInt32(dst, v) }

//go:noinline
func appendInt64(dst []byte, v int64) []byte { return kbin.AppendInt64(dst, v) }

//go:noinline
func appendUuid(dst []byte, v [16]byte) []byte { return kbin.AppendUuid(dst, v) }

// Generated code reads and appends strings, bytes, and array lengths that can
// be compact or unsafe with the functions below, rather than choosing between
// kbin functions at every field.
//
// Most compact lengths are one byte. The read functions decode that byte
// themselves, since Reader.Uvarint is not inlined. The read functions for
// strings and bytes store into dst; see writeDecodeInto in the generator.

func readString(b *kbin.Reader, dst *string, flexible, unsafe bool) {
	var l int
	if flexible {
		if src := b.Src; len(src) > 0 && src[0] < 0x80 {
			b.Src = src[1:]
			l = int(src[0]) - 1
		} else {
			l = int(b.Uvarint()) - 1
		}
	} else {
		l = int(b.Int16())
	}
	if unsafe {
		*dst = kbin.UnsafeString(b.Span(l))
	} else {
		*dst = string(b.Span(l))
	}
}

func readNullableString(b *kbin.Reader, dst **string, flexible, unsafe bool) {
	var l int
	if flexible {
		if src := b.Src; len(src) > 0 && src[0] < 0x80 {
			b.Src = src[1:]
			l = int(src[0]) - 1
		} else {
			l = int(b.Uvarint()) - 1
		}
	} else {
		l = int(b.Int16())
	}
	if l < 0 {
		*dst = nil
		return
	}
	var s string
	if unsafe {
		s = kbin.UnsafeString(b.Span(l))
	} else {
		s = string(b.Span(l))
	}
	*dst = &s
}

func readVarintString(b *kbin.Reader, dst *string, unsafe bool) {
	if unsafe {
		*dst = kbin.UnsafeString(b.VarintBytes())
	} else {
		*dst = string(b.VarintBytes())
	}
}

func readVarintBytes(b *kbin.Reader, dst *[]byte) { *dst = b.VarintBytes() }

func readBytes(b *kbin.Reader, dst *[]byte, flexible bool) {
	var l int
	if flexible {
		if src := b.Src; len(src) > 0 && src[0] < 0x80 {
			b.Src = src[1:]
			l = int(src[0]) - 1
		} else {
			l = int(b.Uvarint()) - 1
		}
	} else {
		l = int(b.Int32())
	}
	if l == -1 { // see kbin.Reader.Bytes
		*dst = []byte{}
		return
	}
	*dst = b.Span(l)
}

func readNullableBytes(b *kbin.Reader, dst *[]byte, flexible bool) {
	var l int
	if flexible {
		if src := b.Src; len(src) > 0 && src[0] < 0x80 {
			b.Src = src[1:]
			l = int(src[0]) - 1
		} else {
			l = int(b.Uvarint()) - 1
		}
	} else {
		l = int(b.Int32())
	}
	if l < 0 {
		*dst = nil
		return
	}
	*dst = b.Span(l)
}

func appendString(dst []byte, s string, flexible bool) []byte {
	if flexible {
		dst = kbin.AppendUvarint(dst, 1+uint32(len(s)))
	} else {
		dst = kbin.AppendInt16(dst, int16(len(s)))
	}
	return append(dst, s...)
}

func appendNullableString(dst []byte, s *string, flexible bool) []byte {
	if s == nil {
		if flexible {
			return append(dst, 0)
		}
		return kbin.AppendInt16(dst, -1)
	}
	if flexible {
		dst = kbin.AppendUvarint(dst, 1+uint32(len(*s)))
	} else {
		dst = kbin.AppendInt16(dst, int16(len(*s)))
	}
	return append(dst, *s...)
}

func appendBytes(dst, b []byte, flexible bool) []byte {
	if flexible {
		dst = kbin.AppendUvarint(dst, 1+uint32(len(b)))
	} else {
		dst = kbin.AppendInt32(dst, int32(len(b)))
	}
	return append(dst, b...)
}

func appendNullableBytes(dst, b []byte, flexible bool) []byte {
	if b == nil {
		if flexible {
			return append(dst, 0)
		}
		return kbin.AppendInt32(dst, -1)
	}
	if flexible {
		dst = kbin.AppendUvarint(dst, 1+uint32(len(b)))
	} else {
		dst = kbin.AppendInt32(dst, int32(len(b)))
	}
	return append(dst, b...)
}

func readArrayLen(b *kbin.Reader, flexible bool) int32 {
	if flexible {
		if src := b.Src; len(src) > 0 && src[0] < 0x80 && int(src[0]) <= len(src) {
			b.Src = src[1:]
			return int32(src[0]) - 1
		}
		return b.CompactArrayLen()
	}
	return b.ArrayLen()
}

func appendArrayLen(dst []byte, n int, flexible bool) []byte {
	if flexible {
		return kbin.AppendCompactArrayLen(dst, n)
	}
	return kbin.AppendArrayLen(dst, n)
}

func appendNullableArrayLen(dst []byte, n int, isNil, flexible bool) []byte {
	if flexible {
		return kbin.AppendCompactNullableArrayLen(dst, n, isNil)
	}
	return kbin.AppendNullableArrayLen(dst, n, isNil)
}
