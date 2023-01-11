package parquet

import "io"

// ScanRowReader constructs a RowReader which exposes rows from reader until
// the predicate returns false for one of the rows, or EOF is reached.
func ScanRowReader(reader RowReader, predicate func(Row) bool) RowReader {
	return &scanRowReader{reader: reader, predicate: predicate}
}

type scanRowReader struct {
	reader    RowReader
	predicate func(Row) bool
	done      bool
}

func (s *scanRowReader) ReadRows(rows []Row) (int, error) {
	if s.done {
		return 0, io.EOF
	}

	n, err := s.reader.ReadRows(rows)

	for i, row := range rows[:n] {
		if !s.predicate(row) {
			s.done = true
			return i, io.EOF
		}
	}

	return n, err
}
