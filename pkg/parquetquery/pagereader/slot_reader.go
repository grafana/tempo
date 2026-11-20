package pagereader

import (
	"bytes"
	"fmt"

	pq "github.com/parquet-go/parquet-go"
)

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
