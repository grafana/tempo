package parquet

import (
	"io"
)

// The ColumnChunk interface represents individual columns of a row group.
type ColumnChunk interface {
	// Returns the column type.
	Type() Type

	// Returns the index of this column in its parent row group.
	Column() int

	// Returns a reader exposing the pages of the column.
	Pages() Pages

	// Returns the components of the page index for this column chunk,
	// containing details about the content and location of pages within the
	// chunk.
	//
	// Note that the returned value may be the same across calls to these
	// methods, programs must treat those as read-only.
	//
	// If the column chunk does not have a page index, the methods return nil.
	ColumnIndex() ColumnIndex
	OffsetIndex() OffsetIndex
	BloomFilter() BloomFilter

	// Returns the number of values in the column chunk.
	//
	// This quantity may differ from the number of rows in the parent row group
	// because repeated columns may hold zero or more values per row.
	NumValues() int64
}

// Pages is an interface implemented by page readers returned by calling the
// Pages method of ColumnChunk instances.
type Pages interface {
	PageReader
	RowSeeker
}

type pageAndValueWriter interface {
	PageWriter
	ValueWriter
}

type columnChunkReader struct {
	// These two fields must be configured to initialize the reader.
	column ColumnChunk // column chunk that the reader
	buffer []Value     // buffer holding values read from the pages
	// The rest of the fields are used to manage the state of the reader as it
	// consumes values from the underlying pages.
	offset int         // offset of the next value in the buffer
	page   Page        // current page where values are being read from
	reader Pages       // reader of column pages
	values ValueReader // reader for values from the current page
}

func (r *columnChunkReader) buffered() int {
	return len(r.buffer) - r.offset
}

func (r *columnChunkReader) seekToRow(rowIndex int64) error {
	// TODO: there are a few optimizations we can make here:
	// * is the row buffered already? => advance the offset
	// * is the row in the current page? => seek in values
	clearValues(r.buffer)
	r.buffer = r.buffer[:0]
	r.offset = 0
	r.page = nil
	r.values = nil

	if r.reader == nil {
		if r.column != nil {
			r.reader = r.column.Pages()
		}
	}

	if r.reader != nil {
		if err := r.reader.SeekToRow(rowIndex); err != nil {
			r.reader = nil
			r.column = nil
			return err
		}
	}
	return nil
}

func (r *columnChunkReader) readPage() (err error) {
	if r.page != nil {
		return nil
	}
	if r.reader == nil {
		if r.column == nil {
			return io.EOF
		}
		r.reader = r.column.Pages()
	}
	for {
		p, err := r.reader.ReadPage()
		if err != nil {
			return err
		}
		if p.NumValues() > 0 {
			r.page = p
			return nil
		}
	}
}

func (r *columnChunkReader) readValues() error {
	for {
		err := r.readValuesFromCurrentPage()
		if err == nil || err != io.EOF {
			return err
		}
		if err := r.readPage(); err != nil {
			return err
		}
	}
}

func (r *columnChunkReader) readValuesFromCurrentPage() error {
	if r.offset < len(r.buffer) {
		return nil
	}
	if r.page == nil {
		return io.EOF
	}
	if r.values == nil {
		r.values = r.page.Values()
	}
	n, err := r.values.ReadValues(r.buffer[:cap(r.buffer)])
	if err != nil && err == io.EOF {
		r.page, r.values = nil, nil
	}
	if n > 0 {
		err = nil
	}
	r.buffer = r.buffer[:n]
	r.offset = 0
	return err
}

func (r *columnChunkReader) writeBufferedRowsTo(w pageAndValueWriter, rowCount int64) (numRows int64, err error) {
	if rowCount == 0 {
		return 0, nil
	}

	for {
		for r.offset < len(r.buffer) {
			values := r.buffer[r.offset:]
			// We can only determine that the full row has been consumed if we
			// have more values in the buffer, and the next value is the start
			// of a new row. Otherwise, we have to load more values from the
			// page, which may yield EOF if all values have been consumed, in
			// which case we know that we have read the full row, and otherwise
			// we will enter this check again on the next loop iteration.
			if numRows == rowCount {
				if values[0].repetitionLevel == 0 {
					return numRows, nil
				}
				values, _ = splitRowValues(values)
			} else {
				values = limitRowValues(values, int(rowCount-numRows))
			}

			n, err := w.WriteValues(values)
			numRows += int64(countRowsOf(values[:n]))
			r.offset += n
			if err != nil {
				return numRows, err
			}
		}

		if err := r.readValuesFromCurrentPage(); err != nil {
			if err == io.EOF {
				err = nil
			}
			return numRows, err
		}
	}
}

func (r *columnChunkReader) writeRowsTo(w pageAndValueWriter, limit int64) (numRows int64, err error) {
	for numRows < limit {
		if r.values != nil {
			n, err := r.writeBufferedRowsTo(w, numRows-limit)
			numRows += n
			if err != nil || numRows == limit {
				return numRows, err
			}
		}

		r.buffer = r.buffer[:0]
		r.offset = 0

		for numRows < limit {
			p, err := r.reader.ReadPage()
			if err != nil {
				return numRows, err
			}

			pageRows := int64(p.NumRows())
			// When the page is fully contained in the remaining range of rows
			// that we intend to copy, we can use an optimized page copy rather
			// than writing rows one at a time.
			//
			// Data pages v1 do not expose the number of rows available, which
			// means we cannot take the optimized page copy path in those cases.
			if pageRows == 0 || int64(pageRows) > limit {
				r.values = p.Values()
				err := r.readValuesFromCurrentPage()
				if err == nil {
					// More values have been buffered, break out of the inner loop
					// to go back to the beginning of the outer loop and write
					// buffered values to the output.
					break
				}
				if err == io.EOF {
					// The page contained no values? Unclear if this is valid but
					// we can handle it by reading the next page.
					r.values = nil
					continue
				}
				return numRows, err
			}

			if _, err := w.WritePage(p); err != nil {
				return numRows, err
			}

			numRows += pageRows
		}
	}
	return numRows, nil
}

type columnReadRowFunc func(Row, int8, []columnChunkReader) (Row, error)

func columnReadRowFuncOf(node Node, columnIndex int, repetitionDepth int8) (int, columnReadRowFunc) {
	var read columnReadRowFunc

	if node.Repeated() {
		repetitionDepth++
	}

	if node.Leaf() {
		columnIndex, read = columnReadRowFuncOfLeaf(columnIndex, repetitionDepth)
	} else {
		columnIndex, read = columnReadRowFuncOfGroup(node, columnIndex, repetitionDepth)
	}

	if node.Repeated() {
		read = columnReadRowFuncOfRepeated(read, repetitionDepth)
	}

	return columnIndex, read
}

//go:noinline
func columnReadRowFuncOfRepeated(read columnReadRowFunc, repetitionDepth int8) columnReadRowFunc {
	return func(row Row, repetitionLevel int8, columns []columnChunkReader) (Row, error) {
		var err error

		for {
			n := len(row)

			if row, err = read(row, repetitionLevel, columns); err != nil {
				return row, err
			}
			if n == len(row) {
				return row, nil
			}

			repetitionLevel = repetitionDepth
		}
	}
}

//go:noinline
func columnReadRowFuncOfGroup(node Node, columnIndex int, repetitionDepth int8) (int, columnReadRowFunc) {
	fields := node.Fields()
	if len(fields) == 1 {
		// Small optimization for a somewhat common case of groups with a single
		// column (like nested list elements for example); there is no need to
		// loop over the group of a single element, we can simply skip to calling
		// the inner read function.
		return columnReadRowFuncOf(fields[0], columnIndex, repetitionDepth)
	}

	group := make([]columnReadRowFunc, len(fields))
	for i := range group {
		columnIndex, group[i] = columnReadRowFuncOf(fields[i], columnIndex, repetitionDepth)
	}

	return columnIndex, func(row Row, repetitionLevel int8, columns []columnChunkReader) (Row, error) {
		var err error

		for _, read := range group {
			if row, err = read(row, repetitionLevel, columns); err != nil {
				break
			}
		}

		return row, err
	}
}

//go:noinline
func columnReadRowFuncOfLeaf(columnIndex int, repetitionDepth int8) (int, columnReadRowFunc) {
	var read columnReadRowFunc

	if repetitionDepth == 0 {
		read = func(row Row, _ int8, columns []columnChunkReader) (Row, error) {
			col := &columns[columnIndex]

			for {
				if col.offset < len(col.buffer) {
					row = append(row, col.buffer[col.offset])
					col.offset++
					return row, nil
				}
				if err := col.readValues(); err != nil {
					return row, err
				}
			}
		}
	} else {
		read = func(row Row, repetitionLevel int8, columns []columnChunkReader) (Row, error) {
			col := &columns[columnIndex]

			for {
				if col.offset < len(col.buffer) {
					if col.buffer[col.offset].repetitionLevel == repetitionLevel {
						row = append(row, col.buffer[col.offset])
						col.offset++
					}
					return row, nil
				}
				if err := col.readValues(); err != nil {
					if repetitionLevel > 0 && err == io.EOF {
						err = nil
					}
					return row, err
				}
			}
		}
	}

	return columnIndex + 1, read
}

var (
	_ RowReaderWithSchema = (*Reader)(nil)
)
