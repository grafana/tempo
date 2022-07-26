package vparquet

import (
	"bytes"
	"context"
	"io"

	"github.com/segmentio/parquet-go"
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

type MultiBlockIteratorRaw struct {
	bookmarks    []*bookmarkRaw
	reconstruct  func(parquet.Row) *Trace
	deconstruct  func(*Trace) parquet.Row
	objsCombined func()
}

func newMultiblockIteratorRaw(reconstruct func(parquet.Row) *Trace, deconstruct func(*Trace) parquet.Row, bookmarks []*bookmarkRaw, objsCombined func()) *MultiBlockIteratorRaw {
	return &MultiBlockIteratorRaw{
		bookmarks:    bookmarks,
		reconstruct:  reconstruct,
		deconstruct:  deconstruct,
		objsCombined: objsCombined,
	}
}

func (m *MultiBlockIteratorRaw) Next(ctx context.Context) ([]byte, parquet.Row, error) {

	if m.done(ctx) {
		return nil, nil, io.EOF
	}

	var lowestID []byte
	var lowestObjects []parquet.Row
	var lowestBookmarks []*bookmarkRaw

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

	lowestObject := m.combine(lowestObjects)

	for _, b := range lowestBookmarks {
		b.clear()
	}

	return lowestID, lowestObject, nil
}

func (m *MultiBlockIteratorRaw) combine(rows []parquet.Row) parquet.Row {
	if len(rows) == 0 {
		return nil
	}

	if len(rows) == 1 {
		return rows[0]
	}

	m.objsCombined()

	c := NewCombiner()
	for i, r := range rows {
		c.ConsumeWithFinal(m.reconstruct(r), i == len(rows))
	}
	tr, _ := c.Result()

	return m.deconstruct(tr)
}

func (m *MultiBlockIteratorRaw) Close() {
	for _, b := range m.bookmarks {
		b.close()
	}
}

func (m *MultiBlockIteratorRaw) done(ctx context.Context) bool {
	for _, b := range m.bookmarks {
		if !b.done(ctx) {
			return false
		}
	}
	return true
}
