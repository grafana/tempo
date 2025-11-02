package parquet

import (
	"bytes"

	"github.com/parquet-go/parquet-go/encoding"
)

type byteArrayPage struct {
	typ         Type
	values      []byte
	offsets     []uint32
	columnIndex int16
}

func newByteArrayPage(typ Type, columnIndex int16, numValues int32, values encoding.Values) *byteArrayPage {
	data, offsets := values.ByteArray()
	return &byteArrayPage{
		typ:         typ,
		values:      data,
		offsets:     offsets[:numValues+1],
		columnIndex: ^columnIndex,
	}
}

func (page *byteArrayPage) Type() Type { return page.typ }

func (page *byteArrayPage) Column() int { return int(^page.columnIndex) }

func (page *byteArrayPage) Dictionary() Dictionary { return nil }

func (page *byteArrayPage) NumRows() int64 { return int64(page.len()) }

func (page *byteArrayPage) NumValues() int64 { return int64(page.len()) }

func (page *byteArrayPage) NumNulls() int64 { return 0 }

func (page *byteArrayPage) Size() int64 { return int64(len(page.values)) + 4*int64(len(page.offsets)) }

func (page *byteArrayPage) RepetitionLevels() []byte { return nil }

func (page *byteArrayPage) DefinitionLevels() []byte { return nil }

func (page *byteArrayPage) Data() encoding.Values {
	return encoding.ByteArrayValues(page.values, page.offsets)
}

func (page *byteArrayPage) Values() ValueReader { return &byteArrayPageValues{page: page} }

func (page *byteArrayPage) len() int { return len(page.offsets) - 1 }

func (page *byteArrayPage) index(i int) []byte {
	j := page.offsets[i+0]
	k := page.offsets[i+1]
	return page.values[j:k:k]
}

func (page *byteArrayPage) min() (min []byte) {
	if n := page.len(); n > 0 {
		min = page.index(0)

		for i := 1; i < n; i++ {
			v := page.index(i)

			if bytes.Compare(v, min) < 0 {
				min = v
			}
		}
	}
	return min
}

func (page *byteArrayPage) max() (max []byte) {
	if n := page.len(); n > 0 {
		max = page.index(0)

		for i := 1; i < n; i++ {
			v := page.index(i)

			if bytes.Compare(v, max) > 0 {
				max = v
			}
		}
	}
	return max
}

func (page *byteArrayPage) bounds() (min, max []byte) {
	if n := page.len(); n > 0 {
		min = page.index(0)
		max = min

		for i := 1; i < n; i++ {
			v := page.index(i)

			switch {
			case bytes.Compare(v, min) < 0:
				min = v
			case bytes.Compare(v, max) > 0:
				max = v
			}
		}
	}
	return min, max
}

func (page *byteArrayPage) Bounds() (min, max Value, ok bool) {
	if ok = len(page.offsets) > 1; ok {
		minBytes, maxBytes := page.bounds()
		min = page.makeValueBytes(minBytes)
		max = page.makeValueBytes(maxBytes)
	}
	return min, max, ok
}

func (page *byteArrayPage) cloneValues() []byte {
	values := make([]byte, len(page.values))
	copy(values, page.values)
	return values
}

func (page *byteArrayPage) cloneOffsets() []uint32 {
	offsets := make([]uint32, len(page.offsets))
	copy(offsets, page.offsets)
	return offsets
}

func (page *byteArrayPage) Slice(i, j int64) Page {
	return &byteArrayPage{
		typ:         page.typ,
		values:      page.values,
		offsets:     page.offsets[i : j+1],
		columnIndex: page.columnIndex,
	}
}

func (page *byteArrayPage) makeValueBytes(v []byte) Value {
	value := makeValueBytes(ByteArray, v)
	value.columnIndex = page.columnIndex
	return value
}

func (page *byteArrayPage) makeValueString(v string) Value {
	value := makeValueString(ByteArray, v)
	value.columnIndex = page.columnIndex
	return value
}
