package parquet

import (
	"fmt"
	"io"

	"github.com/segmentio/parquet-go/internal/debug"
)

// RowGroup is an interface representing a parquet row group. From the Parquet
// docs, a RowGroup is "a logical horizontal partitioning of the data into rows.
// There is no physical structure that is guaranteed for a row group. A row
// group consists of a column chunk for each column in the dataset."
//
// https://github.com/apache/parquet-format#glossary
type RowGroup interface {
	// Returns the number of rows in the group.
	NumRows() int64

	// Returns the list of column chunks in this row group. The chunks are
	// ordered in the order of leaf columns from the row group's schema.
	//
	// If the underlying implementation is not read-only, the returned
	// parquet.ColumnChunk may implement other interfaces: for example,
	// parquet.ColumnBuffer if the chunk is backed by an in-memory buffer,
	// or typed writer interfaces like parquet.Int32Writer depending on the
	// underlying type of values that can be written to the chunk.
	//
	// As an optimization, the row group may return the same slice across
	// multiple calls to this method. Applications should treat the returned
	// slice as read-only.
	ColumnChunks() []ColumnChunk

	// Returns the schema of rows in the group.
	Schema() *Schema

	// Returns the list of sorting columns describing how rows are sorted in the
	// group.
	//
	// The method will return an empty slice if the rows are not sorted.
	SortingColumns() []SortingColumn

	// Returns a reader exposing the rows of the row group.
	//
	// As an optimization, the returned parquet.Rows object may implement
	// parquet.RowWriterTo, and test the RowWriter it receives for an
	// implementation of the parquet.RowGroupWriter interface.
	//
	// This optimization mechanism is leveraged by the parquet.CopyRows function
	// to skip the generic row-by-row copy algorithm and delegate the copy logic
	// to the parquet.Rows object.
	Rows() Rows
}

// Rows is an interface implemented by row readers returned by calling the Rows
// method of RowGroup instances.
//
// Applications should call Close when they are done using a Rows instance in
// order to release the underlying resources held by the row sequence.
//
// After calling Close, all attempts to read more rows will return io.EOF.
type Rows interface {
	RowReaderWithSchema
	RowSeeker
	io.Closer
}

// RowGroupReader is an interface implemented by types that expose sequences of
// row groups to the application.
type RowGroupReader interface {
	ReadRowGroup() (RowGroup, error)
}

// RowGroupWriter is an interface implemented by types that allow the program
// to write row groups.
type RowGroupWriter interface {
	WriteRowGroup(RowGroup) (int64, error)
}

// SortingColumn represents a column by which a row group is sorted.
type SortingColumn interface {
	// Returns the path of the column in the row group schema, omitting the name
	// of the root node.
	Path() []string

	// Returns true if the column will sort values in descending order.
	Descending() bool

	// Returns true if the column will put null values at the beginning.
	NullsFirst() bool
}

// Ascending constructs a SortingColumn value which dictates to sort the column
// at the path given as argument in ascending order.
func Ascending(path ...string) SortingColumn { return ascending(path) }

// Descending constructs a SortingColumn value which dictates to sort the column
// at the path given as argument in descending order.
func Descending(path ...string) SortingColumn { return descending(path) }

// NullsFirst wraps the SortingColumn passed as argument so that it instructs
// the row group to place null values first in the column.
func NullsFirst(sortingColumn SortingColumn) SortingColumn { return nullsFirst{sortingColumn} }

type ascending []string

func (asc ascending) String() string   { return fmt.Sprintf("ascending(%s)", columnPath(asc)) }
func (asc ascending) Path() []string   { return asc }
func (asc ascending) Descending() bool { return false }
func (asc ascending) NullsFirst() bool { return false }

type descending []string

func (desc descending) String() string   { return fmt.Sprintf("descending(%s)", columnPath(desc)) }
func (desc descending) Path() []string   { return desc }
func (desc descending) Descending() bool { return true }
func (desc descending) NullsFirst() bool { return false }

type nullsFirst struct{ SortingColumn }

func (nf nullsFirst) String() string   { return fmt.Sprintf("nulls_first+%s", nf.SortingColumn) }
func (nf nullsFirst) NullsFirst() bool { return true }

func searchSortingColumn(sortingColumns []SortingColumn, path columnPath) int {
	// There are usually a few sorting columns in a row group, so the linear
	// scan is the fastest option and works whether the sorting column list
	// is sorted or not. Please revisit this decision if this code path ends
	// up being more costly than necessary.
	for i, sorting := range sortingColumns {
		if path.equal(sorting.Path()) {
			return i
		}
	}
	return len(sortingColumns)
}

func sortingColumnsHavePrefix(sortingColumns, prefix []SortingColumn) bool {
	if len(sortingColumns) < len(prefix) {
		return false
	}
	for i, sortingColumn := range prefix {
		if !sortingColumnsAreEqual(sortingColumns[i], sortingColumn) {
			return false
		}
	}
	return true
}

func sortingColumnsAreEqual(s1, s2 SortingColumn) bool {
	path1 := columnPath(s1.Path())
	path2 := columnPath(s2.Path())
	return path1.equal(path2) && s1.Descending() == s2.Descending() && s1.NullsFirst() == s2.NullsFirst()
}

// MergeRowGroups constructs a row group which is a merged view of rowGroups. If
// rowGroups are sorted and the passed options include sorting, the merged row
// group will also be sorted.
//
// The function validates the input to ensure that the merge operation is
// possible, ensuring that the schemas match or can be converted to an
// optionally configured target schema passed as argument in the option list.
//
// The sorting columns of each row group are also consulted to determine whether
// the output can be represented. If sorting columns are configured on the merge
// they must be a prefix of sorting columns of all row groups being merged.
func MergeRowGroups(rowGroups []RowGroup, options ...RowGroupOption) (RowGroup, error) {
	config, err := NewRowGroupConfig(options...)
	if err != nil {
		return nil, err
	}

	schema := config.Schema
	if len(rowGroups) == 0 {
		return newEmptyRowGroup(schema), nil
	}
	if schema == nil {
		schema = rowGroups[0].Schema()

		for _, rowGroup := range rowGroups[1:] {
			if !nodesAreEqual(schema, rowGroup.Schema()) {
				return nil, ErrRowGroupSchemaMismatch
			}
		}
	}

	mergedRowGroups := make([]RowGroup, len(rowGroups))
	copy(mergedRowGroups, rowGroups)

	for i, rowGroup := range mergedRowGroups {
		if rowGroupSchema := rowGroup.Schema(); !nodesAreEqual(schema, rowGroupSchema) {
			conv, err := Convert(schema, rowGroupSchema)
			if err != nil {
				return nil, fmt.Errorf("cannot merge row groups: %w", err)
			}
			mergedRowGroups[i] = ConvertRowGroup(rowGroup, conv)
		}
	}

	m := &mergedRowGroup{sorting: config.SortingColumns}
	m.init(schema, mergedRowGroups)

	if len(m.sorting) == 0 {
		// When the row group has no ordering, use a simpler version of the
		// merger which simply concatenates rows from each of the row groups.
		// This is preferable because it makes the output deterministic, the
		// heap merge may otherwise reorder rows across groups.
		return &m.multiRowGroup, nil
	}

	for _, rowGroup := range m.rowGroups {
		if !sortingColumnsHavePrefix(rowGroup.SortingColumns(), m.sorting) {
			return nil, ErrRowGroupSortingColumnsMismatch
		}
	}

	m.sortFuncs = make([]columnSortFunc, len(m.sorting))
	forEachLeafColumnOf(schema, func(leaf leafColumn) {
		if sortingIndex := searchSortingColumn(m.sorting, leaf.path); sortingIndex < len(m.sorting) {
			m.sortFuncs[sortingIndex] = columnSortFunc{
				columnIndex: leaf.columnIndex,
				compare: sortFuncOf(
					leaf.node.Type(),
					&SortConfig{
						MaxRepetitionLevel: int(leaf.maxRepetitionLevel),
						MaxDefinitionLevel: int(leaf.maxDefinitionLevel),
						Descending:         m.sorting[sortingIndex].Descending(),
						NullsFirst:         m.sorting[sortingIndex].NullsFirst(),
					},
				),
			}
		}
	})

	return m, nil
}

type rowGroup struct {
	schema  *Schema
	numRows int64
	columns []ColumnChunk
	sorting []SortingColumn
}

func (r *rowGroup) NumRows() int64                  { return r.numRows }
func (r *rowGroup) ColumnChunks() []ColumnChunk     { return r.columns }
func (r *rowGroup) SortingColumns() []SortingColumn { return r.sorting }
func (r *rowGroup) Schema() *Schema                 { return r.schema }
func (r *rowGroup) Rows() Rows                      { return &rowGroupRows{rowGroup: r} }

func NewRowGroupRowReader(rowGroup RowGroup) Rows {
	return &rowGroupRows{rowGroup: rowGroup}
}

type rowGroupRows struct {
	rowGroup RowGroup
	buffers  []Value
	readers  []asyncPages
	columns  []columnChunkRows
	inited   bool
	closed   bool
	done     chan<- struct{}
}

type columnChunkRows struct {
	rows   int64
	offset int32
	length int32
	page   Page
	values ValueReader
}

const columnBufferSize = defaultValueBufferSize

func (r *rowGroupRows) buffer(i int) []Value {
	j := (i + 0) * columnBufferSize
	k := (i + 1) * columnBufferSize
	return r.buffers[j:k:k]
}

func (r *rowGroupRows) init() {
	columns := r.rowGroup.ColumnChunks()

	r.buffers = make([]Value, len(columns)*columnBufferSize)
	r.readers = make([]asyncPages, len(columns))
	r.columns = make([]columnChunkRows, len(columns))

	done := make(chan struct{})
	r.done = done

	for i, column := range columns {
		r.readers[i].init(column.Pages(), done)
	}

	r.inited = true
	// This finalizer is used to ensure that the goroutines started by calling
	// init on the underlying page readers will be shutdown in the event that
	// Close isn't called and the rowGroupRows object is garbage collected.
	debug.SetFinalizer(r, func(r *rowGroupRows) { r.Close() })
}

func (r *rowGroupRows) clear() {
	for i := range r.columns {
		Release(r.columns[i].page)
	}

	for i := range r.columns {
		r.columns[i] = columnChunkRows{}
	}

	for i := range r.buffers {
		r.buffers[i] = Value{}
	}
}

func (r *rowGroupRows) Reset() {
	for i := range r.readers {
		// Ignore errors because we are resetting the reader, if the error
		// persists we will see it on the next read, and otherwise we can
		// read back from the beginning.
		r.readers[i].SeekToRow(0)
	}
	r.clear()
}

func (r *rowGroupRows) Close() error {
	var lastErr error

	if r.done != nil {
		close(r.done)
		r.done = nil
	}

	for i := range r.readers {
		if err := r.readers[i].Close(); err != nil {
			lastErr = err
		}
	}

	r.clear()
	r.inited = true
	r.closed = true
	return lastErr
}

func (r *rowGroupRows) SeekToRow(rowIndex int64) error {
	var lastErr error

	if r.closed {
		return io.ErrClosedPipe
	}

	if !r.inited {
		r.init()
	}

	for i := range r.readers {
		if err := r.readers[i].SeekToRow(rowIndex); err != nil {
			lastErr = err
		}
	}

	r.clear()
	return lastErr
}

func (r *rowGroupRows) ReadRows(rows []Row) (int, error) {
	if r.closed {
		return 0, io.EOF
	}

	if !r.inited {
		r.init()
	}

	// Limit the number of rows that we read to the smallest number of rows
	// remaining in the current page of each column. This is necessary because
	// the pointers exposed to the returned rows need to remain valid until the
	// next call to ReadRows, SeekToRow, Reset, or Close. If we release one of
	// the columns' page, the rows that were already read during the ReadRows
	// call would be invalidated, and might reference memory locations that have
	// been reused due to pooling of page buffers.
	numRows := int64(len(rows))

	for i := range r.columns {
		c := &r.columns[i]
		// When all rows of the current page of a column have been consumed we
		// have to read the next page. This will effectively invalidate all
		// pointers of values previously held in the page, which is valid if
		// the application respects the RowReader interface and does not retain
		// parquet values without cloning them first.
		for c.rows == 0 {
			var err error
			clearValues(r.buffer(i))

			c.offset = 0
			c.length = 0
			c.values = nil
			Release(c.page)

			c.page, err = r.readers[i].ReadPage()
			if err != nil {
				if err != io.EOF {
					return 0, err
				}
				break
			}

			c.rows = c.page.NumRows()
			c.values = c.page.Values()
		}

		if c.rows < numRows {
			numRows = c.rows
		}
	}

	for i := range rows {
		rows[i] = rows[i][:0]
	}

	if numRows == 0 {
		return 0, io.EOF
	}

	n, err := r.Schema().readRows(r, rows[:numRows], 0)

	for i := range r.columns {
		r.columns[i].rows -= int64(n)
	}

	return n, err
}

func (r *rowGroupRows) Schema() *Schema {
	return r.rowGroup.Schema()
}

type seekRowGroup struct {
	base    RowGroup
	seek    int64
	columns []ColumnChunk
}

func (g *seekRowGroup) NumRows() int64 {
	return g.base.NumRows() - g.seek
}

func (g *seekRowGroup) ColumnChunks() []ColumnChunk {
	return g.columns
}

func (g *seekRowGroup) Schema() *Schema {
	return g.base.Schema()
}

func (g *seekRowGroup) SortingColumns() []SortingColumn {
	return g.base.SortingColumns()
}

func (g *seekRowGroup) Rows() Rows {
	rows := g.base.Rows()
	rows.SeekToRow(g.seek)
	return rows
}

type seekColumnChunk struct {
	base ColumnChunk
	seek int64
}

func (c *seekColumnChunk) Type() Type {
	return c.base.Type()
}

func (c *seekColumnChunk) Column() int {
	return c.base.Column()
}

func (c *seekColumnChunk) Pages() Pages {
	pages := c.base.Pages()
	pages.SeekToRow(c.seek)
	return pages
}

func (c *seekColumnChunk) ColumnIndex() ColumnIndex {
	return c.base.ColumnIndex()
}

func (c *seekColumnChunk) OffsetIndex() OffsetIndex {
	return c.base.OffsetIndex()
}

func (c *seekColumnChunk) BloomFilter() BloomFilter {
	return c.base.BloomFilter()
}

func (c *seekColumnChunk) NumValues() int64 {
	return c.base.NumValues()
}

type emptyRowGroup struct {
	schema  *Schema
	columns []ColumnChunk
}

func newEmptyRowGroup(schema *Schema) *emptyRowGroup {
	columns := schema.Columns()
	rowGroup := &emptyRowGroup{
		schema:  schema,
		columns: make([]ColumnChunk, len(columns)),
	}
	emptyColumnChunks := make([]emptyColumnChunk, len(columns))
	for i, column := range schema.Columns() {
		leaf, _ := schema.Lookup(column...)
		emptyColumnChunks[i].typ = leaf.Node.Type()
		emptyColumnChunks[i].column = int16(leaf.ColumnIndex)
		rowGroup.columns[i] = &emptyColumnChunks[i]
	}
	return rowGroup
}

func (g *emptyRowGroup) NumRows() int64                  { return 0 }
func (g *emptyRowGroup) ColumnChunks() []ColumnChunk     { return g.columns }
func (g *emptyRowGroup) Schema() *Schema                 { return g.schema }
func (g *emptyRowGroup) SortingColumns() []SortingColumn { return nil }
func (g *emptyRowGroup) Rows() Rows                      { return emptyRows{g.schema} }

type emptyColumnChunk struct {
	typ    Type
	column int16
}

func (c *emptyColumnChunk) Type() Type               { return c.typ }
func (c *emptyColumnChunk) Column() int              { return int(c.column) }
func (c *emptyColumnChunk) Pages() Pages             { return emptyPages{} }
func (c *emptyColumnChunk) ColumnIndex() ColumnIndex { return emptyColumnIndex{} }
func (c *emptyColumnChunk) OffsetIndex() OffsetIndex { return emptyOffsetIndex{} }
func (c *emptyColumnChunk) BloomFilter() BloomFilter { return emptyBloomFilter{} }
func (c *emptyColumnChunk) NumValues() int64         { return 0 }

type emptyBloomFilter struct{}

func (emptyBloomFilter) ReadAt([]byte, int64) (int, error) { return 0, io.EOF }
func (emptyBloomFilter) Size() int64                       { return 0 }
func (emptyBloomFilter) Check(Value) (bool, error)         { return false, nil }

type emptyRows struct{ schema *Schema }

func (r emptyRows) Close() error                         { return nil }
func (r emptyRows) Schema() *Schema                      { return r.schema }
func (r emptyRows) ReadRows([]Row) (int, error)          { return 0, io.EOF }
func (r emptyRows) SeekToRow(int64) error                { return nil }
func (r emptyRows) WriteRowsTo(RowWriter) (int64, error) { return 0, nil }

type emptyPages struct{}

func (emptyPages) ReadPage() (Page, error) { return nil, io.EOF }
func (emptyPages) SeekToRow(int64) error   { return nil }
func (emptyPages) Close() error            { return nil }

var (
	_ RowReaderWithSchema = (*rowGroupRows)(nil)
	//_ RowWriterTo         = (*rowGroupRows)(nil)

	_ RowReaderWithSchema = emptyRows{}
	_ RowWriterTo         = emptyRows{}
)
