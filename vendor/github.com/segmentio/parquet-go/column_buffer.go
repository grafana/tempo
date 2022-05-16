package parquet

import (
	"bytes"
	"fmt"
	"io"
	"sort"

	"github.com/segmentio/parquet-go/encoding"
	"github.com/segmentio/parquet-go/encoding/plain"
)

// ColumnBuffer is an interface representing columns of a row group.
//
// ColumnBuffer implements sort.Interface as a way to support reordering the
// rows that have been written to it.
type ColumnBuffer interface {
	// Exposes a read-only view of the column buffer.
	ColumnChunk

	// The column implements ValueReaderAt as a mechanism to read values at
	// specific locations within the buffer.
	ValueReaderAt

	// The column implements ValueWriter as a mechanism to optimize the copy
	// of values into the buffer in contexts where the row information is
	// provided by the values because the repetition and definition levels
	// are set.
	ValueWriter

	// For indexed columns, returns the underlying dictionary holding the column
	// values. If the column is not indexed, nil is returned.
	Dictionary() Dictionary

	// Returns a copy of the column. The returned copy shares no memory with
	// the original, mutations of either column will not modify the other.
	Clone() ColumnBuffer

	// Returns the column as a BufferedPage.
	Page() BufferedPage

	// Clears all rows written to the column.
	Reset()

	// Returns the current capacity of the column (rows).
	Cap() int

	// Returns the number of rows currently written to the column.
	Len() int

	// Compares rows at index i and j and reports whether i < j.
	Less(i, j int) bool

	// Swaps rows at index i and j.
	Swap(i, j int)

	// Returns the size of the column buffer in bytes.
	Size() int64
}

func columnIndexOfNullable(base ColumnBuffer, maxDefinitionLevel int8, definitionLevels []int8) ColumnIndex {
	return &nullableColumnIndex{
		ColumnIndex:        base.ColumnIndex(),
		maxDefinitionLevel: maxDefinitionLevel,
		definitionLevels:   definitionLevels,
	}
}

type nullableColumnIndex struct {
	ColumnIndex
	maxDefinitionLevel int8
	definitionLevels   []int8
}

func (index *nullableColumnIndex) NullPage(i int) bool {
	return index.NullCount(i) == int64(len(index.definitionLevels))
}

func (index *nullableColumnIndex) NullCount(i int) int64 {
	return int64(countLevelsNotEqual(index.definitionLevels, index.maxDefinitionLevel))
}

type nullOrdering func(column ColumnBuffer, i, j int, maxDefinitionLevel, definitionLevel1, definitionLevel2 int8) bool

func nullsGoFirst(column ColumnBuffer, i, j int, maxDefinitionLevel, definitionLevel1, definitionLevel2 int8) bool {
	if definitionLevel1 != maxDefinitionLevel {
		return definitionLevel2 == maxDefinitionLevel
	} else {
		return definitionLevel2 == maxDefinitionLevel && column.Less(i, j)
	}
}

func nullsGoLast(column ColumnBuffer, i, j int, maxDefinitionLevel, definitionLevel1, definitionLevel2 int8) bool {
	return definitionLevel1 == maxDefinitionLevel && (definitionLevel2 != maxDefinitionLevel || column.Less(i, j))
}

// reversedColumnBuffer is an adapter of ColumnBuffer which inverses the order
// in which rows are ordered when the column gets sorted.
//
// This type is used when buffers are constructed with sorting columns ordering
// values in descending order.
type reversedColumnBuffer struct{ ColumnBuffer }

func (col *reversedColumnBuffer) Less(i, j int) bool { return col.ColumnBuffer.Less(j, i) }

// optionalColumnBuffer is an implementation of the ColumnBuffer interface used
// as a wrapper to an underlying ColumnBuffer to manage the creation of
// definition levels.
//
// Null values are not written to the underlying column; instead, the buffer
// tracks offsets of row values in the column, null row values are represented
// by the value -1 and a definition level less than the max.
//
// This column buffer type is used for all leaf columns that have a non-zero
// max definition level and a zero repetition level, which may be because the
// column or one of its parent(s) are marked optional.
type optionalColumnBuffer struct {
	base               ColumnBuffer
	maxDefinitionLevel int8
	rows               []int32
	sortIndex          []int32
	definitionLevels   []int8
	nullOrdering       nullOrdering
}

func newOptionalColumnBuffer(base ColumnBuffer, maxDefinitionLevel int8, nullOrdering nullOrdering) *optionalColumnBuffer {
	n := base.Cap()
	return &optionalColumnBuffer{
		base:               base,
		maxDefinitionLevel: maxDefinitionLevel,
		rows:               make([]int32, 0, n),
		definitionLevels:   make([]int8, 0, n),
		nullOrdering:       nullOrdering,
	}
}

func (col *optionalColumnBuffer) Clone() ColumnBuffer {
	return &optionalColumnBuffer{
		base:               col.base.Clone(),
		maxDefinitionLevel: col.maxDefinitionLevel,
		rows:               append([]int32{}, col.rows...),
		definitionLevels:   append([]int8{}, col.definitionLevels...),
		nullOrdering:       col.nullOrdering,
	}
}

func (col *optionalColumnBuffer) Type() Type {
	return col.base.Type()
}

func (col *optionalColumnBuffer) NumValues() int64 {
	return int64(len(col.definitionLevels))
}

func (col *optionalColumnBuffer) ColumnIndex() ColumnIndex {
	return columnIndexOfNullable(col.base, col.maxDefinitionLevel, col.definitionLevels)
}

func (col *optionalColumnBuffer) OffsetIndex() OffsetIndex {
	return col.base.OffsetIndex()
}

func (col *optionalColumnBuffer) BloomFilter() BloomFilter {
	return col.base.BloomFilter()
}

func (col *optionalColumnBuffer) Dictionary() Dictionary {
	return col.base.Dictionary()
}

func (col *optionalColumnBuffer) Column() int {
	return col.base.Column()
}

func (col *optionalColumnBuffer) Pages() Pages {
	return onePage(col.Page())
}

func (col *optionalColumnBuffer) Page() BufferedPage {
	if !optionalRowsHaveBeenReordered(col.rows) {
		// No need for any cyclic sorting if the rows have not been reordered.
		// This case is also important because the cyclic sorting modifies the
		// buffer which makes it unsafe to read the buffer concurrently.
		return newOptionalPage(col.base.Page(), col.maxDefinitionLevel, col.definitionLevels)
	}

	numNulls := countLevelsNotEqual(col.definitionLevels, col.maxDefinitionLevel)
	numValues := len(col.rows) - numNulls

	if numValues > 0 {
		if cap(col.sortIndex) < numValues {
			col.sortIndex = make([]int32, numValues)
		}
		sortIndex := col.sortIndex[:numValues]
		i := 0
		for _, j := range col.rows {
			if j >= 0 {
				sortIndex[j] = int32(i)
				i++
			}
		}

		// Cyclic sort: O(N)
		for i := range sortIndex {
			for j := int(sortIndex[i]); i != j; j = int(sortIndex[i]) {
				col.base.Swap(i, j)
				sortIndex[i], sortIndex[j] = sortIndex[j], sortIndex[i]
			}
		}
	}

	i := 0
	for _, r := range col.rows {
		if r >= 0 {
			col.rows[i] = int32(i)
			i++
		}
	}

	return newOptionalPage(col.base.Page(), col.maxDefinitionLevel, col.definitionLevels)
}

func (col *optionalColumnBuffer) Reset() {
	col.base.Reset()
	col.rows = col.rows[:0]
	col.definitionLevels = col.definitionLevels[:0]
}

func (col *optionalColumnBuffer) Size() int64 {
	return sizeOfInt32(col.rows) + sizeOfInt32(col.sortIndex) + sizeOfInt8(col.definitionLevels) + col.base.Size()
}

func (col *optionalColumnBuffer) Cap() int { return cap(col.rows) }

func (col *optionalColumnBuffer) Len() int { return len(col.rows) }

func (col *optionalColumnBuffer) Less(i, j int) bool {
	return col.nullOrdering(
		col.base,
		int(col.rows[i]),
		int(col.rows[j]),
		col.maxDefinitionLevel,
		col.definitionLevels[i],
		col.definitionLevels[j],
	)
}

func (col *optionalColumnBuffer) Swap(i, j int) {
	// Because the underlying column does not contain null values, we cannot
	// swap its values at indexes i and j. We swap the row indexes only, then
	// reorder the underlying buffer using a cyclic sort when the buffer is
	// materialized into a page view.
	col.rows[i], col.rows[j] = col.rows[j], col.rows[i]
	col.definitionLevels[i], col.definitionLevels[j] = col.definitionLevels[j], col.definitionLevels[i]
}

func (col *optionalColumnBuffer) WriteValues(values []Value) (n int, err error) {
	rowIndex := int32(col.base.Len())

	for n < len(values) {
		// Collect index range of contiguous null values, from i to n. If this
		// for loop exhausts the values, all remaining if statements and for
		// loops will be no-ops and the loop will terminate.
		i := n
		for n < len(values) && values[n].definitionLevel != col.maxDefinitionLevel {
			n++
		}

		// Write the contiguous null values up until the first non-null value
		// obtained in the for loop above.
		for _, v := range values[i:n] {
			col.rows = append(col.rows, -1)
			col.definitionLevels = append(col.definitionLevels, v.definitionLevel)
		}

		// Collect index range of contiguous non-null values, from i to n.
		i = n
		for n < len(values) && values[n].definitionLevel == col.maxDefinitionLevel {
			n++
		}

		// As long as i < n we have non-null values still to write. It is
		// possible that we just exhausted the input values in which case i == n
		// and the outer for loop will terminate.
		if i < n {
			count, err := col.base.WriteValues(values[i:n])
			col.definitionLevels = appendLevel(col.definitionLevels, col.maxDefinitionLevel, count)

			for count > 0 {
				col.rows = append(col.rows, rowIndex)
				rowIndex++
				count--
			}

			if err != nil {
				return n, err
			}
		}
	}

	return n, nil
}

func (col *optionalColumnBuffer) ReadValuesAt(values []Value, offset int64) (int, error) {
	length := int64(len(col.definitionLevels))
	if offset < 0 {
		return 0, errRowIndexOutOfBounds(offset, length)
	}
	if offset >= length {
		return 0, io.EOF
	}
	if length -= offset; length < int64(len(values)) {
		values = values[:length]
	}

	numNulls1 := int64(countLevelsNotEqual(col.definitionLevels[:offset], col.maxDefinitionLevel))
	numNulls2 := int64(countLevelsNotEqual(col.definitionLevels[offset:offset+length], col.maxDefinitionLevel))

	if numNulls2 < length {
		n, err := col.base.ReadValuesAt(values[:length-numNulls2], offset-numNulls1)
		if err != nil {
			return n, err
		}
	}

	if numNulls2 > 0 {
		columnIndex := ^int16(col.Column())
		i := numNulls2 - 1
		j := length - 1
		definitionLevels := col.definitionLevels[offset : offset+length]
		maxDefinitionLevel := col.maxDefinitionLevel

		for n := len(definitionLevels) - 1; n >= 0 && j > i; n-- {
			if definitionLevels[n] != maxDefinitionLevel {
				values[j] = Value{definitionLevel: definitionLevels[n], columnIndex: columnIndex}
			} else {
				values[j] = values[i]
				i--
			}
			j--
		}
	}

	return int(length), nil
}

func (col *optionalColumnBuffer) Values() ValueReader {
	return &optionalPageReader{page: col.Page().(*optionalPage)}
}

// repeatedColumnBuffer is an implementation of the ColumnBuffer interface used
// as a wrapper to an underlying ColumnBuffer to manage the creation of
// repetition levels, definition levels, and map rows to the region of the
// underlying buffer that contains their sequence of values.
//
// Null values are not written to the underlying column; instead, the buffer
// tracks offsets of row values in the column, null row values are represented
// by the value -1 and a definition level less than the max.
//
// This column buffer type is used for all leaf columns that have a non-zero
// max repetition level, which may be because the column or one of its parent(s)
// are marked repeated.
type repeatedColumnBuffer struct {
	base               ColumnBuffer
	maxRepetitionLevel int8
	maxDefinitionLevel int8
	rows               []region
	repetitionLevels   []int8
	definitionLevels   []int8
	buffer             []Value
	reordering         *repeatedColumnBuffer
	nullOrdering       nullOrdering
}

// The region type maps the logical offset of rows within the repetition and
// definition levels, to the base offsets in the underlying column buffers
// where the non-null values have been written.
type region struct {
	offset     uint32
	baseOffset uint32
}

func sizeOfRegion(regions []region) int64 { return 8 * int64(len(regions)) }

func newRepeatedColumnBuffer(base ColumnBuffer, maxRepetitionLevel, maxDefinitionLevel int8, nullOrdering nullOrdering) *repeatedColumnBuffer {
	n := base.Cap()
	return &repeatedColumnBuffer{
		base:               base,
		maxRepetitionLevel: maxRepetitionLevel,
		maxDefinitionLevel: maxDefinitionLevel,
		rows:               make([]region, 0, n/8),
		repetitionLevels:   make([]int8, 0, n),
		definitionLevels:   make([]int8, 0, n),
		nullOrdering:       nullOrdering,
	}
}

func (col *repeatedColumnBuffer) Clone() ColumnBuffer {
	return &repeatedColumnBuffer{
		base:               col.base.Clone(),
		maxRepetitionLevel: col.maxRepetitionLevel,
		maxDefinitionLevel: col.maxDefinitionLevel,
		rows:               append([]region{}, col.rows...),
		repetitionLevels:   append([]int8{}, col.repetitionLevels...),
		definitionLevels:   append([]int8{}, col.definitionLevels...),
		nullOrdering:       col.nullOrdering,
	}
}

func (col *repeatedColumnBuffer) Type() Type {
	return col.base.Type()
}

func (col *repeatedColumnBuffer) NumValues() int64 {
	return int64(len(col.definitionLevels))
}

func (col *repeatedColumnBuffer) ColumnIndex() ColumnIndex {
	return columnIndexOfNullable(col.base, col.maxDefinitionLevel, col.definitionLevels)
}

func (col *repeatedColumnBuffer) OffsetIndex() OffsetIndex {
	return col.base.OffsetIndex()
}

func (col *repeatedColumnBuffer) BloomFilter() BloomFilter {
	return col.base.BloomFilter()
}

func (col *repeatedColumnBuffer) Dictionary() Dictionary {
	return col.base.Dictionary()
}

func (col *repeatedColumnBuffer) Column() int {
	return col.base.Column()
}

func (col *repeatedColumnBuffer) Pages() Pages {
	return onePage(col.Page())
}

func (col *repeatedColumnBuffer) Page() BufferedPage {
	if repeatedRowsHaveBeenReordered(col.rows) {
		if col.reordering == nil {
			col.reordering = col.Clone().(*repeatedColumnBuffer)
		}

		column := col.reordering
		column.Reset()
		maxNumValues := 0
		defer func() {
			clearValues(col.buffer[:maxNumValues])
		}()

		baseOffset := 0

		for _, row := range col.rows {
			rowOffset := int(row.offset)
			rowLength := repeatedRowLength(col.repetitionLevels[rowOffset:])
			numNulls := countLevelsNotEqual(col.definitionLevels[rowOffset:rowOffset+rowLength], col.maxDefinitionLevel)
			numValues := rowLength - numNulls

			if numValues > 0 {
				if numValues > cap(col.buffer) {
					col.buffer = make([]Value, numValues)
				} else {
					col.buffer = col.buffer[:numValues]
				}
				n, err := col.base.ReadValuesAt(col.buffer, int64(row.baseOffset))
				if err != nil && n < numValues {
					return newErrorPage(col.Column(), "reordering rows of repeated column: %w", err)
				}
				if _, err := column.base.WriteValues(col.buffer); err != nil {
					return newErrorPage(col.Column(), "reordering rows of repeated column: %w", err)
				}
				if numValues > maxNumValues {
					maxNumValues = numValues
				}
			}

			column.rows = append(column.rows, region{
				offset:     uint32(len(column.repetitionLevels)),
				baseOffset: uint32(baseOffset),
			})

			column.repetitionLevels = append(column.repetitionLevels, col.repetitionLevels[rowOffset:rowOffset+rowLength]...)
			column.definitionLevels = append(column.definitionLevels, col.definitionLevels[rowOffset:rowOffset+rowLength]...)
			baseOffset += numValues
		}

		col.swapReorderingBuffer(column)
	}

	return newRepeatedPage(
		col.base.Page(),
		col.maxRepetitionLevel,
		col.maxDefinitionLevel,
		col.repetitionLevels,
		col.definitionLevels,
	)
}

func (col *repeatedColumnBuffer) swapReorderingBuffer(buf *repeatedColumnBuffer) {
	col.base, buf.base = buf.base, col.base
	col.rows, buf.rows = buf.rows, col.rows
	col.repetitionLevels, buf.repetitionLevels = buf.repetitionLevels, col.repetitionLevels
	col.definitionLevels, buf.definitionLevels = buf.definitionLevels, col.definitionLevels
}

func (col *repeatedColumnBuffer) Reset() {
	col.base.Reset()
	col.rows = col.rows[:0]
	col.repetitionLevels = col.repetitionLevels[:0]
	col.definitionLevels = col.definitionLevels[:0]
}

func (col *repeatedColumnBuffer) Size() int64 {
	return sizeOfRegion(col.rows) + sizeOfInt8(col.repetitionLevels) + sizeOfInt8(col.definitionLevels) + col.base.Size()
}

func (col *repeatedColumnBuffer) Cap() int { return cap(col.rows) }

func (col *repeatedColumnBuffer) Len() int { return len(col.rows) }

func (col *repeatedColumnBuffer) Less(i, j int) bool {
	row1 := col.rows[i]
	row2 := col.rows[j]
	less := col.nullOrdering
	row1Length := repeatedRowLength(col.repetitionLevels[row1.offset:])
	row2Length := repeatedRowLength(col.repetitionLevels[row2.offset:])

	for k := 0; k < row1Length && k < row2Length; k++ {
		x := int(row1.offset) + k
		y := int(row2.offset) + k
		definitionLevel1 := col.definitionLevels[j+k]
		definitionLevel2 := col.definitionLevels[j+k]
		switch {
		case less(col.base, x, y, col.maxDefinitionLevel, definitionLevel1, definitionLevel2):
			return true
		case less(col.base, y, x, col.maxDefinitionLevel, definitionLevel2, definitionLevel1):
			return false
		}
	}

	return row1Length < row2Length
}

func (col *repeatedColumnBuffer) Swap(i, j int) {
	// Because the underlying column does not contain null values, and may hold
	// an arbitrary number of values per row, we cannot swap its values at
	// indexes i and j. We swap the row indexes only, then reorder the base
	// column buffer when its view is materialized into a page by creating a
	// copy and writing rows back to it following the order of rows in the
	// repeated column buffer.
	col.rows[i], col.rows[j] = col.rows[j], col.rows[i]
}

func (col *repeatedColumnBuffer) WriteValues(values []Value) (numValues int, err error) {
	// The values may belong to the last row that was written if they do not
	// start with a repetition level less than the column's maximum.
	var continuation Row
	if len(values) > 0 && values[0].repetitionLevel != 0 {
		continuation, values = splitRowValues(values)
	}

	if len(continuation) > 0 {
		for i, v := range continuation {
			if v.definitionLevel == col.maxDefinitionLevel {
				if _, err := col.base.WriteValues(continuation[i : i+1]); err != nil {
					return numValues, err
				}
			}
			col.repetitionLevels = append(col.repetitionLevels, v.repetitionLevel)
			col.definitionLevels = append(col.definitionLevels, v.definitionLevel)
			numValues++
		}
	}

	maxNumValues := 0
	defer func() {
		clearValues(col.buffer[:maxNumValues])
	}()

	var row []Value

	for len(values) > 0 {
		row, values = splitRowValues(values)
		if err := col.writeRow(row); err != nil {
			return numValues, err
		}
		numValues += len(row)
		if len(col.buffer) > maxNumValues {
			maxNumValues = len(col.buffer)
		}
	}

	return numValues, nil
}

func (col *repeatedColumnBuffer) writeRow(row []Value) error {
	col.buffer = col.buffer[:0]
	for _, v := range row {
		if v.definitionLevel == col.maxDefinitionLevel {
			col.buffer = append(col.buffer, v)
		}
	}

	baseOffset := col.base.NumValues()
	if len(col.buffer) > 0 {
		if _, err := col.base.WriteValues(col.buffer); err != nil {
			return err
		}
	}

	col.rows = append(col.rows, region{
		offset:     uint32(len(col.repetitionLevels)),
		baseOffset: uint32(baseOffset),
	})

	for _, v := range row {
		col.repetitionLevels = append(col.repetitionLevels, v.repetitionLevel)
		col.definitionLevels = append(col.definitionLevels, v.definitionLevel)
	}

	return nil
}

func (col *repeatedColumnBuffer) ReadValuesAt(values []Value, offset int64) (int, error) {
	// TODO:
	panic("NOT IMPLEMENTED")
}

func (col *repeatedColumnBuffer) Values() ValueReader {
	return &repeatedPageReader{page: col.Page().(*repeatedPage)}
}

func optionalRowsHaveBeenReordered(rows []int32) bool {
	i := int32(0)
	for _, row := range rows {
		if row < 0 {
			// Skip any row that is null.
			continue
		}

		// If rows have been reordered the indices are not increasing exactly
		// one by one.
		if row != i {
			return true
		}

		// Only increment the index if the row is not null.
		i++
	}
	return false
}

func repeatedRowsHaveBeenReordered(rows []region) bool {
	lastOffset := uint32(0)
	for _, row := range rows {
		if row.offset < lastOffset {
			return true
		}
		lastOffset = row.offset
	}
	return false
}

type byteArrayColumnBuffer struct {
	byteArrayPage
	typ Type
}

func newByteArrayColumnBuffer(typ Type, columnIndex int16, bufferSize int) *byteArrayColumnBuffer {
	return &byteArrayColumnBuffer{
		byteArrayPage: byteArrayPage{
			values:      encoding.MakeByteArrayList(bufferSize / 16),
			columnIndex: ^columnIndex,
		},
		typ: typ,
	}
}

func (col *byteArrayColumnBuffer) Clone() ColumnBuffer {
	return &byteArrayColumnBuffer{
		byteArrayPage: byteArrayPage{
			values:      col.values.Clone(),
			columnIndex: col.columnIndex,
		},
		typ: col.typ,
	}
}

func (col *byteArrayColumnBuffer) Type() Type { return col.typ }

func (col *byteArrayColumnBuffer) ColumnIndex() ColumnIndex {
	return byteArrayColumnIndex{&col.byteArrayPage}
}

func (col *byteArrayColumnBuffer) OffsetIndex() OffsetIndex {
	return byteArrayOffsetIndex{&col.byteArrayPage}
}

func (col *byteArrayColumnBuffer) BloomFilter() BloomFilter { return nil }

func (col *byteArrayColumnBuffer) Dictionary() Dictionary { return nil }

func (col *byteArrayColumnBuffer) Pages() Pages { return onePage(col.Page()) }

func (col *byteArrayColumnBuffer) Page() BufferedPage { return &col.byteArrayPage }

func (col *byteArrayColumnBuffer) Reset() { col.values.Reset() }

func (col *byteArrayColumnBuffer) Cap() int { return col.values.Cap() }

func (col *byteArrayColumnBuffer) Len() int { return col.values.Len() }

func (col *byteArrayColumnBuffer) Less(i, j int) bool { return col.values.Less(i, j) }

func (col *byteArrayColumnBuffer) Swap(i, j int) { col.values.Swap(i, j) }

func (col *byteArrayColumnBuffer) Write(b []byte) (int, error) {
	_, n, err := col.writeByteArrays(b)
	return n, err
}

func (col *byteArrayColumnBuffer) WriteRequired(values []byte) (int, error) {
	return col.WriteByteArrays(values)
}

func (col *byteArrayColumnBuffer) WriteByteArrays(values []byte) (int, error) {
	n, _, err := col.writeByteArrays(values)
	return n, err
}

func (col *byteArrayColumnBuffer) writeByteArrays(values []byte) (c, n int, err error) {
	err = plain.RangeByteArrays(values, func(v []byte) error {
		col.values.Push(v)
		n += plain.ByteArrayLengthSize + len(v)
		c++
		return nil
	})
	return c, n, err
}

func (col *byteArrayColumnBuffer) WriteValues(values []Value) (int, error) {
	for _, v := range values {
		col.values.Push(v.ByteArray())
	}
	return len(values), nil
}

func (col *byteArrayColumnBuffer) ReadValuesAt(values []Value, offset int64) (n int, err error) {
	i := int(offset)
	switch {
	case i < 0:
		return 0, errRowIndexOutOfBounds(offset, int64(col.values.Len()))
	case i >= col.values.Len():
		return 0, io.EOF
	default:
		for n < len(values) && i < col.values.Len() {
			values[n] = makeValueBytes(ByteArray, col.values.Index(i))
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

type fixedLenByteArrayColumnBuffer struct {
	fixedLenByteArrayPage
	typ Type
	tmp []byte
}

func newFixedLenByteArrayColumnBuffer(typ Type, columnIndex int16, bufferSize int) *fixedLenByteArrayColumnBuffer {
	size := typ.Length()
	return &fixedLenByteArrayColumnBuffer{
		fixedLenByteArrayPage: fixedLenByteArrayPage{
			size:        size,
			data:        make([]byte, 0, bufferSize),
			columnIndex: ^columnIndex,
		},
		typ: typ,
		tmp: make([]byte, size),
	}
}

func (col *fixedLenByteArrayColumnBuffer) Clone() ColumnBuffer {
	return &fixedLenByteArrayColumnBuffer{
		fixedLenByteArrayPage: fixedLenByteArrayPage{
			size:        col.size,
			data:        append([]byte{}, col.data...),
			columnIndex: col.columnIndex,
		},
		typ: col.typ,
		tmp: make([]byte, col.size),
	}
}

func (col *fixedLenByteArrayColumnBuffer) Type() Type { return col.typ }

func (col *fixedLenByteArrayColumnBuffer) ColumnIndex() ColumnIndex {
	return fixedLenByteArrayColumnIndex{&col.fixedLenByteArrayPage}
}

func (col *fixedLenByteArrayColumnBuffer) OffsetIndex() OffsetIndex {
	return fixedLenByteArrayOffsetIndex{&col.fixedLenByteArrayPage}
}

func (col *fixedLenByteArrayColumnBuffer) BloomFilter() BloomFilter { return nil }

func (col *fixedLenByteArrayColumnBuffer) Dictionary() Dictionary { return nil }

func (col *fixedLenByteArrayColumnBuffer) Pages() Pages { return onePage(col.Page()) }

func (col *fixedLenByteArrayColumnBuffer) Page() BufferedPage { return &col.fixedLenByteArrayPage }

func (col *fixedLenByteArrayColumnBuffer) Reset() { col.data = col.data[:0] }

func (col *fixedLenByteArrayColumnBuffer) Cap() int { return cap(col.data) / col.size }

func (col *fixedLenByteArrayColumnBuffer) Len() int { return len(col.data) / col.size }

func (col *fixedLenByteArrayColumnBuffer) Less(i, j int) bool {
	return bytes.Compare(col.index(i), col.index(j)) < 0
}

func (col *fixedLenByteArrayColumnBuffer) Swap(i, j int) {
	t, u, v := col.tmp[:col.size], col.index(i), col.index(j)
	copy(t, u)
	copy(u, v)
	copy(v, t)
}

func (col *fixedLenByteArrayColumnBuffer) index(i int) []byte {
	j := (i + 0) * col.size
	k := (i + 1) * col.size
	return col.data[j:k:k]
}

func (col *fixedLenByteArrayColumnBuffer) Write(b []byte) (int, error) {
	n, err := col.WriteFixedLenByteArrays(b)
	return n * col.size, err
}

func (col *fixedLenByteArrayColumnBuffer) WriteRequired(values []byte) (int, error) {
	return col.WriteFixedLenByteArrays(values)
}

func (col *fixedLenByteArrayColumnBuffer) WriteFixedLenByteArrays(values []byte) (int, error) {
	d, m := len(values)/col.size, len(values)%col.size
	if m != 0 {
		return 0, fmt.Errorf("cannot write FIXED_LEN_BYTE_ARRAY values of size %d from input of size %d", col.size, len(values))
	}
	col.data = append(col.data, values...)
	return d, nil
}

func (col *fixedLenByteArrayColumnBuffer) WriteValues(values []Value) (int, error) {
	for _, v := range values {
		col.data = append(col.data, v.ByteArray()...)
	}
	return len(values), nil
}

func (col *fixedLenByteArrayColumnBuffer) ReadValuesAt(values []Value, offset int64) (n int, err error) {
	i := int(offset) * col.size
	switch {
	case i < 0:
		return 0, errRowIndexOutOfBounds(offset, int64(len(col.data)/col.size))
	case i >= len(col.data):
		return 0, io.EOF
	default:
		for n < len(values) && i < len(col.data) {
			values[n] = makeValueBytes(FixedLenByteArray, col.data[i:i+col.size])
			values[n].columnIndex = col.columnIndex
			n++
			i += col.size
		}
		if n < len(values) {
			err = io.EOF
		}
		return n, err
	}
}

var (
	_ sort.Interface = (ColumnBuffer)(nil)
	_ io.Writer      = (*byteArrayColumnBuffer)(nil)
	_ io.Writer      = (*fixedLenByteArrayColumnBuffer)(nil)
)
