package vparquet

import (
	"context"
	"io"
	"strings"

	"github.com/uber-go/atomic"
)

type bookmark struct {
	iter *iterator

	currentObject *Trace
	currentErr    atomic.Error

	resultsCh chan *Trace
	quitCh    chan struct{}
}

func newBookmark(ctx context.Context, iter *iterator, bufferSize int) *bookmark {
	b := &bookmark{
		iter:      iter,
		resultsCh: make(chan *Trace, bufferSize),
		quitCh:    make(chan struct{}),
	}

	go b.prefetchLoop(ctx)

	return b
}

func (b *bookmark) prefetchLoop(ctx context.Context) {
	defer close(b.resultsCh)

	for {
		t, err := b.iter.Next()
		if err == io.EOF {
			return
		}
		if err != nil {
			b.currentErr.Store(err)
			return
		}

		select {
		case <-ctx.Done():
			b.currentErr.Store(err)
			return

		case <-b.quitCh:
			// Signalled to quit early
			return

		case b.resultsCh <- t:
			// Send results. Blocks until available buffer in channel
			// created by receiving in current()
		}
	}
}

func (b *bookmark) current() (*Trace, error) {
	if err := b.currentErr.Load(); err != nil {
		return nil, err
	}

	if b.currentObject != nil {
		return b.currentObject, nil
	}

	// blocking wait on resultsCh
	t, ok := <-b.resultsCh
	if !ok {
		// check err
		if err := b.currentErr.Load(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}

	b.currentObject = t
	return b.currentObject, nil
}

func (b *bookmark) done() bool {
	obj, err := b.current()

	return obj == nil || err != nil
}

func (b *bookmark) clear() {
	b.currentObject = nil
}

func (b *bookmark) close() {
	close(b.quitCh)
}

type MultiBlockPrefetchIterator struct {
	bookmarks []*bookmark
}

func NewMultiblockPrefetchIterator(ctx context.Context, bookmarks []*bookmark) *MultiBlockPrefetchIterator {
	return &MultiBlockPrefetchIterator{
		bookmarks: bookmarks,
	}
}

func (m *MultiBlockPrefetchIterator) Close() {
	for _, b := range m.bookmarks {
		b.close()
	}
}

func (m *MultiBlockPrefetchIterator) Next(ctx context.Context) (*Trace, error) {
	allDone := func() bool {
		for _, b := range m.bookmarks {
			if !b.done() {
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
		currentObject, err := b.current()
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
