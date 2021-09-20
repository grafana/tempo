package encoding

import (
	"context"
	"io"

	"github.com/grafana/tempo/tempodb/encoding/common"

	"github.com/uber-go/atomic"
)

type prefetchIteratorChannel struct {
	iter      Iterator
	resultsCh chan iteratorResult
	quitCh    chan struct{}
	err       atomic.Error
}

var _ Iterator = (*prefetchIteratorChannel)(nil)

// NewPrefetchIteratorChannel Creates a new multiblock iterator. Iterates concurrently in a separate goroutine and results are buffered.
// Traces are deduped and combined using the object combiner.
func NewPrefetchIteratorChannel(ctx context.Context, iter Iterator, bufferSize int) Iterator {
	i := prefetchIteratorChannel{
		iter:      iter,
		resultsCh: make(chan iteratorResult, bufferSize),
		quitCh:    make(chan struct{}, 1),
	}

	go i.iterate(ctx)

	return &i
}

// Close iterator, signals goroutine to exit if still running.
func (i *prefetchIteratorChannel) Close() {
	select {
	// Signal goroutine to quit. Non-blocking, handles if already
	// signalled or goroutine not listening to channel.
	case i.quitCh <- struct{}{}:
	default:
		return
	}
}

// Next returns the next values or error.  Blocking read when data not yet available.
func (i *prefetchIteratorChannel) Next(ctx context.Context) (common.ID, []byte, error) {
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

func (i *prefetchIteratorChannel) iterate(ctx context.Context) {
	defer close(i.resultsCh)

	for {
		id, obj, err := i.iter.Next(ctx)
		if err == io.EOF {
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
