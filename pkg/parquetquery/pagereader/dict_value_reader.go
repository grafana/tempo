package pagereader

import (
	"github.com/grafana/tempo/v3/pkg/parquetquery/base"
	pq "github.com/parquet-go/parquet-go"
)

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
func (r *dictValueReader) nextMatchedValue(c *base.IteratorState, seeking bool) (*pq.Value, bool, error) {
	s := r.slots
	for s.hasMoreSlots() {
		rep, def, index := s.nextSlot()
		c.Row.Next(rep, def, c.MaxDef)

		if seeking && c.BeforeSeekTarget() {
			continue
		}

		c.Stats.InspectedValues++
		keep := r.matchNull // a null has no dictionary entry
		if index >= 0 {
			keep = r.matches[r.indexes[index]]
		}
		if !keep {
			continue
		}
		c.Stats.KeptValues++

		if c.Sampler != nil && !c.Sampler.Sample() {
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
