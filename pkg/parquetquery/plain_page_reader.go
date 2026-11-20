package parquetquery

import (
	"bytes"

	pq "github.com/parquet-go/parquet-go"
	"github.com/parquet-go/parquet-go/encoding"
)

// Seeking within a page by its levels: a page served by dictPageReader or
// plainPageReader is read slot by slot from its definition and repetition levels, so a
// seek within it only has to walk the levels to the target row (skipToRow). The value
// reader (pq.Page.Values) can't skip, so seekWithinPage reslices its page instead, and
// pq.Page.Slice scans the remaining levels of a repeated page on every call and copies
// the remaining values of an int64 or uint64 page.
//
// This is how plainPageReader is wired into SyncIterator, for each page:
//
//	setPage(pg)            the reader is off: next reads pg through its value reader,
//	                       which is faster for a scan
//	on a long seek:
//	  startPlainPageReader start the reader at the next slot, if it can read pg,
//	                       otherwise seekWithinPage reslices pg as before
//	then, and on any later seek:
//	  skipToRow            walk the levels to the target row
//	nextPlainValue()       return the next kept value, until the page is exhausted
//	setPage(nil)

// plainPageReader serves a page from its levels and its values (pq.Page.Data), the way
// dictPageReader serves a dictionary-encoded page from its indexes. It reads pages of
// any physical type, plain or dictionary-encoded (those that dictPageReader doesn't
// serve, mostly because they have no filter). A value is only made into a pq.Value
// when it is at or after the seek target.
//
// The values of a plain page are read through a dictionary of the page's type over
// them (pq.Type.NewDictionary), which makes each value the way the page's own value
// reader does: for example a uint32 page has kind Int32, but its values aren't the
// ones pq.Int32Value makes.
type plainPageReader struct {
	disabled bool // never starts: NilSyncIterator reads the value reader itself
	tried    bool // start was called for the current page

	// For the current page, set by start.
	active    bool
	column    int
	maxDef    byte          // the column's max definition level: a slot at this level holds a value
	values    pq.Dictionary // the page's dictionary, or one over its values for a plain page
	indexes   []int32       // dictionary index of each present value; nil for a plain page
	defLevels []byte        // nil for required columns
	repLevels []byte        // nil for non-repeated columns
	numSlots  int           // one slot per level, or per value when there are no levels
	slot      int           // next slot, present or null
	index     int           // next present value
	value     pq.Value
}

// reset forgets the current page.
func (r *plainPageReader) reset() {
	*r = plainPageReader{disabled: r.disabled}
}

// start points the reader at slot of pg, and reports whether it can read pg. maxDef is
// the column's max definition level in the file, as for dictPageReader. start tries a
// page only once: a page it turns down is resliced, and the slice of a Boolean page can
// begin within a byte, which pq.Page.Data doesn't show.
func (r *plainPageReader) start(pg pq.Page, maxDef byte, slot int) bool {
	if r.disabled || r.tried {
		return false
	}
	if !r.load(pg, maxDef, slot) {
		*r = plainPageReader{}
	}
	r.tried = true
	return r.active
}

// load is start without its checks: it fills the reader from pg and reports whether
// pg can be read.
func (r *plainPageReader) load(pg pq.Page, maxDef byte, slot int) bool {
	defLevels, repLevels := pg.DefinitionLevels(), pg.RepetitionLevels()
	if (defLevels == nil) != (maxDef == 0) {
		return false
	}
	numSlots := int(pg.NumValues())
	if defLevels != nil {
		numSlots = len(defLevels)
	}
	if (repLevels != nil && len(repLevels) != numSlots) || slot > numSlots {
		return false
	}

	// The levels must agree with the values, or a slot would read the wrong value.
	present := countPresent(defLevels, maxDef, 0, numSlots)
	data := pg.Data()
	if dict := pg.Dictionary(); dict != nil {
		if data.Kind() != encoding.Int32 || len(data.Int32()) != present {
			return false
		}
		r.values, r.indexes = dict, data.Int32()
	} else {
		if !hasValues(data, present) {
			return false
		}
		r.values = pg.Type().NewDictionary(pg.Column(), present, data)
	}

	r.active = true
	r.column = pg.Column()
	r.maxDef = maxDef
	r.defLevels, r.repLevels, r.numSlots = defLevels, repLevels, numSlots
	r.slot, r.index = slot, countPresent(defLevels, maxDef, 0, slot)
	return true
}

// hasValues reports whether data holds exactly n values, or, for booleans, at least n.
func hasValues(data encoding.Values, n int) bool {
	switch data.Kind() {
	case encoding.Boolean:
		return len(data.Boolean())*8 >= n
	case encoding.Int32:
		return len(data.Int32()) == n
	case encoding.Int64:
		return len(data.Int64()) == n
	case encoding.Int96:
		return len(data.Int96()) == n
	case encoding.Float:
		return len(data.Float()) == n
	case encoding.Double:
		return len(data.Double()) == n
	case encoding.ByteArray:
		_, offsets := data.ByteArray()
		return len(offsets) == n+1
	case encoding.FixedLenByteArray:
		b, size := data.FixedLenByteArray()
		return size > 0 && len(b) == n*size
	default:
		return false
	}
}

// makeValue returns the value of a slot with levels rep and def. index is the slot's
// present value, or -1 for a null.
func (r *plainPageReader) makeValue(index, rep, def int) pq.Value {
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

// skipToRow moves the reader to the first slot of root row to, see skipToRow.
func (r *plainPageReader) skipToRow(cur, to int32) bool {
	return skipToRow(r.defLevels, r.repLevels, r.maxDef, r.numSlots, &r.slot, &r.index, cur, to)
}

// startPlainPageReader switches the current page from the value reader to
// plainPageReader, at the slot next would read next, and reports whether it did.
func (c *SyncIterator) startPlainPageReader() bool {
	maxDef, ok := fileMaxDefinitionLevel(c.currChunk.ColumnChunk)
	if !ok || !c.plainPageReader.start(c.currPage, maxDef, c.currPageN) {
		return false
	}
	// Return the value buffer, if any: returning nil would put an empty slice in the
	// pool, and the next Get would allocate a new buffer.
	if c.currBuf != nil {
		syncIteratorPoolPut(c.currBuf)
		c.currBuf = nil
	}
	c.currBufN = 0
	c.currValues = nil
	return true
}

// nextPlainValue is next's loop for a page served by plainPageReader, like
// nextMatchedValue: it advances c.curr over every slot and returns the next kept value,
// or ok=false once the page is exhausted. The value is valid until the next call.
func (c *SyncIterator) nextPlainValue(seeking bool) (v *pq.Value, ok bool) {
	r := &c.plainPageReader
	for r.slot < r.numSlots {
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

		index := -1 // -1 marks a null
		if byte(def) == r.maxDef {
			index = r.index
			r.index++
		}

		if seeking && c.beforeSeekTarget() {
			continue
		}

		r.value = r.makeValue(index, rep, def)
		if c.filter != nil {
			if c.stats != nil {
				c.stats.InspectedValues++
			}
			if !c.filter.KeepValue(r.value) {
				continue
			}
			if c.stats != nil {
				c.stats.KeptValues++
			}
		}

		if c.sampler != nil && !c.sampler.Sample() {
			continue
		}
		return &r.value, true
	}
	return nil, false
}

// skipToRow moves a reader's slot to the first slot of root row to. The reader is right
// after a slot of row cur, or at the start of the page with cur the row before it, and
// index counts the present values before slot. It reports false, leaving slot and index
// alone, when row to doesn't start on the page after slot.
func skipToRow(defLevels, repLevels []byte, maxDef byte, numSlots int, slot, index *int, cur, to int32) bool {
	if to <= cur {
		return false
	}
	s := *slot
	if repLevels == nil {
		s += int(to - cur - 1) // one slot per row
	} else {
		var ok bool
		if s, ok = nthRowStart(repLevels, s, int(to-cur)); !ok {
			return false
		}
	}
	if s >= numSlots {
		return false
	}
	*index += countPresent(defLevels, maxDef, *slot, s)
	*slot = s
	return true
}

// nthRowStart returns the slot of the nth row start (repetition level 0) at or after
// slot s.
func nthRowStart(repLevels []byte, s, n int) (int, bool) {
	// Count whole blocks at a time: bytes.Count is vectorized.
	const block = 256
	for s+block <= len(repLevels) {
		c := bytes.Count(repLevels[s:s+block], []byte{0})
		if c >= n {
			break
		}
		n -= c
		s += block
	}
	for ; s < len(repLevels); s++ {
		if repLevels[s] == 0 {
			if n--; n == 0 {
				return s, true
			}
		}
	}
	return 0, false
}

// countPresent returns how many of the slots [from, to) hold a value. defLevels is nil
// for a required column, where every slot does.
func countPresent(defLevels []byte, maxDef byte, from, to int) int {
	if defLevels == nil {
		return to - from
	}
	return bytes.Count(defLevels[from:to], []byte{maxDef})
}
