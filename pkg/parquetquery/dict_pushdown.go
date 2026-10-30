package parquetquery

import (
	"bytes"
	"errors"
	"fmt"

	pq "github.com/parquet-go/parquet-go"
	"github.com/parquet-go/parquet-go/encoding"
)

// pageReader reads every page, both dictionary and plain encoded pages, slot by slot
// from its levels and values. It does so for two reasons:
//
//  1. Dictionary pushdown: on a dictionary-encoded chunk the filter is evaluated once
//     per dictionary entry, and a row is kept or skipped by its dictionary index,
//     without making its value.
//  2. Efficient seeking: a seek within a page walks the page's levels to the target
//     row, instead of reslicing the page (pq.Page.Slice) as parquet-go's value reader
//     would.
//
// This is how it's wired into SyncIterator, for each chunk:
//
//	keepColumnChunk
//	  reset()              forget the previous chunk's matches
//	  matchDictionary()    if the chunk is dictionary-encoded and has a filter: evaluate
//	                       the filter on each entry, and skip the chunk if nothing
//	                       matches
//	setRowGroup
//	  setChunk()           the column's max definition level; fails for a chunk that
//	                       isn't read from a file
//	for each page:
//	  setPage(pg)          open(pg): pushed down when matches is set and pg is
//	                       dictionary-encoded; fails when pg's levels and values disagree
//	  nextMatchedValue()   return the next matching row, until the page is exhausted
//	  closePage()          close()
//	on a seek:
//	  skipToRow()          walk the page's levels to the target row

// How it supports efficient seeking
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

// Reading values. A value is only made into a pq.Value when it is returned or tested
// by the filter: values.Index(indexes[i]) for a dictionary page, values.Index(i) for a
// plain page.
//
// A dictionary-encoded page stores one dictionary index per value. With dictionary
// pushdown the filter's answer for each entry is computed once per chunk, so a slot is
// kept or skipped by its index, and a row that doesn't match is never turned into a
// value. Take an optional string column (max definition level 1) whose chunk has the
// dictionary [api, shop], filtered by = "shop":
//
//	matches = [false, true] // one answer per dictionary entry
//
//	row        0     1     2     3
//	value      shop  null  api   shop
//	def level  1     0     1     1
//	indexes    1           0     1     // = [1, 0, 1]: nulls have no index
//
// For a plain page, pageReader wraps its values in a dictionary (pq.Type.NewDictionary),
// so value i is read as values.Index(i), made the way the page's own value reader makes
// it.
type pageReader struct {
	// For the current chunk, set by matchDictionary, reset and setChunk.
	matches   []bool // the filter's answer for each dictionary entry; nil when the chunk isn't pushed down
	matchNull bool   // the filter's answer for a null
	maxDef    byte   // the column's max definition level: a slot at this level holds a value
	buf       []bool // storage for matches, reused across chunks

	// For the current page, set by open.
	pushdown  bool          // the page is dictionary-encoded and the chunk is pushed down: filter by matches
	values    pq.Dictionary // the page's dictionary, or one over its values for a plain page
	column    int
	indexes   []int32 // dictionary index of each present value; nil for a plain page
	defLevels []byte  // nil for required columns
	repLevels []byte  // nil for non-repeated columns
	slot      int     // next slot, present or null
	index     int     // next present value
	value     pq.Value
	numSlots  int       // one slot per level, or per value when there are no levels
	page      pq.Page   // nil between pages
	pageMax   RowNumber // exclusive: the row number of the next page's first row
}

// matchDictionary evaluates filter against every entry of dict and reports whether
// any entry is kept.
func (r *pageReader) matchDictionary(dict pq.Dictionary, filter Predicate, matchNull bool) (anyMatch bool) {
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

// reset forgets the previous chunk's pushdown, so the next chunk is pushed down only
// once matchDictionary has run for it.
func (r *pageReader) reset() {
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

var errNotFileColumnChunk = errors.New("column chunk isn't read from a file, so its max definition level is unknown")

// setChunk prepares the reader for the pages of cc and forgets the previous page. The
// pushdown set by matchDictionary is kept: keepColumnChunk runs before setChunk. It
// fails when cc isn't read from a file: the max definition level, which tells values
// from nulls, comes from the file's own column.
func (r *pageReader) setChunk(cc pq.ColumnChunk) error {
	r.unload()
	maxDef, ok := fileMaxDefinitionLevel(cc)
	if !ok {
		return errNotFileColumnChunk
	}
	r.maxDef = maxDef
	return nil
}

// open makes pg the current page, its first row right after row number curr, and
// points the reader at its first slot. The previous page must be closed.
func (r *pageReader) open(pg pq.Page, curr RowNumber) error {
	r.page, r.pageMax = pg, curr
	r.pageMax.Skip(pg.NumRows() + 1)
	return r.load(pg)
}

// close releases the current page, if any, and returns the row number of its last row:
// ok is false when there is no page.
func (r *pageReader) close() (last RowNumber, ok bool) {
	r.unload()
	if r.page == nil {
		return RowNumber{}, false
	}
	last = r.pageMax.Preceding()
	pq.Release(r.page)
	r.page, r.pageMax = nil, EmptyRowNumber()
	return last, true
}

// unload forgets the current page's levels and values.
func (r *pageReader) unload() {
	r.pushdown = false
	r.values, r.indexes, r.defLevels, r.repLevels = nil, nil, nil, nil
	r.numSlots, r.slot, r.index = 0, 0, 0
}

// load points the reader at the start of pg. The reader must hold no page: open is
// only called after close. It fails when pg's levels and values disagree, a slot would
// read the wrong value, and then leaves the reader as it was.
//
// pg must be a page as read from its column chunk, never a slice of one
// (pq.Page.Slice). pq.Page.Data shows a page's values from the start of their buffer,
// and the slice of a Boolean page can begin within a byte, at a bit offset Data doesn't
// show, so its values would be read from the wrong bits. Nothing in SyncIterator slices
// pages.
func (r *pageReader) load(pg pq.Page) error {
	defLevels, repLevels := pg.DefinitionLevels(), pg.RepetitionLevels()
	if (defLevels == nil) != (r.maxDef == 0) {
		return fmt.Errorf("page has %d definition levels, max definition level is %d", len(defLevels), r.maxDef)
	}
	numSlots := int(pg.NumValues())
	if defLevels != nil {
		numSlots = len(defLevels)
	}
	if repLevels != nil && len(repLevels) != numSlots {
		return fmt.Errorf("page has %d repetition levels and %d slots", len(repLevels), numSlots)
	}

	numValues := countValues(defLevels, r.maxDef, 0, numSlots)
	data := pg.Data()
	dict := pg.Dictionary()
	if dict != nil && data.Kind() != encoding.Int32 {
		return fmt.Errorf("dictionary page has %v indexes, want int32", data.Kind())
	}
	if err := checkValueCount(data, numValues); err != nil {
		return err
	}
	// Dictionary and plain pages differ only here. A dictionary page's data is one index
	// into its dictionary per value, and its chunk may be pushed down. A plain page's
	// data is the values; a dictionary over them makes value i values.Index(i).
	if dict != nil {
		r.values, r.indexes = dict, data.Int32()
		r.pushdown = r.matches != nil
	} else {
		r.values = pg.Type().NewDictionary(pg.Column(), numValues, data)
	}

	r.column = pg.Column()
	r.defLevels, r.repLevels, r.numSlots = defLevels, repLevels, numSlots
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

// nextMatchedValue is next's loop over the current page: it advances c.curr over every
// slot and returns the next value the filter and sampler keep, or ok=false once the
// page is exhausted. The value is valid until the next call. On the dictionary example:
//
//	slot 0: def 1 = maxDef, takes index 1, matches[1] is true: returns shop (row 0)
//	slot 1: def 0, a null, matchNull is false: skipped
//	slot 2: def 1, takes index 0, matches[0] is false: skipped
//	slot 3: def 1, takes index 1: returns shop (row 3)
func (c *SyncIterator) nextMatchedValue(seeking bool) (v *pq.Value, ok bool) {
	r := &c.pageReader
	for r.hasMoreSlots() {
		rep, def, index := r.nextSlot()
		c.curr.Next(rep, def, c.maxDefinitionLevel)

		if seeking && c.beforeSeekTarget() {
			continue
		}

		if c.filter != nil {
			c.stats.InspectedValues++
			// Decided inline, not in a helper: this is the hot loop, and a call per slot
			// costs more than the decision.
			var keep bool
			switch {
			case !r.pushdown:
				// Not pushed down (a plain page, or a chunk the filter wasn't evaluated
				// on per entry): the filter needs the value itself.
				r.value = r.makeValue(index, rep, def)
				keep = c.filter.KeepValue(r.value)
			case index < 0:
				// Pushed down, a null: it has no dictionary entry, so matchNull decides.
				keep = r.matchNull
			default:
				// Pushed down, a value: the filter's answer for its dictionary entry,
				// computed once per chunk by matchDictionary. No value is made.
				keep = r.matches[r.indexes[index]]
			}
			if !keep {
				continue
			}
			c.stats.KeptValues++
		}
		if c.sampler != nil && !c.sampler.Sample() {
			continue
		}
		if c.filter == nil || r.pushdown { // the value isn't made yet
			r.value = r.makeValue(index, rep, def)
		}
		return &r.value, true
	}
	return nil, false
}

// hasMoreSlots reports whether the page has a next slot.
func (r *pageReader) hasMoreSlots() bool {
	return r.slot < r.numSlots
}

// peekSlot returns the levels and the present value (-1 for a null) of the next slot,
// without moving past it. The page must have more slots.
func (r *pageReader) peekSlot() (rep, def, index int) {
	rep, def, index = 0, int(r.maxDef), -1
	if r.repLevels != nil {
		rep = int(r.repLevels[r.slot])
	}
	if r.defLevels != nil {
		def = int(r.defLevels[r.slot])
	}
	if byte(def) == r.maxDef {
		index = r.index
	}
	return rep, def, index
}

// nextSlot is peekSlot, and moves past the slot. It is small enough to be inlined
// into the hot loops.
func (r *pageReader) nextSlot() (rep, def, index int) {
	rep, def, index = r.peekSlot()
	r.slot++
	if index >= 0 {
		r.index++
	}
	return rep, def, index
}

// makeValue returns the value of a slot with levels rep and def. index is the slot's
// present value, or -1 for a null.
func (r *pageReader) makeValue(index, rep, def int) pq.Value {
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

// skipToRow moves the reader to the first slot of root row targetRow, and reports
// whether it did. The reader is right after a slot of root row currentRow, or at the
// start of the page with currentRow the row before it. It doesn't move when targetRow
// is the next row, which next reaches anyway, or doesn't start on the page.
func (r *pageReader) skipToRow(currentRow, targetRow int32) bool {
	rowsAhead := int(targetRow - currentRow) // 1 is the next row
	if rowsAhead <= 1 {
		return false
	}
	var targetSlot int
	if r.repLevels != nil {
		// A repeated column: a row spans one or more slots, so find where it starts.
		var ok bool
		if targetSlot, ok = r.findRowStart(rowsAhead); !ok {
			return false
		}
	} else {
		// Not repeated (no repetition levels): every slot is a row of its own.
		targetSlot = r.slot + rowsAhead - 1
	}
	if targetSlot >= r.numSlots {
		return false
	}
	r.index += countValues(r.defLevels, r.maxDef, r.slot, targetSlot) // values skipped
	r.slot = targetSlot
	return true
}

// findRowStart returns the slot where a root row starts, the rows-th one at or after the
// reader's slot: rows=1 is the first row start at or after it. A row starts at each slot
// with repetition level 0, so the column must be repeated. ok is false when the page has
// fewer row starts.
func (r *pageReader) findRowStart(rows int) (slot int, ok bool) {
	repLevels := r.repLevels
	slot = r.slot
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
