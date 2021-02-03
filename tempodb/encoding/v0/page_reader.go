package v0

import (
	"fmt"
	"io"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

type pageReader struct {
	r io.ReaderAt
}

// NewPageReader returns a new v0 pageReader.  A v0 pageReader
// is basically a no-op.  It retrieves the requested byte
// ranges and returns them as is.
// A pages "format" is a contiguous collection of objects
// | -- object -- | -- object -- | ...
func NewPageReader(r io.ReaderAt) common.PageReader {
	return &pageReader{
		r: r,
	}
}

// jpe : tests
// Read returns the pages requested in the passed records.  It
// assumes that if there are multiple records they are ordered
// and contiguous
func (r *pageReader) Read(records []*common.Record) ([][]byte, error) {
	if len(records) == 0 {
		return nil, nil
	}

	start := records[0].Start
	length := uint32(0)
	for _, record := range records {
		length += record.Length
	}

	contiguousPages := make([]byte, length)
	_, err := r.r.ReadAt(contiguousPages, int64(start))
	if err != nil {
		return nil, err
	}

	slicePages := make([][]byte, 0, len(records))
	cursor := 0
	for _, record := range records {
		if cursor+int(record.Length) > len(contiguousPages) {
			return nil, fmt.Errorf("record out of bounds while reading pages: %d, %d, %d", cursor, record.Length, len(contiguousPages))
		}

		slicePages = append(slicePages, contiguousPages[cursor:cursor+int(record.Length)])
		cursor += int(record.Length)
	}

	return slicePages, nil
}
