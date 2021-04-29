package common

import (
	"bytes"
	"context"
	"sort"
)

// Records is a slice of *Record
type Records []Record

// At implements IndexReader
func (r Records) At(_ context.Context, i int) (*Record, error) {
	if i < 0 || i >= len(r) {
		return nil, nil
	}

	return &r[i], nil
}

// Find implements IndexReader
func (r Records) Find(_ context.Context, id ID) (*Record, int, error) {
	i := sort.Search(len(r), func(idx int) bool {
		return bytes.Compare(r[idx].ID, id) >= 0
	})

	if i < 0 || i >= len(r) {
		return nil, -1, nil
	}

	return &r[i], i, nil
}

func (r Records) Len() int {
	return len(r)
}
