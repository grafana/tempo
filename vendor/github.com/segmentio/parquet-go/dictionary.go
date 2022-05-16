package parquet

import (
	"bytes"
	"fmt"
	"io"

	"github.com/segmentio/parquet-go/encoding"
	"github.com/segmentio/parquet-go/internal/bits"
)

const (
	// Completely arbitrary, feel free to adjust if a different value would be
	// more representative of the map implementation in Go.
	mapSizeOverheadPerItem = 8
)

// The Dictionary interface represents type-specific implementations of parquet
// dictionaries.
//
// Programs can instantiate dictionaries by call the NewDictionary method of a
// Type object.
type Dictionary interface {
	// Returns the type that the dictionary was created from.
	Type() Type

	// Returns the number of value indexed in the dictionary.
	Len() int

	// Returns the dictionary value at the given index.
	Index(index int32) Value

	// Inserts values from the second slice to the dictionary and writes the
	// indexes at which each value was inserted to the first slice.
	//
	// The method panics if the length of the indexes slice is smaller than the
	// length of the values slice.
	Insert(indexes []int32, values []Value)

	// Given an array of dictionary indexes, lookup the values into the array
	// of values passed as second argument.
	//
	// The method panics if len(indexes) > len(values), or one of the indexes
	// is negative or greater than the highest index in the dictionary.
	Lookup(indexes []int32, values []Value)

	// Returns the min and max values found in the given indexes.
	Bounds(indexed []int32) (min, max Value)

	// Resets the dictionary to its initial state, removing all values.
	Reset()

	// Returns a BufferedPage representing the content of the dictionary.
	//
	// The returned page shares the underlying memory of the buffer, it remains
	// valid to use until the dictionary's Reset method is called.
	Page() BufferedPage
}

func dictCap(bufferSize, valueItemSize int) int {
	indexItemSize := 4 + valueItemSize + mapSizeOverheadPerItem
	return atLeastOne(bufferSize / (valueItemSize + indexItemSize))
}

type byteArrayDictionary struct {
	byteArrayPage
	typ   Type
	index map[string]int32
}

func newByteArrayDictionary(typ Type, columnIndex int16, bufferSize int) *byteArrayDictionary {
	return &byteArrayDictionary{
		typ: typ,
		byteArrayPage: byteArrayPage{
			values:      encoding.MakeByteArrayList(dictCap(bufferSize, 16)),
			columnIndex: ^columnIndex,
		},
	}
}

func readByteArrayDictionary(typ Type, columnIndex int16, numValues int, decoder encoding.Decoder) (Dictionary, error) {
	d := &byteArrayDictionary{
		typ: typ,
		byteArrayPage: byteArrayPage{
			values:      encoding.MakeByteArrayList(atLeastOne(numValues)),
			columnIndex: ^columnIndex,
		},
	}

	for {
		if d.values.Len() == d.values.Cap() {
			d.values.Grow(d.values.Len())
		}
		_, err := decoder.DecodeByteArray(&d.values)
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return d, err
		}
	}
}

func (d *byteArrayDictionary) Type() Type { return newIndexedType(d.typ, d) }

func (d *byteArrayDictionary) Len() int { return d.values.Len() }

func (d *byteArrayDictionary) Index(i int32) Value {
	return makeValueBytes(ByteArray, d.values.Index(int(i)))
}

func (d *byteArrayDictionary) Insert(indexes []int32, values []Value) {
	_ = indexes[:len(values)]

	if d.index == nil {
		index := int32(0)
		d.index = make(map[string]int32, d.values.Cap())
		d.values.Range(func(v []byte) bool {
			d.index[bits.BytesToString(v)] = index
			index++
			return true
		})
	}

	for i, v := range values {
		value := v.ByteArray()

		index, exists := d.index[string(value)]
		if !exists {
			d.values.Push(value)
			index = int32(d.values.Len() - 1)
			stringValue := bits.BytesToString(d.values.Index(int(index)))
			d.index[stringValue] = index
		}

		indexes[i] = index
	}
}

func (d *byteArrayDictionary) Lookup(indexes []int32, values []Value) {
	for i, j := range indexes {
		values[i] = d.Index(j)
	}
}

func (d *byteArrayDictionary) Bounds(indexes []int32) (min, max Value) {
	if len(indexes) > 0 {
		minValue := d.values.Index(int(indexes[0]))
		maxValue := minValue

		for _, i := range indexes[1:] {
			value := d.values.Index(int(i))
			switch {
			case bytes.Compare(value, minValue) < 0:
				minValue = value
			case bytes.Compare(value, maxValue) > 0:
				maxValue = value
			}
		}

		min = makeValueBytes(ByteArray, minValue)
		max = makeValueBytes(ByteArray, maxValue)
	}
	return min, max
}

func (d *byteArrayDictionary) Reset() {
	d.values.Reset()
	d.index = nil
}

func (d *byteArrayDictionary) Page() BufferedPage {
	return &d.byteArrayPage
}

type fixedLenByteArrayDictionary struct {
	fixedLenByteArrayPage
	typ   Type
	index map[string]int32
}

func newFixedLenByteArrayDictionary(typ Type, columnIndex int16, bufferSize int) *fixedLenByteArrayDictionary {
	size := typ.Length()
	return &fixedLenByteArrayDictionary{
		typ: typ,
		fixedLenByteArrayPage: fixedLenByteArrayPage{
			size:        size,
			data:        make([]byte, 0, dictCap(bufferSize, size)*size),
			columnIndex: ^columnIndex,
		},
	}
}

func readFixedLenByteArrayDictionary(typ Type, columnIndex int16, numValues int, decoder encoding.Decoder) (Dictionary, error) {
	size := typ.Length()

	d := &fixedLenByteArrayDictionary{
		typ: typ,
		fixedLenByteArrayPage: fixedLenByteArrayPage{
			size:        size,
			data:        make([]byte, 0, atLeastOne(numValues)*size),
			columnIndex: ^columnIndex,
		},
	}

	for {
		if len(d.data) == cap(d.data) {
			newValues := make([]byte, len(d.data), 2*cap(d.data))
			copy(newValues, d.data)
			d.data = newValues
		}

		n, err := decoder.DecodeFixedLenByteArray(d.size, d.data[len(d.data):cap(d.data)])
		if n > 0 {
			d.data = d.data[:len(d.data)+(n*d.size)]
		}

		if err == io.EOF {
			return d, nil
		}
		if err != nil {
			return nil, fmt.Errorf("reading parquet dictionary of fixed-length binary values of size %d: %w", d.size, err)
		}
	}

}

func (d *fixedLenByteArrayDictionary) Type() Type { return newIndexedType(d.typ, d) }

func (d *fixedLenByteArrayDictionary) Len() int { return len(d.data) / d.size }

func (d *fixedLenByteArrayDictionary) Index(i int32) Value {
	return makeValueBytes(FixedLenByteArray, d.value(i))
}

func (d *fixedLenByteArrayDictionary) value(i int32) []byte {
	return d.data[int(i)*d.size : int(i+1)*d.size]
}

func (d *fixedLenByteArrayDictionary) Insert(indexes []int32, values []Value) {
	_ = indexes[:len(values)]

	if d.index == nil {
		d.index = make(map[string]int32, cap(d.data)/d.size)
		for i, j := 0, int32(0); i < len(d.data); i += d.size {
			d.index[bits.BytesToString(d.data[i:i+d.size])] = j
			j++
		}
	}

	for i, v := range values {
		value := v.ByteArray()

		index, exists := d.index[string(value)]
		if !exists {
			index = int32(d.Len())
			start := len(d.data)
			d.data = append(d.data, value...)
			d.index[bits.BytesToString(d.data[start:])] = index
		}

		indexes[i] = index
	}
}

func (d *fixedLenByteArrayDictionary) Lookup(indexes []int32, values []Value) {
	for i, j := range indexes {
		values[i] = d.Index(j)
	}
}

func (d *fixedLenByteArrayDictionary) Bounds(indexes []int32) (min, max Value) {
	if len(indexes) > 0 {
		minValue := d.value(indexes[0])
		maxValue := minValue

		for _, i := range indexes[1:] {
			value := d.value(i)
			switch {
			case bytes.Compare(value, minValue) < 0:
				minValue = value
			case bytes.Compare(value, maxValue) > 0:
				maxValue = value
			}
		}

		min = makeValueBytes(FixedLenByteArray, minValue)
		max = makeValueBytes(FixedLenByteArray, maxValue)
	}
	return min, max
}

func (d *fixedLenByteArrayDictionary) Reset() {
	d.data = d.data[:0]
	d.index = nil
}

func (d *fixedLenByteArrayDictionary) Page() BufferedPage {
	return &d.fixedLenByteArrayPage
}

type indexedType struct {
	Type
	dict Dictionary
}

func newIndexedType(typ Type, dict Dictionary) *indexedType {
	return &indexedType{Type: typ, dict: dict}
}

func (t *indexedType) NewColumnBuffer(columnIndex, bufferSize int) ColumnBuffer {
	return newIndexedColumnBuffer(t.dict, t, makeColumnIndex(columnIndex), bufferSize)
}

func (t *indexedType) NewColumnReader(columnIndex, bufferSize int) ColumnReader {
	return newIndexedColumnReader(t.dict, t, makeColumnIndex(columnIndex), bufferSize)
}

type indexedPage struct {
	dict        Dictionary
	values      []int32
	columnIndex int16
}

func (page *indexedPage) Column() int { return int(^page.columnIndex) }

func (page *indexedPage) Dictionary() Dictionary { return page.dict }

func (page *indexedPage) NumRows() int64 { return int64(len(page.values)) }

func (page *indexedPage) NumValues() int64 { return int64(len(page.values)) }

func (page *indexedPage) NumNulls() int64 { return 0 }

func (page *indexedPage) Bounds() (min, max Value, ok bool) {
	if ok = len(page.values) > 0; ok {
		min, max = page.dict.Bounds(page.values)
		min.columnIndex = page.columnIndex
		max.columnIndex = page.columnIndex
	}
	return min, max, ok
}

func (page *indexedPage) Clone() BufferedPage {
	return &indexedPage{
		dict:        page.dict,
		values:      append([]int32{}, page.values...),
		columnIndex: page.columnIndex,
	}
}

func (page *indexedPage) Slice(i, j int64) BufferedPage {
	return &indexedPage{
		dict:        page.dict,
		values:      page.values[i:j],
		columnIndex: page.columnIndex,
	}
}

func (page *indexedPage) Size() int64 { return sizeOfInt32(page.values) }

func (page *indexedPage) RepetitionLevels() []int8 { return nil }

func (page *indexedPage) DefinitionLevels() []int8 { return nil }

func (page *indexedPage) WriteTo(e encoding.Encoder) error {
	return e.EncodeInt32(page.values)
}

func (page *indexedPage) Values() ValueReader { return &indexedPageReader{page: page} }

func (page *indexedPage) Buffer() BufferedPage { return page }

type indexedPageReader struct {
	page   *indexedPage
	offset int
}

func (r *indexedPageReader) ReadValues(values []Value) (n int, err error) {
	var v Value
	for n < len(values) && r.offset < len(r.page.values) {
		v = r.page.dict.Index(r.page.values[r.offset])
		v.columnIndex = r.page.columnIndex
		values[n] = v
		r.offset++
		n++
	}
	if r.offset == len(r.page.values) {
		err = io.EOF
	}

	return n, err
}

type indexedColumnBuffer struct {
	indexedPage
	typ Type
}

func newIndexedColumnBuffer(dict Dictionary, typ Type, columnIndex int16, bufferSize int) *indexedColumnBuffer {
	return &indexedColumnBuffer{
		indexedPage: indexedPage{
			dict:        dict,
			values:      make([]int32, 0, bufferSize/4),
			columnIndex: ^columnIndex,
		},
		typ: typ,
	}
}

func (col *indexedColumnBuffer) Clone() ColumnBuffer {
	return &indexedColumnBuffer{
		indexedPage: indexedPage{
			dict:        col.dict,
			values:      append([]int32{}, col.values...),
			columnIndex: col.columnIndex,
		},
		typ: col.typ,
	}
}

func (col *indexedColumnBuffer) Type() Type { return col.typ }

func (col *indexedColumnBuffer) ColumnIndex() ColumnIndex { return indexedColumnIndex{col} }

func (col *indexedColumnBuffer) OffsetIndex() OffsetIndex { return indexedOffsetIndex{col} }

func (col *indexedColumnBuffer) BloomFilter() BloomFilter { return nil }

func (col *indexedColumnBuffer) Dictionary() Dictionary { return col.dict }

func (col *indexedColumnBuffer) Pages() Pages { return onePage(col.Page()) }

func (col *indexedColumnBuffer) Page() BufferedPage { return &col.indexedPage }

func (col *indexedColumnBuffer) Reset() { col.values = col.values[:0] }

func (col *indexedColumnBuffer) Cap() int { return cap(col.values) }

func (col *indexedColumnBuffer) Len() int { return len(col.values) }

func (col *indexedColumnBuffer) Less(i, j int) bool {
	u := col.dict.Index(col.values[i])
	v := col.dict.Index(col.values[j])
	return col.typ.Compare(u, v) < 0
}

func (col *indexedColumnBuffer) Swap(i, j int) {
	col.values[i], col.values[j] = col.values[j], col.values[i]
}

func (col *indexedColumnBuffer) WriteValues(values []Value) (int, error) {
	i := len(col.values)
	j := len(col.values) + len(values)

	if j <= cap(col.values) {
		col.values = col.values[:j]
	} else {
		colValues := make([]int32, j, 2*j)
		copy(colValues, col.values)
		col.values = colValues
	}

	col.dict.Insert(col.values[i:], values)
	return len(values), nil
}

func (col *indexedColumnBuffer) ReadValuesAt(values []Value, offset int64) (n int, err error) {
	i := int(offset)
	switch {
	case i < 0:
		return 0, errRowIndexOutOfBounds(offset, int64(len(col.values)))
	case i >= len(col.values):
		return 0, io.EOF
	default:
		for n < len(values) && i < len(col.values) {
			values[n] = col.dict.Index(col.values[i])
			values[n].columnIndex = col.columnIndex
			n++
			i++
		}
		if n < len(values) {
			err = io.EOF
		}
		return n, err
	}
}

func (col *indexedColumnBuffer) ReadRowAt(row Row, index int64) (Row, error) {
	switch {
	case index < 0:
		return row, errRowIndexOutOfBounds(index, int64(len(col.values)))
	case index >= int64(len(col.values)):
		return row, io.EOF
	default:
		v := col.dict.Index(col.values[index])
		v.columnIndex = col.columnIndex
		return append(row, v), nil
	}
}

type indexedColumnReader struct {
	dict        Dictionary
	typ         Type
	decoder     encoding.Decoder
	buffer      []int32
	offset      int
	remain      int
	columnIndex int16
}

func newIndexedColumnReader(dict Dictionary, typ Type, columnIndex int16, bufferSize int) *indexedColumnReader {
	return &indexedColumnReader{
		dict:        dict,
		typ:         typ,
		buffer:      make([]int32, 0, atLeastOne(bufferSize)),
		columnIndex: ^columnIndex,
	}
}

func (r *indexedColumnReader) Type() Type { return r.typ }

func (r *indexedColumnReader) Column() int { return int(^r.columnIndex) }

func (r *indexedColumnReader) ReadValues(values []Value) (int, error) {
	i := 0
	for {
		for r.offset < len(r.buffer) && i < len(values) {
			count := len(r.buffer) - r.offset
			limit := len(values) - i

			if count > limit {
				count = limit
			}

			indexes := r.buffer[r.offset : r.offset+count]
			dictLen := r.dict.Len()
			for _, index := range indexes {
				if index < 0 || int(index) >= dictLen {
					return i, fmt.Errorf("reading value from indexed page: index out of bounds: %d/%d", index, dictLen)
				}
			}

			r.dict.Lookup(indexes, values[i:])
			r.offset += count
			r.remain -= count

			j := i + int(count)
			for i < j {
				values[i].columnIndex = r.columnIndex
				i++
			}
		}

		if r.remain == 0 {
			return i, io.EOF
		}
		if i == len(values) {
			return i, nil
		}

		length := min(r.remain, cap(r.buffer))
		buffer := r.buffer[:length]
		n, err := r.decoder.DecodeInt32(buffer)
		if n == 0 {
			return i, err
		}

		r.buffer = buffer[:n]
		r.offset = 0
	}
}

func (r *indexedColumnReader) Reset(numValues int, decoder encoding.Decoder) {
	r.decoder = decoder
	r.buffer = r.buffer[:0]
	r.offset = 0
	r.remain = numValues
}

type indexedColumnIndex struct{ col *indexedColumnBuffer }

func (index indexedColumnIndex) NumPages() int       { return 1 }
func (index indexedColumnIndex) NullCount(int) int64 { return 0 }
func (index indexedColumnIndex) NullPage(int) bool   { return false }
func (index indexedColumnIndex) MinValue(int) Value {
	min, _, _ := index.col.Bounds()
	return min
}
func (index indexedColumnIndex) MaxValue(int) Value {
	_, max, _ := index.col.Bounds()
	return max
}
func (index indexedColumnIndex) IsAscending() bool {
	min, max, _ := index.col.Bounds()
	return index.col.typ.Compare(min, max) <= 0
}
func (index indexedColumnIndex) IsDescending() bool {
	min, max, _ := index.col.Bounds()
	return index.col.typ.Compare(min, max) > 0
}

type indexedOffsetIndex struct{ col *indexedColumnBuffer }

func (index indexedOffsetIndex) NumPages() int                { return 1 }
func (index indexedOffsetIndex) Offset(int) int64             { return 0 }
func (index indexedOffsetIndex) CompressedPageSize(int) int64 { return index.col.Size() }
func (index indexedOffsetIndex) FirstRowIndex(int) int64      { return 0 }
