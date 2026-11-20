package pagereader

import (
	"github.com/grafana/tempo/v3/pkg/parquetquery/base"
	pq "github.com/parquet-go/parquet-go"
)

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
func (r *seekableValueReader) nextMatchedValue(c *base.IteratorState, seeking bool) (*pq.Value, bool, error) {
	s := r.slots
	for s.hasMoreSlots() {
		rep, def, index := s.nextSlot()
		c.Row.Next(rep, def, c.MaxDef)

		if seeking && c.BeforeSeekTarget() {
			continue
		}

		made := false
		if c.Filter != nil {
			c.Stats.InspectedValues++
			r.value, made = r.makeValue(index, rep, def), true
			if !c.Filter.KeepValue(r.value) {
				continue
			}
			c.Stats.KeptValues++
		}
		if c.Sampler != nil && !c.Sampler.Sample() {
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
