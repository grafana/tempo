package parquet

import (
	"github.com/parquet-go/parquet-go/encoding"
)

type int32Page struct {
	typ         Type
	values      []int32
	columnIndex int16
}

func newInt32Page(typ Type, columnIndex int16, numValues int32, values encoding.Values) *int32Page {
	return &int32Page{
		typ:         typ,
		values:      values.Int32()[:numValues],
		columnIndex: ^columnIndex,
	}
}

func (page *int32Page) Type() Type { return page.typ }

func (page *int32Page) Column() int { return int(^page.columnIndex) }

func (page *int32Page) Dictionary() Dictionary { return nil }

func (page *int32Page) NumRows() int64 { return int64(len(page.values)) }

func (page *int32Page) NumValues() int64 { return int64(len(page.values)) }

func (page *int32Page) NumNulls() int64 { return 0 }

func (page *int32Page) Size() int64 { return 4 * int64(len(page.values)) }

func (page *int32Page) RepetitionLevels() []byte { return nil }

func (page *int32Page) DefinitionLevels() []byte { return nil }

func (page *int32Page) Data() encoding.Values { return encoding.Int32Values(page.values) }

func (page *int32Page) Values() ValueReader { return &int32PageValues{page: page} }

func (page *int32Page) min() int32 { return minInt32(page.values) }

func (page *int32Page) max() int32 { return maxInt32(page.values) }

func (page *int32Page) bounds() (min, max int32) { return boundsInt32(page.values) }

func (page *int32Page) Bounds() (min, max Value, ok bool) {
	if ok = len(page.values) > 0; ok {
		minInt32, maxInt32 := page.bounds()
		min = page.makeValue(minInt32)
		max = page.makeValue(maxInt32)
	}
	return min, max, ok
}

func (page *int32Page) Slice(i, j int64) Page {
	return &int32Page{
		typ:         page.typ,
		values:      page.values[i:j],
		columnIndex: page.columnIndex,
	}
}

func (page *int32Page) makeValue(v int32) Value {
	value := makeValueInt32(v)
	value.columnIndex = page.columnIndex
	return value
}
