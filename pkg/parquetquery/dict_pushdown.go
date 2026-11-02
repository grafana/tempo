package parquetquery

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"

	pq "github.com/parquet-go/parquet-go"
	"github.com/parquet-go/parquet-go/encoding"
)

const (
	// batchSize is how many values a bufferedValueReader reads at a time.
	batchSize = 1000
	// seekThreshold is how many slots a seek on a buffered page must skip to move the
	// page to a seekableValueReader: a shorter seek reads through the batch.
	seekThreshold = 1000
)

// ErrNotFileColumnChunk is returned by SetChunk for a column chunk that isn't read from a
// file.
var ErrNotFileColumnChunk = errors.New("column chunk isn't read from a file, so its max definition level is unknown")

// valueBufferPool holds the batches of bufferedValueReader.
var valueBufferPool = sync.Pool{
	New: func() any {
		b := make([]pq.Value, 0, batchSize)
		return &b
	},
}

// pageReader reads every page, both dictionary and plain encoded pages, slot by slot
// from its levels. It does so for two reasons:
//
//  1. Dictionary pushdown: on a dictionary-encoded chunk the filter is evaluated once
//     per dictionary entry, and a row is kept or skipped by its dictionary index,
//     without making its value.
//  2. Efficient seeking: a seek within a page walks the page's levels to the target
//     row, instead of reslicing the page (pq.Page.Slice) as parquet-go's value reader
//     would.
//
// It is split in two: slotReader walks the page's levels, and a valueReader reads the
// values of the slots it walks. A page is read by one of three value readers:
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
type pageReader struct {
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
	// nextMatchedValue is next's loop over the current page: it advances c.curr over
	// every slot and returns the next value the filter and sampler keep, or ok=false once
	// the page is exhausted. The value is valid until the next call.
	nextMatchedValue(c *SyncIterator, seeking bool) (v *pq.Value, ok bool, err error)
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
func (r *pageReader) MatchDictionary(dict pq.Dictionary, filter Predicate, matchNull bool) (anyMatch bool) {
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
func (r *pageReader) Reset() {
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
func (r *pageReader) SetChunk(cc pq.ColumnChunk) error {
	r.unload()
	maxDef, ok := fileMaxDefinitionLevel(cc)
	if !ok {
		return ErrNotFileColumnChunk
	}
	r.slots.maxDef = maxDef
	return nil
}

// Page returns the current page, nil between pages.
func (r *pageReader) Page() pq.Page {
	return r.page
}

// Open makes pg the current page and points the reader at its first slot. The previous
// page must be closed. It fails when pg's levels and values disagree, a slot would read
// the wrong value; pg is then still the current page, released by Close.
func (r *pageReader) Open(pg pq.Page) error {
	r.page = pg
	return r.load(pg)
}

// Close releases the current page, if any, and reports whether there was one.
func (r *pageReader) Close() bool {
	r.unload()
	if r.page == nil {
		return false
	}
	pq.Release(r.page)
	r.page = nil
	return true
}

// PushedDown reports whether the current page is read by its dictionary indexes.
func (r *pageReader) PushedDown() bool {
	_, ok := r.values.(*dictValueReader)
	return ok
}

// unload forgets the current page's levels and values.
func (r *pageReader) unload() {
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
func (r *pageReader) load(pg pq.Page) error {
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
func (r *pageReader) initValueReader(pg pq.Page) error {
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
func (r *pageReader) NextMatchedValue(c *SyncIterator, seeking bool) (v *pq.Value, ok bool, err error) {
	return r.values.nextMatchedValue(c, seeking)
}

// SkipToRow moves the reader to the first slot of root row targetRow, and reports
// whether it did; see slotReader.skipToRow. On a buffered page, a seek past more than
// seekThreshold slots moves the rest of the page to a seekableValueReader: walking the
// levels and making only the values it returns is cheaper than decoding every skipped
// value. A shorter seek doesn't move: next reads through the batch, which is cheaper
// when the page is then read on, as by a filter.
func (r *pageReader) SkipToRow(currentRow, targetRow int32) bool {
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
func (r *pageReader) PeekValue() (*pq.Value, error) {
	if !r.slots.hasMoreSlots() {
		return nil, nil
	}
	return r.values.peekValue()
}

// NextSlot moves past the next slot; see valueReader.nextSlot.
func (r *pageReader) NextSlot() {
	r.values.nextSlot()
}

// dictValueReader reads a dictionary-encoded page of a pushed-down chunk. The filter's
// answer for each dictionary entry is computed once per chunk, so a slot is kept or
// skipped by its index, and a row that doesn't match is never turned into a value.
// Take an optional string column (max definition level 1) whose chunk has the
// dictionary [api, shop], filtered by = "shop":
//
//	matches = [false, true] // one answer per dictionary entry
//
//	row        0     1     2     3
//	value      shop  null  api   shop
//	def level  1     0     1     1
//	indexes    1           0     1     // = [1, 0, 1]: nulls have no index
type dictValueReader struct {
	slots     *slotReader
	matches   []bool // the filter's answer for each dictionary entry, from MatchDictionary
	matchNull bool   // the filter's answer for a null
	dict      pq.Dictionary
	indexes   []int32 // dictionary index of each present value
	column    int
	value     pq.Value
}

// newDictValueReader returns a dictValueReader of pg, a dictionary page whose levels
// slots holds, keeping the entries matches keeps.
func newDictValueReader(slots *slotReader, pg pq.Page, matches []bool, matchNull bool) *dictValueReader {
	data := pg.Data()
	return &dictValueReader{
		slots:     slots,
		matches:   matches,
		matchNull: matchNull,
		dict:      pg.Dictionary(),
		indexes:   data.Int32(),
		column:    pg.Column(),
	}
}

func (r *dictValueReader) unload() {}

func (r *dictValueReader) peekValue() (*pq.Value, error) {
	rep, def, index := r.slots.peekSlot()
	var v pq.Value
	if index >= 0 {
		v = r.dict.Index(r.indexes[index])
	}
	r.value = v.Level(rep, def, r.column)
	return &r.value, nil
}

func (r *dictValueReader) nextSlot() { r.slots.nextSlot() }

// nextMatchedValue implements valueReader. A pushed-down page always has a filter. On
// the example:
//
//	slot 0: def 1 = maxDef, takes index 1, matches[1] is true: returns shop (row 0)
//	slot 1: def 0, a null, matchNull is false: skipped
//	slot 2: def 1, takes index 0, matches[0] is false: skipped
//	slot 3: def 1, takes index 1: returns shop (row 3)
func (r *dictValueReader) nextMatchedValue(c *SyncIterator, seeking bool) (*pq.Value, bool, error) {
	s := r.slots
	for s.hasMoreSlots() {
		rep, def, index := s.nextSlot()
		c.curr.Next(rep, def, c.maxDefinitionLevel)

		if seeking && c.beforeSeekTarget() {
			continue
		}

		c.stats.InspectedValues++
		keep := r.matchNull // a null has no dictionary entry
		if index >= 0 {
			keep = r.matches[r.indexes[index]]
		}
		if !keep {
			continue
		}
		c.stats.KeptValues++

		if c.sampler != nil && !c.sampler.Sample() {
			continue
		}
		var v pq.Value
		if index >= 0 {
			v = r.dict.Index(r.indexes[index])
		}
		r.value = v.Level(rep, def, r.column)
		return &r.value, true, nil
	}
	return nil, false, nil
}

// seekableValueReader reads a page that isn't pushed down from its levels, one value at
// a time, after a long seek. A value is only made into a pq.Value
// when it is returned or tested by the filter: values.Index(indexes[i]) for a
// dictionary page, values.Index(i) for a plain page, whose values are wrapped in a
// dictionary (pq.Type.NewDictionary) so value i is made the way the page's own value
// reader makes it.
type seekableValueReader struct {
	slots   *slotReader
	values  pq.Dictionary // the page's dictionary, or one over its values for a plain page
	indexes []int32       // dictionary index of each present value; nil for a plain page
	column  int
	value   pq.Value
}

// newSeekableValueReader returns a seekableValueReader of pg, whose levels slots
// holds.
func newSeekableValueReader(slots *slotReader, pg pq.Page) *seekableValueReader {
	v := &seekableValueReader{slots: slots, column: pg.Column()}
	data := pg.Data()
	if dict := pg.Dictionary(); dict != nil {
		v.values, v.indexes = dict, data.Int32()
	} else {
		v.values = pg.Type().NewDictionary(pg.Column(), slots.numValues, data)
	}
	return v
}

func (r *seekableValueReader) unload() {}

func (r *seekableValueReader) peekValue() (*pq.Value, error) {
	rep, def, index := r.slots.peekSlot()
	r.value = r.makeValue(index, rep, def)
	return &r.value, nil
}

func (r *seekableValueReader) nextSlot() { r.slots.nextSlot() }

// nextMatchedValue implements valueReader.
func (r *seekableValueReader) nextMatchedValue(c *SyncIterator, seeking bool) (*pq.Value, bool, error) {
	s := r.slots
	for s.hasMoreSlots() {
		rep, def, index := s.nextSlot()
		c.curr.Next(rep, def, c.maxDefinitionLevel)

		if seeking && c.beforeSeekTarget() {
			continue
		}

		made := false
		if c.filter != nil {
			c.stats.InspectedValues++
			r.value, made = r.makeValue(index, rep, def), true
			if !c.filter.KeepValue(r.value) {
				continue
			}
			c.stats.KeptValues++
		}
		if c.sampler != nil && !c.sampler.Sample() {
			continue
		}
		if !made {
			r.value = r.makeValue(index, rep, def)
		}
		return &r.value, true, nil
	}
	return nil, false, nil
}

// makeValue returns the value of a slot with levels rep and def. index is the slot's
// present value, or -1 for a null.
func (r *seekableValueReader) makeValue(index, rep, def int) pq.Value {
	var v pq.Value
	switch {
	case index < 0:
	case r.indexes != nil:
		v = r.values.Index(r.indexes[index])
	default:
		v = r.values.Index(int32(index))
	}
	return v.Level(rep, def, r.column)
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
func (r *bufferedValueReader) nextMatchedValue(c *SyncIterator, seeking bool) (*pq.Value, bool, error) {
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
		c.curr.Next(rep, def, c.maxDefinitionLevel)

		if seeking && c.beforeSeekTarget() {
			continue
		}

		if c.filter != nil {
			c.stats.InspectedValues++
			if !c.filter.KeepValue(*v) {
				continue
			}
			c.stats.KeptValues++
		}
		if c.sampler != nil && !c.sampler.Sample() {
			continue
		}
		return v, true, nil
	}
	return nil, false, nil
}

// slotReader walks a page's levels: the levels of each slot, which value a slot holds,
// and where a row starts.
//
// Parquet stores nested columns with Dremel encoding: every entry of a page has a
// repetition and a definition level, and those levels, not the values, tell where a
// row starts and whether an entry holds a value. So a seek only reads the levels:
//
//   - slot: one entry of the page, a value or a null.
//   - rep: the repetition level, 0 where a root row starts.
//   - def: the definition level; maxDef, the column's max definition level, where the
//     slot holds a value.
//   - values: only the slots that hold a value (pq.Page.Data).
//   - index: the next value to read; it moves only at slots that hold one.
//
// For example:
//
//	slot       0  1  2  3  4
//	row        0  0  1  2  2
//	rep        0  1  0  0  1
//	def        2  2  0  2  2    // maxDef = 2; slot 2 is a null
//	index      0  1     2  3
//	values     a  b     c  d    // = pq.Page.Data()
//
// A seek finds the target row's first slot by counting row starts (rep 0) from the
// current slot, and its index by counting the values (def maxDef) it skips; reading then
// continues from that slot and index.
type slotReader struct {
	maxDef byte // the column's max definition level, set per chunk by SetChunk

	// For the current page, set by load.
	defLevels []byte // nil for required columns
	repLevels []byte // nil for non-repeated columns
	numSlots  int    // one slot per level, or per value when there are no levels
	numValues int    // how many slots hold a value
	slot      int    // next slot, present or null
	index     int    // next present value
}

// load points the reader at the first slot of pg. It fails, and leaves the reader as it
// was, when pg's levels disagree with each other or with maxDef.
func (s *slotReader) load(pg pq.Page) error {
	defLevels, repLevels := pg.DefinitionLevels(), pg.RepetitionLevels()
	if (defLevels == nil) != (s.maxDef == 0) {
		return fmt.Errorf("page has %d definition levels, max definition level is %d", len(defLevels), s.maxDef)
	}
	numSlots := int(pg.NumValues())
	if defLevels != nil {
		numSlots = len(defLevels)
	}
	if repLevels != nil && len(repLevels) != numSlots {
		return fmt.Errorf("page has %d repetition levels and %d slots", len(repLevels), numSlots)
	}
	s.defLevels, s.repLevels, s.numSlots = defLevels, repLevels, numSlots
	s.numValues = countValues(defLevels, s.maxDef, 0, numSlots)
	s.slot, s.index = 0, 0
	return nil
}

// unload forgets the page; maxDef is kept for the chunk's next page.
func (s *slotReader) unload() {
	s.defLevels, s.repLevels, s.numSlots, s.numValues = nil, nil, 0, 0
	s.slot, s.index = 0, 0
}

// hasMoreSlots reports whether the page has a next slot.
func (s *slotReader) hasMoreSlots() bool {
	return s.slot < s.numSlots
}

// peekSlot returns the levels and the present value (-1 for a null) of the next slot,
// without moving past it. The page must have more slots.
func (s *slotReader) peekSlot() (rep, def, index int) {
	rep, def, index = 0, int(s.maxDef), -1
	if s.repLevels != nil {
		rep = int(s.repLevels[s.slot])
	}
	if s.defLevels != nil {
		def = int(s.defLevels[s.slot])
	}
	if byte(def) == s.maxDef {
		index = s.index
	}
	return rep, def, index
}

// nextSlot is peekSlot, and moves past the slot. It is small enough to be inlined
// into the hot loops.
func (s *slotReader) nextSlot() (rep, def, index int) {
	rep, def, index = s.peekSlot()
	s.slot++
	if index >= 0 {
		s.index++
	}
	return rep, def, index
}

// skipToRow moves to the first slot of root row targetRow, and reports whether it did.
// The reader is right after a slot of root row currentRow, or at the start of the page
// with currentRow the row before it. It doesn't move when targetRow is the next row,
// which next reaches anyway, or doesn't start on the page.
func (s *slotReader) skipToRow(currentRow, targetRow int32) bool {
	rowsAhead := int(targetRow - currentRow) // 1 is the next row
	if rowsAhead <= 1 {
		return false
	}
	var targetSlot int
	if s.repLevels != nil {
		// A repeated column: a row spans one or more slots, so find where it starts.
		var ok bool
		if targetSlot, ok = s.findRowStart(rowsAhead); !ok {
			return false
		}
	} else {
		// Not repeated (no repetition levels): every slot is a row of its own.
		targetSlot = s.slot + rowsAhead - 1
	}
	if targetSlot >= s.numSlots {
		return false
	}
	s.index += countValues(s.defLevels, s.maxDef, s.slot, targetSlot) // values skipped
	s.slot = targetSlot
	return true
}

// findRowStart returns the slot where a root row starts, the rows-th one at or after the
// reader's slot: rows=1 is the first row start at or after it. A row starts at each slot
// with repetition level 0, so the column must be repeated. ok is false when the page has
// fewer row starts.
func (s *slotReader) findRowStart(rows int) (slot int, ok bool) {
	repLevels := s.repLevels
	slot = s.slot
	// Skip whole blocks of slots while the row is past them: bytes.Count is vectorized.
	const blockSize = 256
	for slot+blockSize <= len(repLevels) {
		rowStarts := bytes.Count(repLevels[slot:slot+blockSize], []byte{0})
		if rowStarts >= rows {
			break
		}
		rows -= rowStarts
		slot += blockSize
	}
	for ; slot < len(repLevels); slot++ {
		if repLevels[slot] == 0 {
			if rows--; rows == 0 {
				return slot, true
			}
		}
	}
	return 0, false
}

// countValues returns how many of the slots fromSlot..toSlot-1 hold a value rather
// than a null: those at the max definition level. A required column has no definition
// levels: every slot holds a value.
func countValues(defLevels []byte, maxDef byte, fromSlot, toSlot int) int {
	if defLevels == nil {
		return toSlot - fromSlot
	}
	return bytes.Count(defLevels[fromSlot:toSlot], []byte{maxDef})
}
