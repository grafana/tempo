package common

import (
	"bytes"
	"hash"
	"hash/fnv"
	"sort"
)

// This file contains types that need to be referenced by both the ./encoding and ./encoding/vX packages.
// It primarily exists here to break dependency loops.

// ID in TempoDB
type ID []byte

type idMapEntry[T any] struct {
	id    ID
	entry T
}

// IDMap is a helper for recording and checking for IDs. Not safe for concurrent use.
type IDMap[T any] struct {
	m map[uint64]idMapEntry[T]
	h hash.Hash64
}

func NewIDMap[T any]() *IDMap[T] {
	return &IDMap[T]{
		m: map[uint64]idMapEntry[T]{},
		h: fnv.New64(),
	}
}

// tokenForID returns a token for use in a hash map given a span id and span kind
// buffer must be a 4 byte slice and is reused for writing the span kind to the hashing function
// kind is used along with the actual id b/c in zipkin traces span id is not guaranteed to be unique
// as it is shared between client and server spans.
func (m *IDMap[T]) tokenFor(id ID) uint64 {
	m.h.Reset()
	_, _ = m.h.Write(id)
	return m.h.Sum64()
}

func (m *IDMap[T]) Set(id ID, val T) {
	m.m[m.tokenFor(id)] = idMapEntry[T]{id, val}
}

func (m *IDMap[T]) Has(id ID) bool {
	_, ok := m.m[m.tokenFor(id)]
	return ok
}

func (m *IDMap[T]) Get(id ID) (T, bool) {
	v, ok := m.m[m.tokenFor(id)]
	return v.entry, ok
}

func (m *IDMap[T]) Len() int {
	return len(m.m)
}

func (m *IDMap[T]) ValuesSortedByID() []T {
	// Copy and sort entries by ID
	entries := make([]idMapEntry[T], 0, len(m.m))
	for _, e := range m.m {
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool {
		return bytes.Compare(entries[i].id, entries[j].id) == -1
	})

	// Copy sorted values
	values := make([]T, 0, len(entries))
	for _, e := range entries {
		values = append(values, e.entry)
	}

	return values
}
