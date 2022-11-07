package common

import (
	"hash"
	"hash/fnv"
)

// This file contains types that need to be referenced by both the ./encoding and ./encoding/vX packages.
// It primarily exists here to break dependency loops.

// ID in TempoDB
type ID []byte

// IDMap is a helper for recording and checking for IDs. Not safe for concurrent use.
type IDMap struct {
	m map[uint64]struct{}
	h hash.Hash64
}

func NewIDMap() *IDMap {
	return &IDMap{
		m: map[uint64]struct{}{},
		h: fnv.New64(),
	}
}

// tokenForID returns a token for use in a hash map given a span id and span kind
// buffer must be a 4 byte slice and is reused for writing the span kind to the hashing function
// kind is used along with the actual id b/c in zipkin traces span id is not guaranteed to be unique
// as it is shared between client and server spans.
func (m *IDMap) tokenFor(id ID) uint64 {
	m.h.Reset()
	_, _ = m.h.Write(id)
	return m.h.Sum64()
}

func (m *IDMap) Set(id ID) {
	m.m[m.tokenFor(id)] = struct{}{}
}

func (m *IDMap) Has(id ID) bool {
	_, ok := m.m[m.tokenFor(id)]
	return ok
}
