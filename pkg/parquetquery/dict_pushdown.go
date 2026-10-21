package parquetquery

import (
	pq "github.com/parquet-go/parquet-go"
)

// Dictionary pushdown: on a dictionary-encoded chunk the filter is evaluated once per
// dictionary entry, and rows match by their integer index. A row that doesn't match
// is never turned into a value.
//
// The answers are frozen for the chunk, so a filter whose answer for a value changes
// mid-scan (CallbackPredicate) keeps its answer from the start of the chunk.

// buildDictKeep evaluates the filter against every entry of dict into c.dictKeep and
// reports whether any entry is kept. It reuses c.dictKeep's storage across chunks.
func (c *SyncIterator) buildDictKeep(dict pq.Dictionary) (anyKept bool) {
	n := dict.Len()
	if cap(c.dictKeep) < n {
		c.dictKeep = make([]bool, n)
	}
	c.dictKeep = c.dictKeep[:n]
	for i := range n {
		k := c.filter.KeepValue(dict.Index(int32(i)))
		c.dictKeep[i] = k
		anyKept = anyKept || k
	}
	return anyKept
}

// dictPageReader walks one dictionary-encoded page by its levels and dictionary
// indexes, without decoding values.
type dictPageReader struct {
	active     bool
	dict       pq.Dictionary
	column     int
	indexes    []int32 // dictionary index of each present value
	defLevels  []byte  // nil for required columns
	repLevels  []byte  // nil for non-repeated columns
	presentDef byte    // definition level of a present value on this page
	slot       int     // next level slot, present or null
	index      int     // next entry of indexes
	value      pq.Value
}

// reset points the reader at pg, or deactivates it when pg isn't served by pushdown.
func (r *dictPageReader) reset(pg pq.Page, dictKeep []bool) {
	*r = dictPageReader{value: r.value}
	if pg == nil || dictKeep == nil {
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
	// Present values sit at the page's highest definition level. A page of only nulls
	// has no indexes, which the bounds check in next covers.
	for _, d := range r.defLevels {
		r.presentDef = max(r.presentDef, d)
	}
}

// nextDictValue is next's loop for a dictionary page: it advances c.curr over every
// slot and returns the next kept value, or ok=false once the page is exhausted. The
// value is valid until the next call.
func (c *SyncIterator) nextDictValue(seeking bool) (v *pq.Value, ok bool) {
	r := &c.dictPage
	slots := len(r.indexes)
	if r.defLevels != nil {
		slots = len(r.defLevels)
	}

	for r.slot < slots {
		rep, def := 0, int(r.presentDef)
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
		if byte(def) == r.presentDef && r.index < len(r.indexes) {
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
			if !c.dictKeepNull {
				continue
			}
		} else if !c.dictKeep[index] {
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
