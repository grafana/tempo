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

	if m.done(ctx) {
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

		// Left pad with zeroes for consistent comparison
		currentID := padTraceID(currentObject.TraceID)
		comparison := strings.Compare(currentID, lowestID)

		if comparison == 0 {
			lowestObjects = append(lowestObjects, currentObject)
			lowestBookmarks = append(lowestBookmarks, b)
		} else if len(lowestID) == 0 || comparison == -1 {
			lowestID = currentID
			lowestObjects = []*Trace{currentObject}
			lowestBookmarks = []*bookmark{b}
		}
	}

	lowestObject := lowestObjects[0]
	if len(lowestObjects) > 0 {
		lowestObject = CombineTraces(lowestObjects...)
	}
	for _, b := range lowestBookmarks {
		b.clear()
	}

	return lowestObject, nil
}

// Fully leftpad the hex string with 0's to make a complete trace ID.
func padTraceID(s string) string {
	if len(s) < 32 {
		return strings.Repeat("0", 32-len(s)) + s
	}
	return s
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
