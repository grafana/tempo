package parquet

import (
	"fmt"
	"io"

	"github.com/segmentio/parquet-go/deprecated"
	"github.com/segmentio/parquet-go/encoding"
	"github.com/segmentio/parquet-go/encoding/plain"
	"github.com/segmentio/parquet-go/internal/bits"
)

// Page values represent sequences of parquet values. From the Parquet
// documentation: "Column chunks are a chunk of the data for a particular
// column. They live in a particular row group and are guaranteed to be
// contiguous in the file. Column chunks are divided up into pages. A page is
// conceptually an indivisible unit (in terms of compression and encoding).
// There can be multiple page types which are interleaved in a column chunk."
//
// https://github.com/apache/parquet-format#glossary
type Page interface {
	// Returns the column index that this page belongs to.
	Column() int

	// If the page contains indexed values, calling this method returns the
	// dictionary in which the values are looked up. Otherwise, the method
	// returns nil.
	Dictionary() Dictionary

	// Returns the number of rows, values, and nulls in the page. The number of
	// rows may be less than the number of values in the page if the page is
	// part of a repeated column.
	NumRows() int64
	NumValues() int64
	NumNulls() int64

	// Returns the min and max values currently buffered in the writer.
	//
	// The third value is a boolean indicating whether the page bounds were
	// available. Page bounds may not be known if the page contained no values
	// or only nulls, or if they were read from a parquet file which had neither
	// page statistics nor a page index.
	Bounds() (min, max Value, ok bool)

	// Returns the size of the page in bytes (uncompressed).
	Size() int64

	// Returns a reader exposing the values contained in the page.
	//
	// Depending on the underlying implementation, the returned reader may
	// support reading an array of typed Go values by implementing interfaces
	// like parquet.Int32Reader. Applications should use type assertions on
	// the returned reader to determine whether those optimizations are
	// available.
	Values() ValueReader

	// Buffer returns the page as a BufferedPage, which may be the page itself
	// if it was already buffered.
	Buffer() BufferedPage
}

// BufferedPage is an extension of the Page interface implemented by pages
// that are buffered in memory.
type BufferedPage interface {
	Page

	// Returns a copy of the page which does not share any of the buffers, but
	// contains the same values, repetition and definition levels.
	Clone() BufferedPage

	// Returns a new page which is as slice of the receiver between row indexes
	// i and j.
	Slice(i, j int64) BufferedPage

	// Expose the lists of repetition and definition levels of the page.
	//
	// The returned slices may be empty when the page has no repetition or
	// definition levels.
	RepetitionLevels() []int8
	DefinitionLevels() []int8

	// Writes the page to the given encoder.
	WriteTo(encoding.Encoder) error
}

// CompressedPage is an extension of the Page interface implemented by pages
// that have been compressed to their on-file representation.
type CompressedPage interface {
	Page

	// Returns a representation of the page header.
	PageHeader() PageHeader

	// Returns a reader exposing the content of the compressed page.
	PageData() io.Reader

	// Returns the size of the page data.
	PageSize() int64

	// CRC returns the IEEE CRC32 checksum of the page.
	CRC() uint32
}

// PageReader is an interface implemented by types that support producing a
// sequence of pages.
type PageReader interface {
	ReadPage() (Page, error)
}

// PageWriter is an interface implemented by types that support writing pages
// to an underlying storage medium.
type PageWriter interface {
	WritePage(Page) (int64, error)
}

type singlePage struct {
	page Page
	seek int64
}

func (r *singlePage) ReadPage() (Page, error) {
	if numRows := r.page.NumRows(); r.seek < numRows {
		seek := r.seek
		r.seek = numRows
		if seek > 0 {
			return r.page.Buffer().Slice(seek, numRows), nil
		}
		return r.page, nil
	}
	return nil, io.EOF
}

func (r *singlePage) SeekToRow(rowIndex int64) error {
	r.seek = rowIndex
	return nil
}

func onePage(page Page) Pages { return &singlePage{page: page} }

// CopyPages copies pages from src to dst, returning the number of values that
// were copied.
//
// The function returns any error it encounters reading or writing pages, except
// for io.EOF from the reader which indicates that there were no more pages to
// read.
func CopyPages(dst PageWriter, src PageReader) (numValues int64, err error) {
	for {
		p, err := src.ReadPage()
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return numValues, err
		}
		n, err := dst.WritePage(p)
		numValues += n
		if err != nil {
			return numValues, err
		}
	}
}

func sizeOfBytes(data []byte) int64 { return 1 * int64(len(data)) }

func sizeOfBool(data []bool) int64 { return 1 * int64(len(data)) }

func sizeOfInt8(data []int8) int64 { return 1 * int64(len(data)) }

func sizeOfInt32(data []int32) int64 { return 4 * int64(len(data)) }

func sizeOfInt64(data []int64) int64 { return 8 * int64(len(data)) }

func sizeOfInt96(data []deprecated.Int96) int64 { return 12 * int64(len(data)) }

func sizeOfFloat32(data []float32) int64 { return 4 * int64(len(data)) }

func sizeOfFloat64(data []float64) int64 { return 8 * int64(len(data)) }

func forEachPageSlice(page BufferedPage, wantSize int64, do func(BufferedPage) error) error {
	numRows := page.NumRows()
	if numRows == 0 {
		return nil
	}

	pageSize := page.Size()
	numPages := (pageSize + (wantSize - 1)) / wantSize
	rowIndex := int64(0)
	if numPages < 2 {
		return do(page)
	}

	for numPages > 0 {
		lastRowIndex := rowIndex + ((numRows - rowIndex) / numPages)
		if err := do(page.Slice(rowIndex, lastRowIndex)); err != nil {
			return err
		}
		rowIndex = lastRowIndex
		numPages--
	}

	return nil
}

// errorPage is an implementation of the Page interface which always errors when
// attempting to read its values.
//
// The error page declares that it contains one value (even if it does not)
// as a way to ensure that it is not ignored due to being empty when written
// to a file.
type errorPage struct {
	err         error
	columnIndex int
}

func newErrorPage(columnIndex int, msg string, args ...interface{}) *errorPage {
	return &errorPage{
		err:         fmt.Errorf(msg, args...),
		columnIndex: columnIndex,
	}
}

func (page *errorPage) Column() int                       { return page.columnIndex }
func (page *errorPage) Dictionary() Dictionary            { return nil }
func (page *errorPage) NumRows() int64                    { return 1 }
func (page *errorPage) NumValues() int64                  { return 1 }
func (page *errorPage) NumNulls() int64                   { return 0 }
func (page *errorPage) Bounds() (min, max Value, ok bool) { return }
func (page *errorPage) Clone() BufferedPage               { return page }
func (page *errorPage) Slice(i, j int64) BufferedPage     { return page }
func (page *errorPage) Size() int64                       { return 1 }
func (page *errorPage) RepetitionLevels() []int8          { return nil }
func (page *errorPage) DefinitionLevels() []int8          { return nil }
func (page *errorPage) WriteTo(encoding.Encoder) error    { return page.err }
func (page *errorPage) Values() ValueReader               { return &errorValueReader{err: page.err} }
func (page *errorPage) Buffer() BufferedPage              { return page }

func errPageBoundsOutOfRange(i, j, n int64) error {
	return fmt.Errorf("page bounds out of range [%d:%d]: with length %d", i, j, n)
}

func countLevelsEqual(levels []int8, value int8) int {
	return bits.CountByte(bits.Int8ToBytes(levels), byte(value))
}

func countLevelsNotEqual(levels []int8, value int8) int {
	return len(levels) - countLevelsEqual(levels, value)
}

func appendLevel(levels []int8, value int8, count int) []int8 {
	if count > 0 {
		i := len(levels)
		j := len(levels) + 1

		if n := len(levels) + count; cap(levels) < n {
			newLevels := make([]int8, n)
			copy(newLevels, levels)
			levels = newLevels
		} else {
			levels = levels[:n]
		}

		for levels[i] = value; j < len(levels); j *= 2 {
			copy(levels[j:], levels[i:j])
		}
	}
	return levels
}

type optionalPage struct {
	base               BufferedPage
	maxDefinitionLevel int8
	definitionLevels   []int8
}

func newOptionalPage(base BufferedPage, maxDefinitionLevel int8, definitionLevels []int8) *optionalPage {
	return &optionalPage{
		base:               base,
		maxDefinitionLevel: maxDefinitionLevel,
		definitionLevels:   definitionLevels,
	}
}

func (page *optionalPage) Column() int {
	return page.base.Column()
}

func (page *optionalPage) Dictionary() Dictionary {
	return page.base.Dictionary()
}

func (page *optionalPage) NumRows() int64 {
	return int64(len(page.definitionLevels))
}

func (page *optionalPage) NumValues() int64 {
	return int64(len(page.definitionLevels))
}

func (page *optionalPage) NumNulls() int64 {
	return int64(countLevelsNotEqual(page.definitionLevels, page.maxDefinitionLevel))
}

func (page *optionalPage) Bounds() (min, max Value, ok bool) {
	return page.base.Bounds()
}

func (page *optionalPage) Clone() BufferedPage {
	return newOptionalPage(
		page.base.Clone(),
		page.maxDefinitionLevel,
		append([]int8{}, page.definitionLevels...),
	)
}

func (page *optionalPage) Slice(i, j int64) BufferedPage {
	numNulls1 := int64(countLevelsNotEqual(page.definitionLevels[:i], page.maxDefinitionLevel))
	numNulls2 := int64(countLevelsNotEqual(page.definitionLevels[i:j], page.maxDefinitionLevel))
	return newOptionalPage(
		page.base.Slice(i-numNulls1, j-(numNulls1+numNulls2)),
		page.maxDefinitionLevel,
		page.definitionLevels[i:j],
	)
}

func (page *optionalPage) Size() int64 {
	return page.base.Size() + sizeOfInt8(page.definitionLevels)
}

func (page *optionalPage) RepetitionLevels() []int8 {
	return nil
}

func (page *optionalPage) DefinitionLevels() []int8 {
	return page.definitionLevels
}

func (page *optionalPage) WriteTo(e encoding.Encoder) error {
	return page.base.WriteTo(e)
}

func (page *optionalPage) Values() ValueReader {
	return &optionalPageReader{page: page}
}

func (page *optionalPage) Buffer() BufferedPage {
	return page
}

type optionalPageReader struct {
	page   *optionalPage
	values ValueReader
	offset int
}

func (r *optionalPageReader) ReadValues(values []Value) (n int, err error) {
	if r.values == nil {
		r.values = r.page.base.Values()
	}
	maxDefinitionLevel := r.page.maxDefinitionLevel
	columnIndex := ^int16(r.page.Column())

	for n < len(values) && r.offset < len(r.page.definitionLevels) {
		for n < len(values) && r.offset < len(r.page.definitionLevels) && r.page.definitionLevels[r.offset] != maxDefinitionLevel {
			values[n] = Value{
				definitionLevel: r.page.definitionLevels[r.offset],
				columnIndex:     columnIndex,
			}
			r.offset++
			n++
		}

		i := n
		j := r.offset
		for i < len(values) && j < len(r.page.definitionLevels) && r.page.definitionLevels[j] == maxDefinitionLevel {
			i++
			j++
		}

		if n < i {
			for j, err = r.values.ReadValues(values[n:i]); j > 0; j-- {
				values[n].definitionLevel = maxDefinitionLevel
				r.offset++
				n++
			}
			// Do not return on an io.EOF here as we may still have null values to read.
			if err != nil && err != io.EOF {
				return n, err
			}
		}
	}

	if r.offset == len(r.page.definitionLevels) {
		err = io.EOF
	}
	return n, err
}

type repeatedPage struct {
	base               BufferedPage
	maxRepetitionLevel int8
	maxDefinitionLevel int8
	definitionLevels   []int8
	repetitionLevels   []int8
}

func newRepeatedPage(base BufferedPage, maxRepetitionLevel, maxDefinitionLevel int8, repetitionLevels, definitionLevels []int8) *repeatedPage {
	return &repeatedPage{
		base:               base,
		maxRepetitionLevel: maxRepetitionLevel,
		maxDefinitionLevel: maxDefinitionLevel,
		definitionLevels:   definitionLevels,
		repetitionLevels:   repetitionLevels,
	}
}

func (page *repeatedPage) Column() int {
	return page.base.Column()
}

func (page *repeatedPage) Dictionary() Dictionary {
	return page.base.Dictionary()
}

func (page *repeatedPage) NumRows() int64 {
	return int64(countLevelsEqual(page.repetitionLevels, 0))
}

func (page *repeatedPage) NumValues() int64 {
	return int64(len(page.definitionLevels))
}

func (page *repeatedPage) NumNulls() int64 {
	return int64(countLevelsNotEqual(page.definitionLevels, page.maxDefinitionLevel))
}

func (page *repeatedPage) Bounds() (min, max Value, ok bool) {
	return page.base.Bounds()
}

func (page *repeatedPage) Clone() BufferedPage {
	return newRepeatedPage(
		page.base.Clone(),
		page.maxRepetitionLevel,
		page.maxDefinitionLevel,
		append([]int8{}, page.repetitionLevels...),
		append([]int8{}, page.definitionLevels...),
	)
}

func (page *repeatedPage) Slice(i, j int64) BufferedPage {
	numRows := page.NumRows()
	if i < 0 || i > numRows {
		panic(errPageBoundsOutOfRange(i, j, numRows))
	}
	if j < 0 || j > numRows {
		panic(errPageBoundsOutOfRange(i, j, numRows))
	}
	if i > j {
		panic(errPageBoundsOutOfRange(i, j, numRows))
	}

	rowIndex0 := int64(0)
	rowIndex1 := int64(len(page.repetitionLevels))
	rowIndex2 := int64(len(page.repetitionLevels))

	for k, def := range page.repetitionLevels {
		if def != page.maxRepetitionLevel {
			if rowIndex0 == i {
				rowIndex1 = int64(k)
			}
			if rowIndex0 == j {
				rowIndex2 = int64(k)
			}
			rowIndex0++
		}
	}

	numNulls1 := int64(countLevelsNotEqual(page.definitionLevels[:rowIndex1], page.maxDefinitionLevel))
	numNulls2 := int64(countLevelsNotEqual(page.definitionLevels[rowIndex1:rowIndex2], page.maxDefinitionLevel))

	i = rowIndex1 - numNulls1
	j = rowIndex2 - (numNulls1 + numNulls2)

	return newRepeatedPage(
		page.base.Slice(i, j),
		page.maxRepetitionLevel,
		page.maxDefinitionLevel,
		page.repetitionLevels[rowIndex1:rowIndex2],
		page.definitionLevels[rowIndex1:rowIndex2],
	)
}

func (page *repeatedPage) Size() int64 {
	return sizeOfInt8(page.repetitionLevels) + sizeOfInt8(page.definitionLevels) + page.base.Size()
}

func (page *repeatedPage) RepetitionLevels() []int8 {
	return page.repetitionLevels
}

func (page *repeatedPage) DefinitionLevels() []int8 {
	return page.definitionLevels
}

func (page *repeatedPage) WriteTo(e encoding.Encoder) error {
	return page.base.WriteTo(e)
}

func (page *repeatedPage) Values() ValueReader {
	return &repeatedPageReader{page: page}
}

func (page *repeatedPage) Buffer() BufferedPage {
	return page
}

type repeatedPageReader struct {
	page   *repeatedPage
	values ValueReader
	offset int
}

func (r *repeatedPageReader) ReadValues(values []Value) (n int, err error) {
	if r.values == nil {
		r.values = r.page.base.Values()
	}
	maxDefinitionLevel := r.page.maxDefinitionLevel
	columnIndex := ^int16(r.page.Column())

	for n < len(values) && r.offset < len(r.page.definitionLevels) {
		for n < len(values) && r.offset < len(r.page.definitionLevels) && r.page.definitionLevels[r.offset] != maxDefinitionLevel {
			values[n] = Value{
				repetitionLevel: r.page.repetitionLevels[r.offset],
				definitionLevel: r.page.definitionLevels[r.offset],
				columnIndex:     columnIndex,
			}
			r.offset++
			n++
		}

		i := n
		j := r.offset
		for i < len(values) && j < len(r.page.definitionLevels) && r.page.definitionLevels[j] == maxDefinitionLevel {
			i++
			j++
		}

		if n < i {
			for j, err = r.values.ReadValues(values[n:i]); j > 0; j-- {
				values[n].repetitionLevel = r.page.repetitionLevels[r.offset]
				values[n].definitionLevel = maxDefinitionLevel
				r.offset++
				n++
			}
			if err != nil && err != io.EOF {
				return n, err
			}
		}
	}

	if r.offset == len(r.page.definitionLevels) {
		err = io.EOF
	}
	return n, err
}

type byteArrayPage struct {
	values      encoding.ByteArrayList
	columnIndex int16
}

func (page *byteArrayPage) Column() int { return int(^page.columnIndex) }

func (page *byteArrayPage) Dictionary() Dictionary { return nil }

func (page *byteArrayPage) NumRows() int64 { return int64(page.values.Len()) }

func (page *byteArrayPage) NumValues() int64 { return int64(page.values.Len()) }

func (page *byteArrayPage) NumNulls() int64 { return 0 }

func (page *byteArrayPage) min() (min []byte) {
	if page.values.Len() > 0 {
		min = page.values.Index(0)
		for i := 1; i < page.values.Len(); i++ {
			v := page.values.Index(i)
			if string(v) < string(min) {
				min = v
			}
		}
	}
	return min
}

func (page *byteArrayPage) max() (max []byte) {
	if page.values.Len() > 0 {
		max = page.values.Index(0)
		for i := 1; i < page.values.Len(); i++ {
			v := page.values.Index(i)
			if string(v) > string(max) {
				max = v
			}
		}
	}
	return max
}

func (page *byteArrayPage) bounds() (min, max []byte) {
	if page.values.Len() > 0 {
		min = page.values.Index(0)
		max = min

		for i := 1; i < page.values.Len(); i++ {
			v := page.values.Index(i)
			switch {
			case string(v) < string(min):
				min = v
			case string(v) > string(max):
				max = v
			}
		}
	}
	return min, max
}

func (page *byteArrayPage) Bounds() (min, max Value, ok bool) {
	if ok = page.values.Len() > 0; ok {
		minBytes, maxBytes := page.bounds()
		min = makeValueBytes(ByteArray, minBytes)
		max = makeValueBytes(ByteArray, maxBytes)
	}
	return min, max, ok
}

func (page *byteArrayPage) Clone() BufferedPage {
	return &byteArrayPage{
		values:      page.values.Clone(),
		columnIndex: page.columnIndex,
	}
}

func (page *byteArrayPage) Slice(i, j int64) BufferedPage {
	return &byteArrayPage{
		values:      page.values.Slice(int(i), int(j)),
		columnIndex: page.columnIndex,
	}
}

func (page *byteArrayPage) Size() int64 { return page.values.Size() }

func (page *byteArrayPage) RepetitionLevels() []int8 { return nil }

func (page *byteArrayPage) DefinitionLevels() []int8 { return nil }

func (page *byteArrayPage) WriteTo(e encoding.Encoder) error { return e.EncodeByteArray(page.values) }

func (page *byteArrayPage) Values() ValueReader { return &byteArrayPageReader{page: page} }

func (page *byteArrayPage) Buffer() BufferedPage { return page }

type byteArrayPageReader struct {
	page   *byteArrayPage
	offset int
}

func (r *byteArrayPageReader) Read(b []byte) (int, error) {
	_, n, err := r.readByteArrays(b)
	return n, err
}

func (r *byteArrayPageReader) ReadRequired(values []byte) (int, error) {
	return r.ReadByteArrays(values)
}

func (r *byteArrayPageReader) ReadByteArrays(values []byte) (int, error) {
	n, _, err := r.readByteArrays(values)
	return n, err
}

func (r *byteArrayPageReader) readByteArrays(values []byte) (c, n int, err error) {
	for r.offset < r.page.values.Len() {
		b := r.page.values.Index(r.offset)
		k := plain.ByteArrayLengthSize + len(b)
		if k > (len(values) - n) {
			break
		}
		plain.PutByteArrayLength(values[n:], len(b))
		n += plain.ByteArrayLengthSize
		n += copy(values[n:], b)
		r.offset++
		c++
	}
	if r.offset == r.page.values.Len() {
		err = io.EOF
	} else if n == 0 && len(values) > 0 {
		err = io.ErrShortBuffer
	}
	return c, n, err
}

func (r *byteArrayPageReader) ReadValues(values []Value) (n int, err error) {
	for n < len(values) && r.offset < r.page.values.Len() {
		values[n] = makeValueBytes(ByteArray, r.page.values.Index(r.offset))
		values[n].columnIndex = r.page.columnIndex
		r.offset++
		n++
	}
	if r.offset == r.page.values.Len() {
		err = io.EOF
	}
	return n, err
}

type fixedLenByteArrayPage struct {
	size        int
	data        []byte
	columnIndex int16
}

func (page *fixedLenByteArrayPage) Column() int { return int(^page.columnIndex) }

func (page *fixedLenByteArrayPage) Dictionary() Dictionary { return nil }

func (page *fixedLenByteArrayPage) NumRows() int64 { return int64(len(page.data) / page.size) }

func (page *fixedLenByteArrayPage) NumValues() int64 { return int64(len(page.data) / page.size) }

func (page *fixedLenByteArrayPage) NumNulls() int64 { return 0 }

func (page *fixedLenByteArrayPage) min() []byte {
	return bits.MinFixedLenByteArray(page.size, page.data)
}

func (page *fixedLenByteArrayPage) max() []byte {
	return bits.MaxFixedLenByteArray(page.size, page.data)
}

func (page *fixedLenByteArrayPage) bounds() (min, max []byte) {
	return bits.MinMaxFixedLenByteArray(page.size, page.data)
}

func (page *fixedLenByteArrayPage) Bounds() (min, max Value, ok bool) {
	if ok = len(page.data) > 0; ok {
		minBytes, maxBytes := page.bounds()
		min = makeValueBytes(FixedLenByteArray, minBytes)
		max = makeValueBytes(FixedLenByteArray, maxBytes)
	}
	return min, max, ok
}

func (page *fixedLenByteArrayPage) Clone() BufferedPage {
	return &fixedLenByteArrayPage{
		size:        page.size,
		data:        append([]byte{}, page.data...),
		columnIndex: page.columnIndex,
	}
}

func (page *fixedLenByteArrayPage) Slice(i, j int64) BufferedPage {
	return &fixedLenByteArrayPage{
		size:        page.size,
		data:        page.data[i*int64(page.size) : j*int64(page.size)],
		columnIndex: page.columnIndex,
	}
}

func (page *fixedLenByteArrayPage) Size() int64 { return sizeOfBytes(page.data) }

func (page *fixedLenByteArrayPage) RepetitionLevels() []int8 { return nil }

func (page *fixedLenByteArrayPage) DefinitionLevels() []int8 { return nil }

func (page *fixedLenByteArrayPage) WriteTo(e encoding.Encoder) error {
	return e.EncodeFixedLenByteArray(page.size, page.data)
}

func (page *fixedLenByteArrayPage) Values() ValueReader {
	return &fixedLenByteArrayPageReader{page: page}
}

func (page *fixedLenByteArrayPage) Buffer() BufferedPage { return page }

type fixedLenByteArrayPageReader struct {
	page   *fixedLenByteArrayPage
	offset int
}

func (r *fixedLenByteArrayPageReader) Read(b []byte) (n int, err error) {
	n, err = r.ReadFixedLenByteArrays(b)
	return n * r.page.size, err
}

func (r *fixedLenByteArrayPageReader) ReadRequired(values []byte) (int, error) {
	return r.ReadFixedLenByteArrays(values)
}

func (r *fixedLenByteArrayPageReader) ReadFixedLenByteArrays(values []byte) (n int, err error) {
	n = copy(values, r.page.data[r.offset:]) / r.page.size
	r.offset += n * r.page.size
	if r.offset == len(r.page.data) {
		err = io.EOF
	} else if n == 0 && len(values) > 0 {
		err = io.ErrShortBuffer
	}
	return n, err
}

func (r *fixedLenByteArrayPageReader) ReadValues(values []Value) (n int, err error) {
	for n < len(values) && r.offset < len(r.page.data) {
		values[n] = makeValueBytes(FixedLenByteArray, r.page.data[r.offset:r.offset+r.page.size])
		values[n].columnIndex = r.page.columnIndex
		r.offset += r.page.size
		n++
	}
	if r.offset == len(r.page.data) {
		err = io.EOF
	}
	return n, err
}
