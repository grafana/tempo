package vparquet

import (
	"bytes"
	"context"
	"io"
)

type MultiBlockIterator struct {
	bookmarks []*bookmark
}

func newMultiblockIterator(bookmarks []*bookmark) *MultiBlockIterator {
	return &MultiBlockIterator{
		bookmarks: bookmarks,
	}
}

func (m *MultiBlockIterator) Next(ctx context.Context) (*Trace, error) {

	if m.done(ctx) {
		return nil, io.EOF
	}

	var lowestID []byte
	var lowestObjects []*Trace
	var lowestBookmarks []*bookmark

	// find lowest ID of the new object
	for _, b := range m.bookmarks {
		currentObject, err := b.current(ctx)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if currentObject == nil {
			continue
		}

		comparison := bytes.Compare(currentObject.TraceID, lowestID)

		if comparison == 0 {
			lowestObjects = append(lowestObjects, currentObject)
			lowestBookmarks = append(lowestBookmarks, b)
		} else if len(lowestID) == 0 || comparison == -1 {
			lowestID = currentObject.TraceID

			// reset and reuse
			lowestObjects = lowestObjects[:0]
			lowestBookmarks = lowestBookmarks[:0]

			lowestObjects = append(lowestObjects, currentObject)
			lowestBookmarks = append(lowestBookmarks, b)
		}
	}

	lowestObject := CombineTraces(lowestObjects...)

	for _, b := range lowestBookmarks {
		b.clear()
	}

	return lowestObject, nil
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
