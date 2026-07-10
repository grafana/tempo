// Package deltauuid implements the compact wire encoding for a sequence of
// 16-byte UUIDs used by the bloom gateway's query response to keep its
// rejected-block-UUID set small (DESIGN.md § Protocol: "sorted delta-encoded
// block UUIDs"; IMPLEMENTATION_PLAN.md §0 "Delta-UUID wire scheme").
//
// Encoding: for each UUID in turn, one length byte (0-16) followed by that
// many big-endian bytes of the *gap* from the previous UUID — the unsigned
// 128-bit difference, computed with 2^128 wraparound rather than a signed
// subtraction, stored in its minimal byte-width form (no leading zero
// bytes). The first entry's "previous" is the all-zero UUID.
//
// This package is standalone (no bloomgateway import) so a future
// query-frontend decoder can depend on it directly.
package deltauuid

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/bits"
)

// ErrInvalidLengthByte is returned by DecodeSortedDeltas when an entry's
// length byte is outside the valid 0-16 range — a gap between two 16-byte
// values can never need more than 16 bytes.
var ErrInvalidLengthByte = errors.New("deltauuid: length byte out of range (must be 0-16)")

// ErrTruncatedInput is returned by DecodeSortedDeltas when the input ends
// before an entry's length byte says it should.
var ErrTruncatedInput = errors.New("deltauuid: truncated input")

// EncodeSortedDeltas encodes sorted as: for each entry, one length byte
// (0-16) followed by that many big-endian bytes of the gap from the
// previous entry (the all-zero UUID for the first entry). DecodeSortedDeltas
// is the exact inverse.
//
// sorted is expected to be ascending and deduplicated — that is what makes
// the encoding compact (small gaps). EncodeSortedDeltas does not validate
// this precondition and never errors on it: a gap is always the *unsigned*
// 128-bit difference computed with 2^128 wraparound, which always fits in
// 16 bytes regardless of the relative order of two consecutive entries. So
// out-of-order or duplicate input still round-trips exactly through
// DecodeSortedDeltas (in its original, unsorted order); it just costs up to
// one byte per entry more than emitting the raw 16-byte UUID (the length
// byte), never more — the same pathological-case bound sorted input has in
// the worst case.
func EncodeSortedDeltas(sorted [][16]byte) []byte {
	// Worst case is 17 B/entry (1 length byte + all 16 gap bytes);
	// preallocating it guarantees this loop never reallocates, at the cost
	// of some transient over-allocation when gaps are small.
	out := make([]byte, 0, len(sorted)*17)

	var prev [16]byte // previous entry; starts at the all-zero UUID
	for _, cur := range sorted {
		gapHi, gapLo := sub128(cur, prev)
		out = appendGap(out, gapHi, gapLo)
		prev = cur
	}
	return out
}

// DecodeSortedDeltas is the exact inverse of EncodeSortedDeltas. It returns
// an error (wrapping ErrInvalidLengthByte or ErrTruncatedInput) on malformed
// input rather than panicking.
func DecodeSortedDeltas(b []byte) ([][16]byte, error) {
	// The densest possible encoding is 1 B/entry (an all-zero gap), so
	// len(b) is a safe upper bound for the entry count; a smaller hint
	// avoids over-allocating for the common case where gaps are larger.
	out := make([][16]byte, 0, len(b)/2+1)

	var prev [16]byte
	i := 0
	for i < len(b) {
		length := int(b[i])
		i++
		if length > 16 {
			return nil, fmt.Errorf("%w: got %d at offset %d", ErrInvalidLengthByte, length, i-1)
		}
		if len(b)-i < length {
			return nil, fmt.Errorf("%w: entry at offset %d needs %d bytes, only %d remain", ErrTruncatedInput, i-1, length, len(b)-i)
		}

		var gap [16]byte
		copy(gap[16-length:], b[i:i+length])
		i += length

		cur := add128(prev, gap)
		out = append(out, cur)
		prev = cur
	}
	return out, nil
}

// sub128 returns (a - b) mod 2^128, as the high and low 64 bits of the
// big-endian 128-bit result.
func sub128(a, b [16]byte) (hi, lo uint64) {
	ahi, alo := binary.BigEndian.Uint64(a[:8]), binary.BigEndian.Uint64(a[8:])
	bhi, blo := binary.BigEndian.Uint64(b[:8]), binary.BigEndian.Uint64(b[8:])
	lo, borrow := bits.Sub64(alo, blo, 0)
	hi, _ = bits.Sub64(ahi, bhi, borrow)
	return hi, lo
}

// add128 returns (a + gap) mod 2^128 as a big-endian [16]byte — the inverse
// of sub128.
func add128(a, gap [16]byte) [16]byte {
	ahi, alo := binary.BigEndian.Uint64(a[:8]), binary.BigEndian.Uint64(a[8:])
	ghi, glo := binary.BigEndian.Uint64(gap[:8]), binary.BigEndian.Uint64(gap[8:])
	lo, carry := bits.Add64(alo, glo, 0)
	hi, _ := bits.Add64(ahi, ghi, carry)

	var out [16]byte
	binary.BigEndian.PutUint64(out[:8], hi)
	binary.BigEndian.PutUint64(out[8:], lo)
	return out
}

// appendGap appends one entry (length byte + minimal big-endian bytes) for
// the 128-bit value (hi, lo) to out.
func appendGap(out []byte, hi, lo uint64) []byte {
	var buf [16]byte
	binary.BigEndian.PutUint64(buf[:8], hi)
	binary.BigEndian.PutUint64(buf[8:], lo)

	length := gapLength(hi, lo)
	out = append(out, byte(length))
	return append(out, buf[16-length:]...)
}

// gapLength returns the minimal number of big-endian bytes (0-16) needed to
// represent the 128-bit value (hi, lo) with no leading zero byte.
func gapLength(hi, lo uint64) int {
	switch {
	case hi != 0:
		// Every byte of lo is significant once hi is nonzero — only hi's own
		// leading zero bytes can be stripped.
		return 16 - bits.LeadingZeros64(hi)/8
	case lo != 0:
		return 8 - bits.LeadingZeros64(lo)/8
	default:
		return 0
	}
}
