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
	cursor := uint32(0)
	previousEnd := uint64(0)
	for _, record := range records {
		end := cursor + record.Length
		if end > uint32(len(contiguousPages)) {
			return nil, fmt.Errorf("record out of bounds while reading pages: %d, %d, %d, %d", cursor, record.Length, end, len(contiguousPages))
		}

		if previousEnd != record.Start && previousEnd != 0 {
			return nil, fmt.Errorf("non-contiguous pages requested from pageReader: %d, %+v", previousEnd, record)
		}

		slicePages = append(slicePages, contiguousPages[cursor:end])
		cursor += record.Length
		previousEnd = record.Start + uint64(record.Length)
	}

	return slicePages, nil
}
