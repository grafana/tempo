package parquet

import (
	"github.com/parquet-go/parquet-go/encoding"
)

type fixedLenByteArrayPage struct {
	typ         Type
	data        []byte
	size        int
	columnIndex int16
}

func newFixedLenByteArrayPage(typ Type, columnIndex int16, numValues int32, values encoding.Values) *fixedLenByteArrayPage {
	data, size := values.FixedLenByteArray()
	return &fixedLenByteArrayPage{
		typ:         typ,
		data:        data[:int(numValues)*size],
		size:        size,
		columnIndex: ^columnIndex,
	}
}

func (page *fixedLenByteArrayPage) Type() Type { return page.typ }

func (page *fixedLenByteArrayPage) Column() int { return int(^page.columnIndex) }

func (page *fixedLenByteArrayPage) Dictionary() Dictionary { return nil }

func (page *fixedLenByteArrayPage) NumRows() int64 { return int64(len(page.data) / page.size) }

func (page *fixedLenByteArrayPage) NumValues() int64 { return int64(len(page.data) / page.size) }

func (page *fixedLenByteArrayPage) NumNulls() int64 { return 0 }

func (page *fixedLenByteArrayPage) Size() int64 { return int64(len(page.data)) }

func (page *fixedLenByteArrayPage) RepetitionLevels() []byte { return nil }

func (page *fixedLenByteArrayPage) DefinitionLevels() []byte { return nil }

func (page *fixedLenByteArrayPage) Data() encoding.Values {
	return encoding.FixedLenByteArrayValues(page.data, page.size)
}

func (page *fixedLenByteArrayPage) Values() ValueReader {
	return &fixedLenByteArrayPageValues{page: page}
}

func (page *fixedLenByteArrayPage) min() []byte { return minFixedLenByteArray(page.data, page.size) }

func (page *fixedLenByteArrayPage) max() []byte { return maxFixedLenByteArray(page.data, page.size) }

func (page *fixedLenByteArrayPage) bounds() (min, max []byte) {
	return boundsFixedLenByteArray(page.data, page.size)
}

func (page *fixedLenByteArrayPage) Bounds() (min, max Value, ok bool) {
	if ok = len(page.data) > 0; ok {
		minBytes, maxBytes := page.bounds()
		min = page.makeValueBytes(minBytes)
		max = page.makeValueBytes(maxBytes)
	}
	return min, max, ok
}

func (page *fixedLenByteArrayPage) Slice(i, j int64) Page {
	return &fixedLenByteArrayPage{
		typ:         page.typ,
		data:        page.data[i*int64(page.size) : j*int64(page.size)],
		size:        page.size,
		columnIndex: page.columnIndex,
	}
}

func (page *fixedLenByteArrayPage) makeValueBytes(v []byte) Value {
	value := makeValueBytes(FixedLenByteArray, v)
	value.columnIndex = page.columnIndex
	return value
}

func (page *fixedLenByteArrayPage) makeValueString(v string) Value {
	value := makeValueString(FixedLenByteArray, v)
	value.columnIndex = page.columnIndex
	return value
}
