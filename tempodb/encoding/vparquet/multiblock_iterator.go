package vparquet

import (
	"bytes"
	"context"
	"io"

	"github.com/pkg/errors"
	"github.com/segmentio/parquet-go"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

type iteratable interface {
	parquet.Row | *Trace
}

type combineFn[T iteratable] func([]T) (T, error)

type MultiBlockIterator[T iteratable] struct {
	bookmarks []*bookmark[T]
	combine   combineFn[T]
}

func newMultiblockIterator[T iteratable](bookmarks []*bookmark[T], combine combineFn[T]) *MultiBlockIterator[T] {
	return &MultiBlockIterator[T]{
		bookmarks: bookmarks,
		combine:   combine,
	}
}

func (m *MultiBlockIterator[T]) Next(ctx context.Context) (common.ID, T, error) {
	if m.done(ctx) {
		return nil, nil, io.EOF
	}

	var (
		lowestID        common.ID
		lowestObjects   []T
		lowestBookmarks []*bookmark[T]
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

func (m *MultiBlockIterator[T]) Close() {
	for _, b := range m.bookmarks {
		b.close()
	}
}

func (m *MultiBlockIterator[T]) done(ctx context.Context) bool {
	for _, b := range m.bookmarks {
		if !b.done(ctx) {
			return false
		}
	}
	return true
}

type bookmark[T iteratable] struct {
	iter iterIterator[T]

	currentID     common.ID
	currentObject T
	currentErr    error
}

func newBookmark[T iteratable](iter iterIterator[T]) *bookmark[T] {
	return &bookmark[T]{
		iter: iter,
	}
}

func (b *bookmark[T]) current(ctx context.Context) ([]byte, T, error) {
	if b.currentErr != nil {
		return nil, nil, b.currentErr
	}

	if b.currentObject != nil {
		return b.currentID, b.currentObject, nil
	}

	b.currentID, b.currentObject, b.currentErr = b.iter.Next(ctx)
	return b.currentID, b.currentObject, b.currentErr
}

func (b *bookmark[T]) done(ctx context.Context) bool {
	_, obj, err := b.current(ctx)

	return obj == nil || err != nil
}

func (b *bookmark[T]) clear() {
	b.currentID = nil
	b.currentObject = nil
}

func (b *bookmark[T]) close() {
	b.iter.Close()
}

type iterIterator[T iteratable] interface { // jpe make this name not terrible
	Next(ctx context.Context) (common.ID, T, error)
	Close()
}
