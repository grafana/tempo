package pagereader

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/grafana/tempo/v3/pkg/parquetquery/base"
	pq "github.com/parquet-go/parquet-go"
)

// batchSize is how many values a bufferedValueReader reads at a time.
const batchSize = 1000

// valueBufferPool holds the batches of bufferedValueReader.
var valueBufferPool = sync.Pool{
	New: func() any {
		b := make([]pq.Value, 0, batchSize)
		return &b
	},
}

// bufferedValueReader reads a page that isn't pushed down until its first seek, through
// parquet-go's value reader in batches. That is how a scan reads fastest: parquet-go
// makes the values with their levels in one pass, where seekableValueReader pays a call
// to the dictionary and to pq.Value.Level per value. The slotReader still counts every
// slot, so a seek can move the rest of the page to a seekableValueReader at any point.
type bufferedValueReader struct {
	slots  *slotReader
	values pq.ValueReader // the page's value reader
	buf    *[]pq.Value    // from valueBufferPool
	batch  []pq.Value     // the values read ahead, with their levels
	next   int            // next value of batch
}

// newBufferedValueReader returns a bufferedValueReader of pg, whose levels slots holds.
func newBufferedValueReader(slots *slotReader, pg pq.Page) *bufferedValueReader {
	return &bufferedValueReader{slots: slots, values: pg.Values(), buf: valueBufferPool.Get().(*[]pq.Value)}
}

// unload returns the buffer to the pool.
func (r *bufferedValueReader) unload() {
	clear((*r.buf)[:cap(*r.buf)]) // don't keep the values' page alive
	valueBufferPool.Put(r.buf)
	r.buf, r.batch = nil, nil
}

func (r *bufferedValueReader) peekValue() (*pq.Value, error) {
	if r.next == len(r.batch) {
		if err := r.readBatch(); err != nil {
			return nil, err
		}
	}
	return &r.batch[r.next], nil
}

// nextSlot moves past the next slot and its value, which peekValue has read.
func (r *bufferedValueReader) nextSlot() {
	r.next++
	r.slots.nextSlot()
}

// readBatch reads the next values of the page.
func (r *bufferedValueReader) readBatch() error {
	buf := (*r.buf)[:cap(*r.buf)]
	n, err := r.values.ReadValues(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if n == 0 {
		return fmt.Errorf("page's value reader ended before its last %d slots", r.slots.numSlots-r.slots.slot)
	}
	r.batch, r.next = buf[:n], 0
	return nil
}

// nextMatchedValue implements valueReader.
func (r *bufferedValueReader) nextMatchedValue(c *base.IteratorState, seeking bool) (*pq.Value, bool, error) {
	s := r.slots
	for s.hasMoreSlots() {
		if r.next == len(r.batch) {
			if err := r.readBatch(); err != nil {
				return nil, false, err
			}
		}
		v := &r.batch[r.next]
		r.next++
		rep, def, _ := s.nextSlot()
		c.Row.Next(rep, def, c.MaxDef)

		if seeking && c.BeforeSeekTarget() {
			continue
		}

		if c.Filter != nil {
			c.Stats.InspectedValues++
			if !c.Filter.KeepValue(*v) {
				continue
			}
			c.Stats.KeptValues++
		}
		if c.Sampler != nil && !c.Sampler.Sample() {
			continue
		}
		return v, true, nil
	}
	return nil, false, nil
}
