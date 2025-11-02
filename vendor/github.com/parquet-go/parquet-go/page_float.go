package parquet

import (
	"github.com/parquet-go/parquet-go/encoding"
)

type floatPage struct {
	typ         Type
	values      []float32
	columnIndex int16
}

func newFloatPage(typ Type, columnIndex int16, numValues int32, values encoding.Values) *floatPage {
	return &floatPage{
		typ:         typ,
		values:      values.Float()[:numValues],
		columnIndex: ^columnIndex,
	}
}

func (page *floatPage) Type() Type { return page.typ }

func (page *floatPage) Column() int { return int(^page.columnIndex) }

func (page *floatPage) Dictionary() Dictionary { return nil }

func (page *floatPage) NumRows() int64 { return int64(len(page.values)) }

func (page *floatPage) NumValues() int64 { return int64(len(page.values)) }

func (page *floatPage) NumNulls() int64 { return 0 }

func (page *floatPage) Size() int64 { return 4 * int64(len(page.values)) }

func (page *floatPage) RepetitionLevels() []byte { return nil }

func (page *floatPage) DefinitionLevels() []byte { return nil }

func (page *floatPage) Data() encoding.Values { return encoding.FloatValues(page.values) }

func (page *floatPage) Values() ValueReader { return &floatPageValues{page: page} }

func (page *floatPage) min() float32 { return minFloat32(page.values) }

func (page *floatPage) max() float32 { return maxFloat32(page.values) }

func (page *floatPage) bounds() (min, max float32) { return boundsFloat32(page.values) }

func (page *floatPage) Bounds() (min, max Value, ok bool) {
	if ok = len(page.values) > 0; ok {
		minFloat32, maxFloat32 := page.bounds()
		min = page.makeValue(minFloat32)
		max = page.makeValue(maxFloat32)
	}
	return min, max, ok
}

func (page *floatPage) Slice(i, j int64) Page {
	return &floatPage{
		typ:         page.typ,
		values:      page.values[i:j],
		columnIndex: page.columnIndex,
	}
}

func (page *floatPage) makeValue(v float32) Value {
	value := makeValueFloat(v)
	value.columnIndex = page.columnIndex
	return value
}
