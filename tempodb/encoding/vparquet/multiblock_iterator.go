package vparquet

import (
	"context"
	"io"
	"strings"
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
	allDone := func() bool {
		for _, b := range m.bookmarks {
			if !b.done(ctx) {
				return false
			}
		}
		return true
	}

	// check if all bookmarks are done
	if allDone() {
		return nil, io.EOF
	}

	var lowestID string
	var lowestObjects []*Trace
	var lowestBookmarks []*bookmark

	// find lowest ID of the new object
	for _, b := range m.bookmarks {
		currentObject, err := b.current(ctx)
		if err != nil {
			return nil, err
		}
		if currentObject == nil {
			continue
		}

		comparison := strings.Compare(currentObject.TraceID, lowestID)

		if comparison == 0 {
			lowestObjects = append(lowestObjects, currentObject)
			lowestBookmarks = append(lowestBookmarks, b)
		} else if len(lowestID) == 0 || comparison == -1 {
			lowestID = currentObject.TraceID
			lowestObjects = []*Trace{currentObject}
			lowestBookmarks = []*bookmark{b}
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
