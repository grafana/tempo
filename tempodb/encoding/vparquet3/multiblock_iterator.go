package vparquet3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/parquet-go/parquet-go"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

type iteratable interface {
	parquet.Row | *Trace | *uint8
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
		lowestBookmarks []*bookmark[T]
	)

	// find lowest ID of the new object
	for _, b := range m.bookmarks {
		id, err := b.peekID(ctx)
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, nil, err
		}
		if id == nil {
			continue
		}

		comparison := bytes.Compare(id, lowestID)

		if comparison == 0 {
			lowestBookmarks = append(lowestBookmarks, b)
		} else if len(lowestID) == 0 || comparison == -1 {
			lowestID = id

			// reset and reuse
			lowestBookmarks = lowestBookmarks[:0]
			lowestBookmarks = append(lowestBookmarks, b)
		}
	}

	// now get the lowest objects from our bookmarks
	lowestObjects := make([]T, 0, len(lowestBookmarks))
	for _, b := range lowestBookmarks {
		_, obj, err := b.current(ctx)
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, nil, err
		}
		if obj == nil {
			// this should never happen. id was non-nil above
			return nil, nil, errors.New("unexpected nil object from lowest bookmark")
		}
		lowestObjects = append(lowestObjects, obj)
	}

	lowestObject, err := m.combine(lowestObjects)
	if err != nil {
		return nil, nil, fmt.Errorf("combining: %w", err)
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
	iter bookmarkIterator[T]

	currentID     common.ID
	currentObject T
	currentErr    error
}

type bookmarkIterator[T iteratable] interface {
	Next(ctx context.Context) (common.ID, T, error)
	Close()
	peekNextID(ctx context.Context) (common.ID, error)
}

func newBookmark[T iteratable](iter bookmarkIterator[T]) *bookmark[T] {
	return &bookmark[T]{
		iter: iter,
	}
}

func (b *bookmark[T]) peekID(ctx context.Context) (common.ID, error) {
	nextID, err := b.iter.peekNextID(ctx)
	if !errors.Is(err, common.ErrUnsupported) {
		return nextID, err
	}

	id, _, err := b.current(ctx)
	return id, err
}

func (b *bookmark[T]) current(ctx context.Context) (common.ID, T, error) {
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
	nextID, err := b.iter.peekNextID(ctx)
	if !errors.Is(err, common.ErrUnsupported) {
		return nextID == nil || err != nil
	}

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
