package encoding

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

type multiblockIterator struct {
	combiner     common.ObjectCombiner
	bookmarks    []*bookmark
	dataEncoding string
	ctx          context.Context
	resultsCh    chan iteratorResult
	quitCh       chan bool

	errMtx sync.RWMutex
	err    error
}

type iteratorResult struct {
	ID   common.ID
	Body []byte
}

// NewMultiblockIterator Creates a new multiblock iterator. Iterates concurrently in a separate goroutine and results are buffered.
// Traces are deduped and combined using the object combiner.
func NewMultiblockIterator(ctx context.Context, inputs []Iterator, bufferSize int, combiner common.ObjectCombiner, dataEncoding string) Iterator {
	i := multiblockIterator{
		ctx:          ctx,
		combiner:     combiner,
		dataEncoding: dataEncoding,
		resultsCh:    make(chan iteratorResult, bufferSize),
		quitCh:       make(chan bool, 1),
	}

	for _, iter := range inputs {
		i.bookmarks = append(i.bookmarks, newBookmark(iter))
	}

	go i.iterate()

	return &i
}

func (i *multiblockIterator) Close() {
	select {
	case i.quitCh <- true:
	default:
		return
	}
}

func (i *multiblockIterator) Next(ctx context.Context) (common.ID, []byte, error) {
	if err := i.getErr(); err != nil {
		return nil, nil, err
	}

	select {
	case <-i.ctx.Done():
		return nil, nil, i.ctx.Err()

	case res, ok := <-i.resultsCh:
		if !ok {
			return nil, nil, io.EOF
		}

		return res.ID, res.Body, nil
	}
}

func (i *multiblockIterator) allDone() bool {
	for _, b := range i.bookmarks {
		if !b.done(i.ctx) {
			return false
		}
	}
	return true
}

func (i *multiblockIterator) getErr() error {
	i.errMtx.RLock()
	defer i.errMtx.RUnlock()
	return i.err
}

func (i *multiblockIterator) setErr(err error) {
	i.errMtx.Lock()
	defer i.errMtx.Unlock()
	i.err = err
}

func (i *multiblockIterator) iterate() {
	defer close(i.resultsCh)

	for !i.allDone() {
		var lowestID []byte
		var lowestObject []byte
		var lowestBookmark *bookmark

		// find lowest ID of the new object
		for _, b := range i.bookmarks {
			currentID, currentObject, err := b.current(i.ctx)
			if err == io.EOF {
				continue
			} else if err != nil {
				i.setErr(err)
				return
			}

			comparison := bytes.Compare(currentID, lowestID)

			if comparison == 0 {
				lowestObject, _ = i.combiner.Combine(currentObject, lowestObject, i.dataEncoding)
				b.clear()
			} else if len(lowestID) == 0 || comparison == -1 {
				lowestID = currentID
				lowestObject = currentObject
				lowestBookmark = b
			}
		}

		if len(lowestID) == 0 || len(lowestObject) == 0 || lowestBookmark == nil {
			i.setErr(errors.New("failed to find a lowest object in compaction"))
			return
		}

		// Copy slices allows data to escape the iterators
		res := iteratorResult{
			ID:   append([]byte(nil), lowestID...),
			Body: append([]byte(nil), lowestObject...),
		}

		lowestBookmark.clear()

		select {

		case <-i.ctx.Done():
			i.setErr(i.ctx.Err())
			return

		case <-i.quitCh:
			return

		case i.resultsCh <- res:
		}
	}
}

type bookmark struct {
	iter Iterator

	currentID     []byte
	currentObject []byte
	currentErr    error
}

func newBookmark(iter Iterator) *bookmark {
	return &bookmark{
		iter: iter,
	}
}

func (b *bookmark) current(ctx context.Context) ([]byte, []byte, error) {
	if len(b.currentID) != 0 && len(b.currentObject) != 0 {
		return b.currentID, b.currentObject, nil
	}

	if b.currentErr != nil {
		return nil, nil, b.currentErr
	}

	b.currentID, b.currentObject, b.currentErr = b.iter.Next(ctx)
	if b.currentErr != nil {
		return nil, nil, b.currentErr
	}

	return b.currentID, b.currentObject, nil
}

func (b *bookmark) done(ctx context.Context) bool {
	_, _, err := b.current(ctx)

	return err != nil
}

func (b *bookmark) clear() {
	b.currentID = nil
	b.currentObject = nil
}
