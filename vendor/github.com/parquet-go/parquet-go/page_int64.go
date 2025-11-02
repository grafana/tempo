package parquet

import (
	"github.com/parquet-go/parquet-go/encoding"
)

type int64Page struct {
	typ         Type
	values      []int64
	columnIndex int16
}

func newInt64Page(typ Type, columnIndex int16, numValues int32, values encoding.Values) *int64Page {
	return &int64Page{
		typ:         typ,
		values:      values.Int64()[:numValues],
		columnIndex: ^columnIndex,
	}
}

func (page *int64Page) Type() Type { return page.typ }

func (page *int64Page) Column() int { return int(^page.columnIndex) }

func (page *int64Page) Dictionary() Dictionary { return nil }

func (page *int64Page) NumRows() int64 { return int64(len(page.values)) }

func (page *int64Page) NumValues() int64 { return int64(len(page.values)) }

func (page *int64Page) NumNulls() int64 { return 0 }

func (page *int64Page) Size() int64 { return 8 * int64(len(page.values)) }

func (page *int64Page) RepetitionLevels() []byte { return nil }

func (page *int64Page) DefinitionLevels() []byte { return nil }

func (page *int64Page) Data() encoding.Values { return encoding.Int64Values(page.values) }

func (page *int64Page) Values() ValueReader { return &int64PageValues{page: page} }

func (page *int64Page) min() int64 { return minInt64(page.values) }

func (page *int64Page) max() int64 { return maxInt64(page.values) }

func (page *int64Page) bounds() (min, max int64) { return boundsInt64(page.values) }

func (page *int64Page) Bounds() (min, max Value, ok bool) {
	if ok = len(page.values) > 0; ok {
		minInt64, maxInt64 := page.bounds()
		min = page.makeValue(minInt64)
		max = page.makeValue(maxInt64)
	}
	return min, max, ok
}

func (page *int64Page) Slice(i, j int64) Page {
	return &int64Page{
		typ:         page.typ,
		values:      page.values[i:j],
		columnIndex: page.columnIndex,
	}
}

func (page *int64Page) makeValue(v int64) Value {
	value := makeValueInt64(v)
	value.columnIndex = page.columnIndex
	return value
}
