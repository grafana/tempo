package vparquet

import (
	"context"
	"io"

	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/segmentio/parquet-go"
	"go.uber.org/atomic"
)

// nolint: unused, deadcode
type prefetchIter struct {
	iter      RawIterator
	resultsCh chan fetchEntry
	quitCh    chan struct{}
	err       atomic.Error
}

type fetchEntry struct {
	id  common.ID
	row parquet.Row
}

var _ RawIterator = (*prefetchIter)(nil)

// nolint: unused, deadcode
func newPrefetchIterator(ctx context.Context, iter RawIterator, bufferSize int) *prefetchIter {
	p := &prefetchIter{
		iter:      iter,
		resultsCh: make(chan fetchEntry, bufferSize),
		quitCh:    make(chan struct{}, 1),
	}

	go p.prefetchLoop(ctx)

	return p
}

// nolint: unused, deadcode
func (p *prefetchIter) prefetchLoop(ctx context.Context) {
	defer close(p.resultsCh)
	defer p.iter.Close()

	for {
		id, t, err := p.iter.Next(ctx)
		if err != nil && err != io.EOF {
			p.err.Store(err)
			return
		}

		// block iterator returns nil error on io.EOF
		if t == nil || err == io.EOF {
			return
		}

		select {
		case <-ctx.Done():
			p.err.Store(err)
			return

		case <-p.quitCh:
			// Signalled to quit early
			return

		case p.resultsCh <- fetchEntry{id, t}:
			// Send results. Blocks until available buffer in channel
			// created by receiving in current()
		}
	}
}

func (p *prefetchIter) Next(ctx context.Context) (common.ID, parquet.Row, error) {
	if err := p.err.Load(); err != nil {
		return nil, nil, err
	}

	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()

	case f, ok := <-p.resultsCh:
		if !ok {
			// Closed due to error?
			if err := p.err.Load(); err != nil {
				return nil, nil, err
			}
			return nil, nil, io.EOF
		}

		return f.id, f.row, nil
	}
}

func (p *prefetchIter) Close() {
	close(p.quitCh)
}
