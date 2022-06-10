package vparquet

import (
	"context"
	"io"

	"go.uber.org/atomic"
)

type prefetchIter struct {
	iter Iterator

	resultsCh chan *Trace
	quitCh    chan struct{}
	err       atomic.Error
}

func newPrefetchIterator(ctx context.Context, iter Iterator, bufferSize int) Iterator {
	p := &prefetchIter{
		iter:      iter,
		resultsCh: make(chan *Trace, bufferSize),
		quitCh:    make(chan struct{}, 1),
	}

	go p.prefetchLoop(ctx)

	return p
}

func (p *prefetchIter) prefetchLoop(ctx context.Context) {
	defer close(p.resultsCh)

	for {
		t, err := p.iter.Next(ctx)
		if err == io.EOF {
			return
		}
		if err != nil {
			p.err.Store(err)
			return
		}

		select {
		case <-ctx.Done():
			p.err.Store(err)
			return

		case <-p.quitCh:
			// Signalled to quit early
			return

		case p.resultsCh <- t:
			// Send results. Blocks until available buffer in channel
			// created by receiving in current()
		}
	}
}

func (p *prefetchIter) Next(ctx context.Context) (*Trace, error) {
	if err := p.err.Load(); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()

	case t, ok := <-p.resultsCh:
		if !ok {
			// Closed due to error?
			if err := p.err.Load(); err != nil {
				return nil, err
			}
			return nil, io.EOF
		}

		return t, nil
	}
}

func (b *prefetchIter) Close() {
	close(b.quitCh)
}
