package v2

import (
	"context"
	"errors"
	"io"

	"github.com/uber-go/atomic"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

type prefetchIterator struct {
	iter      BytesIterator
	resultsCh chan iteratorResult
	quitCh    chan struct{}
	err       atomic.Error
}

var _ BytesIterator = (*prefetchIterator)(nil)

// NewPrefetchIterator Creates a new multiblock iterator. Iterates concurrently in a separate goroutine and results are buffered.
func NewPrefetchIterator(ctx context.Context, iter BytesIterator, bufferSize int) BytesIterator {
	i := prefetchIterator{
		iter:      iter,
		resultsCh: make(chan iteratorResult, bufferSize),
		quitCh:    make(chan struct{}, 1),
	}

	go i.iterate(ctx)

	return &i
}

// Close iterator, signals goroutine to exit if still running.
func (i *prefetchIterator) Close() {
	select {
	// Signal goroutine to quit. Non-blocking, handles if already
	// signalled or goroutine not listening to channel.
	case i.quitCh <- struct{}{}:
	default:
		return
	}
}

// Next returns the next values or error.  Blocking read when data not yet available.
func (i *prefetchIterator) NextBytes(ctx context.Context) (common.ID, []byte, error) {
	if err := i.err.Load(); err != nil {
		return nil, nil, err
	}

	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()

	case res, ok := <-i.resultsCh:
		if !ok {
			// Closed due to error?
			if err := i.err.Load(); err != nil {
				return nil, nil, err
			}
			return nil, nil, io.EOF
		}

		return res.id, res.object, nil
	}
}

func (i *prefetchIterator) iterate(ctx context.Context) {
	defer close(i.resultsCh)
	defer i.iter.Close()

	for {
		id, obj, err := i.iter.NextBytes(ctx)
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			i.err.Store(err)
		}

		// Copy slices allows data to escape the iterators
		res := iteratorResult{
			id:     id,
			object: obj,
		}

		select {

		case <-ctx.Done():
			i.err.Store(ctx.Err())
			return

		case <-i.quitCh:
			// Signalled to quit early
			return

		case i.resultsCh <- res:
			// Send results. Blocks until available buffer in channel
			// created by receiving in Next()
		}
	}
}
