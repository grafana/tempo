package vparquet

import (
	"bytes"
	"context"
	"io"

	"github.com/pkg/errors"
	"github.com/segmentio/parquet-go"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

type combineFn func([]parquet.Row) (parquet.Row, error)

type MultiBlockIterator struct {
	bookmarks []*bookmark
	combine   combineFn
}

var _ RawIterator = (*MultiBlockIterator)(nil)

func newMultiblockIterator(bookmarks []*bookmark, combine combineFn) *MultiBlockIterator {
	return &MultiBlockIterator{
		bookmarks: bookmarks,
		combine:   combine,
	}
}

func (m *MultiBlockIterator) Next(ctx context.Context) (common.ID, parquet.Row, error) {

	if m.done(ctx) {
		return nil, nil, io.EOF
	}

	var (
		lowestID        common.ID
		lowestObjects   []parquet.Row
		lowestBookmarks []*bookmark
	)

	// find lowest ID of the new object
	for _, b := range m.bookmarks {
		id, currentObject, err := b.current(ctx)
		if err != nil && err != io.EOF {
			return nil, nil, err
		}
		if currentObject == nil {
			continue
		}

		comparison := bytes.Compare(id, lowestID)

		if comparison == 0 {
			lowestObjects = append(lowestObjects, currentObject)
			lowestBookmarks = append(lowestBookmarks, b)
		} else if len(lowestID) == 0 || comparison == -1 {
			lowestID = id

			// reset and reuse
			lowestObjects = lowestObjects[:0]
			lowestBookmarks = lowestBookmarks[:0]

			lowestObjects = append(lowestObjects, currentObject)
			lowestBookmarks = append(lowestBookmarks, b)
		}
	}

	lowestObject, err := m.combine(lowestObjects)
	if err != nil {
		return nil, nil, errors.Wrap(err, "combining")
	}

	for _, b := range lowestBookmarks {
		b.clear()
	}

	return lowestID, lowestObject, nil
}

func (m *MultiBlockIterator) Close() {
	for _, b := range m.bookmarks {
		b.close()
	}
}

func (m *MultiBlockIterator) done(ctx context.Context) bool {
	for _, b := range m.bookmarks {
		if !b.done(ctx) {
			return false
		}
	}
	return true
}
