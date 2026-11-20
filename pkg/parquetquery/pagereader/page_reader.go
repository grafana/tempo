// Package pagereader reads a column's parquet pages, both dictionary and plain encoded,
// slot by slot from their levels, for parquetquery's SyncIterator. It does so for two
// reasons:
//
//  1. Dictionary pushdown: on a dictionary-encoded chunk the filter is evaluated once
//     per dictionary entry, and a row is kept or skipped by its dictionary index,
//     without making its value.
//  2. Efficient seeking: a seek within a page walks the page's levels to the target
//     row, instead of reslicing the page (pq.Page.Slice) as parquet-go's value reader
//     would.
//
// A PageReader is split in two: slotReader walks the page's levels, and a valueReader reads
// the values of the slots it walks. A page is read by one of three value readers:
//
//   - dictValueReader: a dictionary page of a pushed-down chunk, kept or skipped by its
//     indexes.
//   - bufferedValueReader: any other page, until its first long seek: parquet-go's
//     value reader, in batches.
//   - seekableValueReader: any other page after a long seek, one value at a time.
//
// This is how it's wired into SyncIterator, for each chunk:
//
//	keepColumnChunk
//	  Reset()              forget the previous chunk's matches
//	  MatchDictionary()    if the chunk is dictionary-encoded and has a filter: evaluate
//	                       the filter on each entry, and skip the chunk if nothing
//	                       matches
//	setRowGroup
//	  SetChunk()           the column's max definition level; fails for a chunk that
//	                       isn't read from a file
//	for each page:
//	  setPage(pg)          Open(pg): loads the page's levels and its value reader; fails
//	                       when pg's levels and values disagree
//	  NextMatchedValue()   the value reader's: return the next matching row, until the
//	                       page is exhausted
//	  closePage()          Close()
//	on a seek:
//	  SkipToRow()          walk the page's levels to the target row; on a long seek, a
//	                       buffered page moves to a seekableValueReader
package pagereader

import (
	"errors"
	"fmt"

	"github.com/grafana/tempo/v3/pkg/parquetquery/base"
	pq "github.com/parquet-go/parquet-go"
	"github.com/parquet-go/parquet-go/encoding"
)

// seekThreshold is how many slots a seek on a buffered page must skip to move the page
// to a seekableValueReader: a shorter seek reads through the batch.
const seekThreshold = 1000

// ErrNotFileColumnChunk is returned by SetChunk for a column chunk that isn't read from a
// file.
var ErrNotFileColumnChunk = errors.New("column chunk isn't read from a file, so its max definition level is unknown")

// PageReader reads the pages of one column chunk at a time; see the package documentation.
// The zero value is ready for SetChunk.
type PageReader struct {
	// For the current chunk, set by MatchDictionary and Reset.
	matches   []bool // the filter's answer for each dictionary entry; nil when the chunk isn't pushed down
	matchNull bool   // the filter's answer for a null
	buf       []bool // storage for matches, reused across chunks

	// For the current page, set by Open.
	slots  slotReader  // the page's levels
	values valueReader // the page's value reader; nil between pages
	page   pq.Page     // nil between pages
}

// valueReader reads the values of a page's slots, as its slotReader walks them.
type valueReader interface {
	// nextMatchedValue is next's loop over the current page: it advances c over
	// every slot and returns the next value the filter and sampler keep, or ok=false once
	// the page is exhausted. The value is valid until the next call.
	nextMatchedValue(c *base.IteratorState, seeking bool) (v *pq.Value, ok bool, err error)
	// peekValue returns the value of the next slot, with its levels, without moving past
	// it: NilSyncIterator looks at a slot before it moves on. The page must have more
	// slots. The value is valid until the reader moves.
	peekValue() (*pq.Value, error)
	// nextSlot moves past the next slot, without reading its value.
	nextSlot()
	// unload releases what the reader holds of the page.
	unload()
}

// MatchDictionary evaluates filter against every entry of dict and reports whether
// any entry is kept.
func (r *PageReader) MatchDictionary(dict pq.Dictionary, filter base.Predicate, matchNull bool) (anyMatch bool) {
	n := dict.Len()
	if cap(r.buf) < n {
		r.buf = make([]bool, n)
	}
	matches := r.buf[:n]
	for i := range n {
		m := filter.KeepValue(dict.Index(int32(i)))
		matches[i] = m
		anyMatch = anyMatch || m
	}

	r.matches, r.matchNull = matches, matchNull
	return anyMatch
}

// Reset forgets the previous chunk's pushdown, so the next chunk is pushed down only
// once MatchDictionary has run for it.
func (r *PageReader) Reset() {
	r.matches = nil
}

// fileMaxDefinitionLevel returns cc's max definition level from the file's own column.
// The row group's schema isn't used: a file opened with another schema
// (parquet.FileSchema) can map the column to a different leaf.
func fileMaxDefinitionLevel(cc pq.ColumnChunk) (byte, bool) {
	fc, ok := cc.(*pq.FileColumnChunk)
	if !ok {
		return 0, false
	}
	col, ok := fc.Node().(*pq.Column)
	if !ok {
		return 0, false
	}
	return byte(col.MaxDefinitionLevel()), true
}

// SetChunk prepares the reader for the pages of cc and forgets the previous page. The
// pushdown set by MatchDictionary is kept: keepColumnChunk runs before SetChunk. It
// fails with ErrNotFileColumnChunk when cc isn't read from a file: the max definition
// level, which tells values from nulls, comes from the file's own column.
func (r *PageReader) SetChunk(cc pq.ColumnChunk) error {
	r.unload()
	maxDef, ok := fileMaxDefinitionLevel(cc)
	if !ok {
		return ErrNotFileColumnChunk
	}
	r.slots.maxDef = maxDef
	return nil
}

// Page returns the current page, nil between pages.
func (r *PageReader) Page() pq.Page {
	return r.page
}

// Open makes pg the current page and points the reader at its first slot. The previous
// page must be closed. It fails when pg's levels and values disagree, a slot would read
// the wrong value; pg is then still the current page, released by Close.
func (r *PageReader) Open(pg pq.Page) error {
	r.page = pg
	return r.load(pg)
}

// Close releases the current page, if any, and reports whether there was one.
func (r *PageReader) Close() bool {
	r.unload()
	if r.page == nil {
		return false
	}
	pq.Release(r.page)
	r.page = nil
	return true
}

// PushedDown reports whether the current page is read by its dictionary indexes.
func (r *PageReader) PushedDown() bool {
	_, ok := r.values.(*dictValueReader)
	return ok
}

// unload forgets the current page's levels and values.
func (r *PageReader) unload() {
	r.slots.unload()
	if r.values != nil {
		r.values.unload()
		r.values = nil
	}
}

// load points the reader at the start of pg. The reader must hold no page: Open is only
// called after Close. It fails when pg's levels and values disagree, a slot would read
// the wrong value, and then holds no page.
//
// pg must be a page as read from its column chunk, never a slice of one
// (pq.Page.Slice). pq.Page.Data shows a page's values from the start of their buffer,
// and the slice of a Boolean page can begin within a byte, at a bit offset Data doesn't
// show, so its values would be read from the wrong bits. Nothing in SyncIterator slices
// pages.
func (r *PageReader) load(pg pq.Page) error {
	err := r.slots.load(pg)
	if err == nil {
		err = r.initValueReader(pg)
	}
	if err != nil {
		r.unload()
	}
	return err
}

// initValueReader picks the value reader of pg. A dictionary page of a pushed-down
// chunk is read by its indexes, without making values. Any other page is buffered until
// its first long seek.
func (r *PageReader) initValueReader(pg pq.Page) error {
	data := pg.Data()
	dict := pg.Dictionary()
	if dict != nil && data.Kind() != encoding.Int32 {
		return fmt.Errorf("dictionary page has %v indexes, want int32", data.Kind())
	}
	if err := checkValueCount(data, r.slots.numValues); err != nil {
		return err
	}

	if dict != nil && r.matches != nil {
		r.values = newDictValueReader(&r.slots, pg, r.matches, r.matchNull)
	} else {
		r.values = newBufferedValueReader(&r.slots, pg)
	}
	return nil
}

// checkValueCount checks that a page's data has one entry for each slot that holds a
// value, want of them: the values of a plain page, or the indexes of a dictionary
// page. Otherwise reading the slots in order would give a slot another slot's value.
// parquet-go always decodes pages this way, so this only catches a broken invariant,
// for a length comparison per page.
func checkValueCount(data encoding.Values, want int) error {
	var got int
	switch data.Kind() {
	case encoding.Boolean:
		// Booleans are bit-packed: the last byte may hold fewer than 8.
		if got = len(data.Boolean()) * 8; got >= want && got-want < 8 {
			return nil
		}
	case encoding.Int32:
		got = len(data.Int32())
	case encoding.Int64:
		got = len(data.Int64())
	case encoding.Int96:
		got = len(data.Int96())
	case encoding.Float:
		got = len(data.Float())
	case encoding.Double:
		got = len(data.Double())
	case encoding.ByteArray:
		_, offsets := data.ByteArray()
		got = len(offsets) - 1
	case encoding.FixedLenByteArray:
		b, size := data.FixedLenByteArray()
		if size <= 0 {
			return fmt.Errorf("page of kind %v has value size %d", data.Kind(), size)
		}
		got = len(b) / size
		if len(b)%size != 0 {
			return fmt.Errorf("page of kind %v has %d bytes, not a multiple of %d", data.Kind(), len(b), size)
		}
	default:
		return fmt.Errorf("page of unknown kind %v", data.Kind())
	}
	if got != want {
		return fmt.Errorf("page of kind %v has %d values for %d present slots", data.Kind(), got, want)
	}
	return nil
}

// NextMatchedValue returns the next value of the current page that c's filter and
// sampler keep; see valueReader.nextMatchedValue.
func (r *PageReader) NextMatchedValue(c *base.IteratorState, seeking bool) (v *pq.Value, ok bool, err error) {
	return r.values.nextMatchedValue(c, seeking)
}

// SkipToRow moves the reader to the first slot of root row targetRow, and reports
// whether it did; see slotReader.skipToRow. On a buffered page, a seek past more than
// seekThreshold slots moves the rest of the page to a seekableValueReader: walking the
// levels and making only the values it returns is cheaper than decoding every skipped
// value. A shorter seek doesn't move: next reads through the batch, which is cheaper
// when the page is then read on, as by a filter.
func (r *PageReader) SkipToRow(currentRow, targetRow int32) bool {
	fromSlot, fromIndex := r.slots.slot, r.slots.index
	if !r.slots.skipToRow(currentRow, targetRow) {
		return false
	}
	if _, ok := r.values.(*bufferedValueReader); ok {
		if r.slots.slot-fromSlot <= seekThreshold {
			// A short seek: next reads through the batch instead.
			r.slots.slot, r.slots.index = fromSlot, fromIndex
			return false
		}
		r.values.unload()
		r.values = newSeekableValueReader(&r.slots, r.page)
	}
	return true
}

// PeekValue returns the value of the next slot without moving past it, or nil once the
// page has no more slots; see valueReader.peekValue.
func (r *PageReader) PeekValue() (*pq.Value, error) {
	if !r.slots.hasMoreSlots() {
		return nil, nil
	}
	return r.values.peekValue()
}

// NextSlot moves past the next slot; see valueReader.nextSlot.
func (r *PageReader) NextSlot() {
	r.values.nextSlot()
}
