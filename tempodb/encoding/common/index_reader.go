package common

import (
	"bytes"
	"sort"
)

// Records is a slice of *Record
type Records []*Record

// At implements IndexReader
func (r Records) At(i int) *Record {
	if i < 0 || i >= len(r) {
		return nil
	}

	return r[i]
}

// Find implements IndexReader
func (r Records) Find(id ID) (*Record, int) {
	i := sort.Search(len(r), func(idx int) bool {
		return bytes.Compare(r[idx].ID, id) >= 0
	})

	if i < 0 || i >= len(r) {
		return nil, -1
	}

	return r[i], i
}
