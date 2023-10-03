package v2

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/uber-go/atomic"

	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type multiblockIterator struct {
	combiner     model.ObjectCombiner
	bookmarks    []*bookmark
	dataEncoding string
	resultsCh    chan iteratorResult
	quitCh       chan struct{}
	err          atomic.Error
	logger       log.Logger
}

var _ BytesIterator = (*multiblockIterator)(nil)

type iteratorResult struct {
	id     common.ID
	object []byte
}

// NewMultiblockIterator Creates a new multiblock iterator. Iterates concurrently in a separate goroutine and results are buffered.
// Traces are deduped and combined using the object combiner.
func NewMultiblockIterator(ctx context.Context, inputs []BytesIterator, bufferSize int, combiner model.ObjectCombiner, dataEncoding string, logger log.Logger) BytesIterator {
	i := multiblockIterator{
		combiner:     combiner,
		dataEncoding: dataEncoding,
		resultsCh:    make(chan iteratorResult, bufferSize),
		quitCh:       make(chan struct{}, 1),
		logger:       logger,
	}

	for _, iter := range inputs {
		i.bookmarks = append(i.bookmarks, newBookmark(iter))
	}

	go i.iterate(ctx)

	return &i
}

// Close iterator, signals goroutine to exit if still running.
func (i *multiblockIterator) Close() {
	select {
	// Signal goroutine to quit. Non-blocking, handles if already
	// signalled or goroutine not listening to channel.
	case i.quitCh <- struct{}{}:
	default:
		return
	}
}

// Next returns the next values or error.  Blocking read when data not yet available.
func (i *multiblockIterator) NextBytes(ctx context.Context) (common.ID, []byte, error) {
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

func (i *multiblockIterator) allDone(ctx context.Context) bool {
	for _, b := range i.bookmarks {
		if !b.done(ctx) {
			return false
		}
	}
	return true
}

func (i *multiblockIterator) iterate(ctx context.Context) {
	defer close(i.resultsCh)

	for !i.allDone(ctx) {
		var lowestID []byte
		var lowestObjects [][]byte
		var lowestBookmarks []*bookmark

		// find lowest ID of the new object
		for _, b := range i.bookmarks {
			currentID, currentObject, err := b.current(ctx)
			if errors.Is(err, io.EOF) {
				continue
			} else if err != nil {
				i.err.Store(err)
				return
			}

			comparison := bytes.Compare(currentID, lowestID)

			if comparison == 0 {
				lowestObjects = append(lowestObjects, currentObject)
				lowestBookmarks = append(lowestBookmarks, b)
			} else if len(lowestID) == 0 || comparison == -1 {
				lowestID = currentID
				lowestObjects = [][]byte{currentObject}
				lowestBookmarks = []*bookmark{b}
			}
		}

		lowestObject, _, err := i.combiner.Combine(i.dataEncoding, lowestObjects...)
		if err != nil {
			i.err.Store(fmt.Errorf("error combining while Nexting: %w", err))
			return
		}
		for _, b := range lowestBookmarks {
			b.clear()
		}

		if len(lowestID) == 0 || len(lowestObject) == 0 || len(lowestBookmarks) == 0 {
			// Skip empty objects or when the bookmarks failed to return an object.
			// This intentional here because we concluded that the bookmarks have already
			// been skipping most empties (but not all) and there is no reason to treat the
			// unskipped edge cases differently. Edge cases:
			// * Two empties in a row:  the bookmark won't skip the second empty. Since
			//   we already skipped the first, go ahead and skip the second.
			// * Last trace across all blocks is empty. In that case there is no next record
			//   for the bookmarks to skip to, and lowestBookmark remains nil.  Since we
			//   already skipped every other empty, skip the last (but not least) entry.
			// (todo: research needed to determine how empties get in the block)
			level.Warn(i.logger).Log("msg", "multiblock iterator skipping empty object", "id", hex.EncodeToString(lowestID), "obj", lowestObject, "bookmark", lowestBookmarks)
			continue
		}

		// Copy slices allows data to escape the iterators
		res := iteratorResult{
			id:     append([]byte(nil), lowestID...),
			object: append([]byte(nil), lowestObject...),
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

type bookmark struct {
	iter BytesIterator

	currentID     []byte
	currentObject []byte
	currentErr    error
}

func newBookmark(iter BytesIterator) *bookmark {
	return &bookmark{
		iter: iter,
	}
}

func (b *bookmark) current(ctx context.Context) ([]byte, []byte, error) {
	// This check is how the bookmark knows to iterate after being cleared,
	// but it also unintentionally skips empty objects that somehow got in
	// the block (b.currentObject is empty slice).  Normal usage of the bookmark
	// is to call done() and then current(). done() calls current() which reads iter(.Next)()
	// and saves empty, it is then iterated again by a direct call to current(),
	// which interprets the empty object as a cleared state and iterates again.
	// This is mostly harmless and has been true historically for some time,
	// which isn't great because it masks empty objects present in a block
	// (todo: research needed to determine how they get there), but it's made worse
	// in that the skip fails in some edge cases:
	// * Two empty objects in a row:  done()/current() will return the
	//   second empty object and not skip it, and this fails up the call chain.
	// * Last object is empty: This is an issue in multiblock-iterator, see
	//   notes there.
	if len(b.currentID) != 0 && len(b.currentObject) != 0 {
		return b.currentID, b.currentObject, nil
	}

	if b.currentErr != nil {
		return nil, nil, b.currentErr
	}

	b.currentID, b.currentObject, b.currentErr = b.iter.NextBytes(ctx)
	return b.currentID, b.currentObject, b.currentErr
}

func (b *bookmark) done(ctx context.Context) bool {
	_, _, err := b.current(ctx)

	return err != nil
}

func (b *bookmark) clear() {
	b.currentID = nil
	b.currentObject = nil
}
