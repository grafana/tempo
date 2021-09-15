package encoding

import (
	"context"
	"fmt"
	"io"

	"github.com/grafana/tempo/pkg/flushqueues"
	"github.com/grafana/tempo/tempodb/encoding/common"

	"github.com/uber-go/atomic"
)

type prefetchIterator struct {
	iter Iterator
	q    *flushqueues.Queueimpl7
	done atomic.Bool
	err  atomic.Error
}

var _ Iterator = (*prefetchIterator)(nil)

// NewPrefetchIterator Creates a new prefetch iterator. Iterates concurrently in a separate goroutine and results are buffered.
func NewPrefetchIterator(ctx context.Context, iter Iterator, bufferSize int) Iterator {
	i := prefetchIterator{
		iter: iter,
		q:    flushqueues.NewQ(),
	}

	go i.iterate(ctx)

	return &i
}

// Close iterator, signals goroutine to exit if still running.
func (i *prefetchIterator) Close() {
	i.done.Store(true)
}

// Next returns the next values or error.  Blocking read when data not yet available.
func (i *prefetchIterator) Next(ctx context.Context) (common.ID, []byte, error) {
	if err := i.err.Load(); err != nil {
		return nil, nil, err
	}

	if i.done.Load() {
		return nil, nil, io.EOF
	}
	res, ok := i.q.Pop()
	if !ok {
		return nil, nil, io.EOF
	}
	res2 := res.(iteratorResult)
	return res2.id, res2.object, nil
}

func (i *prefetchIterator) iterate(ctx context.Context) {
	for {
		id, obj, err := i.iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			i.err.Store(err)
			break
		}

		res := iteratorResult{
			id:     id,
			object: obj,
		}

		i.q.Push(res)
	}

	fmt.Println("done with this garbage")
}
