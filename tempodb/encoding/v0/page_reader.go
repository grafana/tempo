package v0

import (
	"io"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

type pageReader struct {
	r io.ReaderAt
}

// NewPageReader returns a new v0 pageReader.  A v0 pageReader
// is basically a no-op.  It retrieves the requested byte
// ranges and returns them as is.
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
	start = 0
	for _, record := range records {
		slicePages = append(slicePages, contiguousPages[start:int(record.Length)])
		start += uint64(record.Length)
	}

	return slicePages, nil
}
