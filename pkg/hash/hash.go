// Package hash centralizes Tempo's hash function choices.
//
// # Choosing between Digest (xxhash) and FNV-1a
//
// Use the Go standard library hash/fnv to hash short inputs (up to 30-40 bytes)
// and when you don't need the avalanche/collision properties of xxhash.
// TokenFor, TokenForTraceID, and HashForTraceID are fnv-1-backed.
//
// Use Digest (xxhash) for larger inputs (larger than 30-40 bytes) or when
// better avalanche/collision properties are needed. Avalanche properties can
// matter for inputs that can have long identical prefixes e.g. TraceQL
// queries or resource properties.
//
// See hash_test.go for comparative benchmarks.
package hash

import (
	"encoding/binary"
	"hash/fnv"

	"github.com/cespare/xxhash/v2"
)

// Digest is a 64-bit hash accumulator backed by xxhash.
//
// The zero value is NOT valid; use New, NewValue, or call Reset before
// the first Write.
type Digest struct {
	xxhash.Digest
}

// New returns an initialized *Digest.
func New() *Digest {
	d := &Digest{}
	d.Reset()
	return d
}

// NewValue returns an initialized Digest value. Use when you want
// to express stack semantics explicitly; note that taking the address of
// the returned value and passing it across function boundaries may still
// cause it to escape to the heap.
func NewValue() Digest {
	var d Digest
	d.Reset()
	return d
}

// WriteUint64 appends n to d as 8 little-endian bytes.
func (d *Digest) WriteUint64(n uint64) {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], n)
	_, _ = d.Write(buf[:])
}

// Sum is intentionally not supported on hash.Digest because the
// xxhash.Digest.Sum signature is a frequent source of bugs: it appends
// 8 bytes to the argument and returns a []byte, not the uint64 most
// callers expect.
//
// Deprecated: Use Sum64 instead. Calling Sum panics.
func (d *Digest) Sum(_ []byte) []byte {
	panic("hash.Digest.Sum is not supported; use Sum64")
}

// Sum64 returns the 64-bit xxhash digest of b with a zero seed.
//
// Prefer this over constructing a Digest when hashing a single byte
// slice. It is significantly cheaper than New().Write(b).Sum64().
func Sum64(b []byte) uint64 { return xxhash.Sum64(b) }

// Sum64String returns the 64-bit xxhash digest of s with a zero seed.
//
// Prefer this over constructing a Digest when hashing a single string.
func Sum64String(s string) uint64 { return xxhash.Sum64String(s) }

// TokenFor generates a token used for finding ingesters from a ring.
// It hashes the userID followed by b with FNV-1 32-bit.
//
// Not suitable for in-memory hashing or deduping because it is only
// 32-bit; the collision rate is about 1 in 8000.
func TokenFor(userID string, b []byte) uint32 {
	h := fnv.New32()
	_, _ = h.Write([]byte(userID))
	_, _ = h.Write(b)
	return h.Sum32()
}

// TokenForTraceID generates a hashed value for a trace id with FNV-1
// 32-bit. Used for bloom lookups.
//
// Do not change because it will break lookups on existing bloom filters.
func TokenForTraceID(b []byte) uint32 {
	h := fnv.New32()
	_, _ = h.Write(b)
	return h.Sum32()
}

// HashForTraceID generates a generic 64-bit hash for a trace ID with
// FNV-1 64-bit, suitable for mapping and deduping.
//
//revive:disable-next-line:exported // Name kept for continuity with the previous util.HashForTraceID location.
func HashForTraceID(tid []byte) uint64 {
	h := fnv.New64()
	_, _ = h.Write(tid)
	return h.Sum64()
}
