package parquet

import (
	"github.com/parquet-go/parquet-go/encoding"
)

type uint32Page struct {
	typ         Type
	values      []uint32
	columnIndex int16
}

func newUint32Page(typ Type, columnIndex int16, numValues int32, values encoding.Values) *uint32Page {
	return &uint32Page{
		typ:         typ,
		values:      values.Uint32()[:numValues],
		columnIndex: ^columnIndex,
	}
}

func (page *uint32Page) Type() Type { return page.typ }

func (page *uint32Page) Column() int { return int(^page.columnIndex) }

func (page *uint32Page) Dictionary() Dictionary { return nil }

func (page *uint32Page) NumRows() int64 { return int64(len(page.values)) }

func (page *uint32Page) NumValues() int64 { return int64(len(page.values)) }

func (page *uint32Page) NumNulls() int64 { return 0 }

func (page *uint32Page) Size() int64 { return 4 * int64(len(page.values)) }

func (page *uint32Page) RepetitionLevels() []byte { return nil }

func (page *uint32Page) DefinitionLevels() []byte { return nil }

func (page *uint32Page) Data() encoding.Values { return encoding.Uint32Values(page.values) }

func (page *uint32Page) Values() ValueReader { return &uint32PageValues{page: page} }

func (page *uint32Page) min() uint32 { return minUint32(page.values) }

func (page *uint32Page) max() uint32 { return maxUint32(page.values) }

func (page *uint32Page) bounds() (min, max uint32) { return boundsUint32(page.values) }

func (page *uint32Page) Bounds() (min, max Value, ok bool) {
	if ok = len(page.values) > 0; ok {
		minUint32, maxUint32 := page.bounds()
		min = page.makeValue(minUint32)
		max = page.makeValue(maxUint32)
	}
	return min, max, ok
}

func (page *uint32Page) Slice(i, j int64) Page {
	return &uint32Page{
		typ:         page.typ,
		values:      page.values[i:j],
		columnIndex: page.columnIndex,
	}
}

func (page *uint32Page) makeValue(v uint32) Value {
	value := makeValueUint32(v)
	value.columnIndex = page.columnIndex
	return value
}
