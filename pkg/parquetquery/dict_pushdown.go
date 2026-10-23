package parquetquery

import (
	pq "github.com/parquet-go/parquet-go"
)

// Dictionary pushdown: on a dictionary-encoded chunk the filter is evaluated once per
// dictionary entry, and rows match by their integer index. A row that doesn't match
// is never turned into a value.
//
// This is how it's wired into SyncIterator, for each chunk:
//
//	keepColumnChunk
//	  reset()              forget the previous chunk's matches
//	  matchDictionary()    if the chunk is dictionary-encoded and has a filter: evaluate
//	                       the filter on each entry, and skip the chunk if nothing
//	                       matches; matches is kept only for a chunk read from a file
//	for each page:
//	  setPage(pg)          active when matches is set and pg is dictionary-encoded,
//	                       otherwise next reads the page value by value
//	  nextMatchedValue()   return the next matching row, until the page is exhausted
//	  setPage(nil)
//	on a long seek:
//	  setPage(slice)       seekWithinPage reslices the page and restarts the reader on it

// Example of how dictPageReader works: it serves dictionary-encoded pages from their
// levels and dictionary indexes, without decoding values. Take an optional string
// column (max definition level 1) whose chunk has the dictionary [api, shop], filtered
// by = "shop":
//
//	matches = [false, true] // one answer per dictionary entry
//
//	row        0     1     2     3
//	value      shop  null  api   shop
//	def level  1     0     1     1
//	indexes    1           0     1     // = [1, 0, 1]: nulls have no index
//
// A page has a level for every slot, present or null, but an index only for each
// present value. So the reader walks the slots and takes the next index only when the
// slot's definition level is maxDef.
type dictPageReader struct {
	// For the current chunk, set by matchDictionary and cleared by reset.
	matches   []bool // the filter's answer for each dictionary entry; nil when the chunk isn't pushed down
	matchNull bool   // the filter's answer for a null
	maxDef    byte   // the column's max definition level: a slot at this level holds a value
	buf       []bool // storage for matches, reused across chunks

	// For the current page, set by setPage.
	active    bool
	dict      pq.Dictionary
	column    int
	indexes   []int32 // dictionary index of each present value
	defLevels []byte  // nil for required columns
	repLevels []byte  // nil for non-repeated columns
	slot      int     // next slot, present or null
	index     int     // next entry of indexes
	value     pq.Value
}

// matchDictionary evaluates filter against every entry of dict and reports whether
// any entry is kept. The chunk is pushed down only when its definition levels can be
// read from the file's own column.
func (r *dictPageReader) matchDictionary(dict pq.Dictionary, cc pq.ColumnChunk, filter Predicate, matchNull bool) (anyMatch bool) {
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

	if maxDef, ok := fileMaxDefinitionLevel(cc); ok {
		r.matches, r.matchNull, r.maxDef = matches, matchNull, maxDef
	}
	return anyMatch
}

// reset forgets the previous chunk's results, so the next chunk is pushed down only
// once matchDictionary has run for it.
func (r *dictPageReader) reset() {
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

// setPage points the reader at pg, or deactivates it when pg isn't pushed down.
func (r *dictPageReader) setPage(pg pq.Page) {
	r.active, r.dict, r.indexes, r.defLevels, r.repLevels, r.slot, r.index = false, nil, nil, nil, nil, 0, 0
	if pg == nil || r.matches == nil {
		return
	}
	dict := pg.Dictionary()
	if dict == nil {
		return
	}
	r.active = true
	r.dict = dict
	r.column = pg.Column()
	data := pg.Data()
	r.indexes = data.Int32()
	r.defLevels = pg.DefinitionLevels()
	r.repLevels = pg.RepetitionLevels()
}

// nextMatchedValue is next's loop for a pushed-down page: it advances c.curr over every
// slot and returns the next kept value, or ok=false once the page is exhausted. The
// value is valid until the next call. On the dictPageReader example:
//
//	slot 0: def 1 = maxDef, takes index 1, matches[1] is true: returns shop (row 0)
//	slot 1: def 0, a null, matchNull is false: skipped
//	slot 2: def 1, takes index 0, matches[0] is false: skipped
//	slot 3: def 1, takes index 1: returns shop (row 3)
func (c *SyncIterator) nextMatchedValue(seeking bool) (v *pq.Value, ok bool) {
	r := &c.dictPageReader
	slots := len(r.indexes) // a required column has no levels: every slot is present
	if r.defLevels != nil {
		slots = len(r.defLevels)
	}

	for r.slot < slots {
		rep, def := 0, int(r.maxDef)
		if r.repLevels != nil {
			rep = int(r.repLevels[r.slot])
		}
		if r.defLevels != nil {
			def = int(r.defLevels[r.slot])
		}
		r.slot++
		c.curr.Next(rep, def, c.maxDefinitionLevel)
		c.currPageN++

		index := int32(-1) // -1 marks a null
		if byte(def) == r.maxDef && r.index < len(r.indexes) {
			index = r.indexes[r.index]
			r.index++
		}

		if seeking && c.beforeSeekTarget() {
			continue
		}

		if c.stats != nil {
			c.stats.InspectedValues++
		}
		if index < 0 {
			if !r.matchNull {
				continue
			}
		} else if !r.matches[index] {
			continue
		}
		if c.stats != nil {
			c.stats.KeptValues++
		}

		if c.sampler != nil && !c.sampler.Sample() {
			continue
		}

		if index < 0 {
			r.value = pq.Value{}.Level(rep, def, r.column)
		} else {
			r.value = r.dict.Index(index).Level(rep, def, r.column)
		}
		return &r.value, true
	}
	return nil, false
}
