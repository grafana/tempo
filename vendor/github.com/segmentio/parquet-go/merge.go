package parquet

import (
	"container/heap"
	"fmt"
	"io"
)

type mergedRowGroup struct {
	multiRowGroup
	sorting []SortingColumn
	compare func(Row, Row) int
}

func (m *mergedRowGroup) SortingColumns() []SortingColumn {
	return m.sorting
}

func (m *mergedRowGroup) Rows() Rows {
	// The row group needs to respect a sorting order; the merged row reader
	// uses a heap to merge rows from the row groups.
	rows := make([]Rows, len(m.rowGroups))
	for i := range rows {
		rows[i] = m.rowGroups[i].Rows()
	}
	return &mergedRowGroupRows{
		merge: mergedRowReader{
			compare: m.compare,
			readers: makeBufferedRowReaders(len(rows), func(i int) RowReader { return rows[i] }),
		},
		rows:   rows,
		schema: m.schema,
	}
}

type mergedRowGroupRows struct {
	merge     mergedRowReader
	rowIndex  int64
	seekToRow int64
	rows      []Rows
	schema    *Schema
}

func (r *mergedRowGroupRows) Close() (lastErr error) {
	r.merge.close()
	r.rowIndex = 0
	r.seekToRow = 0

	for _, rows := range r.rows {
		if err := rows.Close(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

func (r *mergedRowGroupRows) ReadRows(rows []Row) (int, error) {
	for r.rowIndex < r.seekToRow {
		n := int(r.seekToRow - r.rowIndex)
		if n > len(rows) {
			n = len(rows)
		}
		n, err := r.merge.ReadRows(rows[:n])
		if err != nil {
			return 0, err
		}
		r.rowIndex += int64(n)
	}

	return r.merge.ReadRows(rows)
}

func (r *mergedRowGroupRows) SeekToRow(rowIndex int64) error {
	if rowIndex >= r.rowIndex {
		r.seekToRow = rowIndex
		return nil
	}
	return fmt.Errorf("SeekToRow: merged row reader cannot seek backward from row %d to %d", r.rowIndex, rowIndex)
}

func (r *mergedRowGroupRows) Schema() *Schema {
	return r.schema
}

func MergeRowReaders(readers []RowReader, compare func(Row, Row) int) RowReader {
	return &mergedRowReader{
		compare: compare,
		readers: makeBufferedRowReaders(len(readers), func(i int) RowReader { return readers[i] }),
	}
}

func makeBufferedRowReaders(numReaders int, readerAt func(int) RowReader) []*bufferedRowReader {
	buffers := make([]bufferedRowReader, numReaders)
	readers := make([]*bufferedRowReader, numReaders)

	for i := range readers {
		buffers[i].rows = readerAt(i)
		readers[i] = &buffers[i]
	}

	return readers
}

type mergedRowReader struct {
	compare     func(Row, Row) int
	readers     []*bufferedRowReader
	initialized bool
}

func (m *mergedRowReader) initialize() error {
	for i, r := range m.readers {
		switch err := r.read(); err {
		case nil:
		case io.EOF:
			m.readers[i] = nil
		default:
			m.readers = nil
			return err
		}
	}

	n := 0
	for _, r := range m.readers {
		if r != nil {
			m.readers[n] = r
			n++
		}
	}

	clear := m.readers[n:]
	for i := range clear {
		clear[i] = nil
	}

	m.readers = m.readers[:n]
	heap.Init(m)
	return nil
}

func (m *mergedRowReader) close() {
	for _, r := range m.readers {
		r.close()
	}
	m.readers = nil
}

func (m *mergedRowReader) ReadRows(rows []Row) (n int, err error) {
	if !m.initialized {
		m.initialized = true

		if err := m.initialize(); err != nil {
			return 0, err
		}
	}

	for n < len(rows) && len(m.readers) != 0 {
		r := m.readers[0]

		rows[n] = append(rows[n][:0], r.head()...)
		n++

		if err := r.next(); err != nil {
			if err != io.EOF {
				return n, err
			}
			heap.Pop(m)
		} else {
			heap.Fix(m, 0)
		}
	}

	if len(m.readers) == 0 {
		err = io.EOF
	}

	return n, err
}

func (m *mergedRowReader) Less(i, j int) bool {
	return m.compare(m.readers[i].head(), m.readers[j].head()) < 0
}

func (m *mergedRowReader) Len() int {
	return len(m.readers)
}

func (m *mergedRowReader) Swap(i, j int) {
	m.readers[i], m.readers[j] = m.readers[j], m.readers[i]
}

func (m *mergedRowReader) Push(x interface{}) {
	panic("NOT IMPLEMENTED")
}

func (m *mergedRowReader) Pop() interface{} {
	i := len(m.readers) - 1
	r := m.readers[i]
	m.readers = m.readers[:i]
	return r
}

type bufferedRowReader struct {
	rows RowReader
	off  int32
	end  int32
	buf  [10]Row
}

func (r *bufferedRowReader) head() Row {
	return r.buf[r.off]
}

func (r *bufferedRowReader) next() error {
	if r.off++; r.off == r.end {
		r.off = 0
		r.end = 0
		return r.read()
	}
	return nil
}

func (r *bufferedRowReader) read() error {
	if r.rows == nil {
		return io.EOF
	}
	n, err := r.rows.ReadRows(r.buf[r.end:])
	if err != nil && n == 0 {
		return err
	}
	r.end += int32(n)
	return nil
}

func (r *bufferedRowReader) close() {
	r.rows = nil
	r.off = 0
	r.end = 0
}

var (
	_ RowReaderWithSchema = (*mergedRowGroupRows)(nil)
)
