package vparquet

import (
	"context"
	"io"
	"strings"

	"github.com/uber-go/atomic"
)

type MultiBlockPrefetchIterator struct {
	bookmarks []*bookmark

	resultsCh chan *Trace
	quitCh    chan struct{}
	err       atomic.Error
}

func NewMultiblockPrefetchIterator(ctx context.Context, bookmarks []*bookmark, bufferSize int) *MultiBlockPrefetchIterator {
	m := &MultiBlockPrefetchIterator{
		bookmarks: bookmarks,
		resultsCh: make(chan *Trace, bufferSize),
		quitCh:    make(chan struct{}),
	}

	go m.prefetchLoop(ctx)

	return m
}

func (m *MultiBlockPrefetchIterator) Close() {
	select {
	// Signal goroutine to quit. Non-blocking, handles if already
	// signalled or goroutine not listening to channel.
	case m.quitCh <- struct{}{}:
	default:
		return
	}
}

func (m *MultiBlockPrefetchIterator) Next(ctx context.Context) (*Trace, error) {
	if err := m.err.Load(); err != nil {
		return nil, err
	}

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()

		case t, ok := <-m.resultsCh:
			if !ok {
				// check err
				if err := m.err.Load(); err != nil {
					return nil, err
				}
				return nil, io.EOF
			}
			return t, nil
		}
	}
}

func (m *MultiBlockPrefetchIterator) prefetchLoop(ctx context.Context) {
	defer close(m.resultsCh)

	for {
		t, err := m.fetch()
		if err == io.EOF {
			return
		}
		if err != nil {
			m.err.Store(err)
			return
		}

		select {
		case <-ctx.Done():
			m.err.Store(ctx.Err())
			return

		case <-m.quitCh:
			// Signalled to quit early
			return

		case m.resultsCh <- t:
			// Send results. Blocks until available buffer in channel
			// created by receiving in Next()
		}
	}
}

func (m *MultiBlockPrefetchIterator) fetch() (*Trace, error) {
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
