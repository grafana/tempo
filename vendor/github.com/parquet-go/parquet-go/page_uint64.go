package parquet

import (
	"github.com/parquet-go/parquet-go/encoding"
)

type uint64Page struct {
	typ         Type
	values      []uint64
	columnIndex int16
}

func newUint64Page(typ Type, columnIndex int16, numValues int32, values encoding.Values) *uint64Page {
	return &uint64Page{
		typ:         typ,
		values:      values.Uint64()[:numValues],
		columnIndex: ^columnIndex,
	}
}

func (page *uint64Page) Type() Type { return page.typ }

func (page *uint64Page) Column() int { return int(^page.columnIndex) }

func (page *uint64Page) Dictionary() Dictionary { return nil }

func (page *uint64Page) NumRows() int64 { return int64(len(page.values)) }

func (page *uint64Page) NumValues() int64 { return int64(len(page.values)) }

func (page *uint64Page) NumNulls() int64 { return 0 }

func (page *uint64Page) Size() int64 { return 8 * int64(len(page.values)) }

func (page *uint64Page) RepetitionLevels() []byte { return nil }

func (page *uint64Page) DefinitionLevels() []byte { return nil }

func (page *uint64Page) Data() encoding.Values { return encoding.Uint64Values(page.values) }

func (page *uint64Page) Values() ValueReader { return &uint64PageValues{page: page} }

func (page *uint64Page) min() uint64 { return minUint64(page.values) }

func (page *uint64Page) max() uint64 { return maxUint64(page.values) }

func (page *uint64Page) bounds() (min, max uint64) { return boundsUint64(page.values) }

func (page *uint64Page) Bounds() (min, max Value, ok bool) {
	if ok = len(page.values) > 0; ok {
		minUint64, maxUint64 := page.bounds()
		min = page.makeValue(minUint64)
		max = page.makeValue(maxUint64)
	}
	return min, max, ok
}

func (page *uint64Page) Slice(i, j int64) Page {
	return &uint64Page{
		typ:         page.typ,
		values:      page.values[i:j],
		columnIndex: page.columnIndex,
	}
}

func (page *uint64Page) makeValue(v uint64) Value {
	value := makeValueUint64(v)
	value.columnIndex = page.columnIndex
	return value
}
