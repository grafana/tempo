package v2

import (
	"context"
	"fmt"
	"math"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/base"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

const maxByte = byte(0xff)
const minByte = byte(0x00)

type indexReader struct {
	r backend.ContextReader

	pageSizeBytes int
	totalRecords  int

	pageCache map[int]*page // indexReader is not concurrency safe, but since it is currently used within one request it is fine.
}

// jpe - cache requested pages - needs total records.  write to header?

// NewIndexReader returns an index reader for a byte slice of marshalled
// ordered records.
// The index has not changed between v0 and v1.
func NewIndexReader(r backend.ContextReader, pageSizeBytes int, totalRecords int) (common.IndexReader, error) {
	return &indexReader{
		r: r,

		pageSizeBytes: pageSizeBytes,
		totalRecords:  totalRecords,

		pageCache: map[int]*page{},
	}, nil
}

func (r *indexReader) At(ctx context.Context, i int) (*common.Record, error) {
	if i < 0 || i >= r.totalRecords {
		return nil, nil
	}

	recordsPerPage := objectsPerPage(base.RecordLength, r.pageSizeBytes)
	if recordsPerPage == 0 {
		return nil, fmt.Errorf("page %d is too small for one record", r.pageSizeBytes)
	}
	pageIdx := i / recordsPerPage
	recordIdx := i % recordsPerPage

	page, err := r.getPage(ctx, pageIdx)
	if err != nil {
		return nil, err
	}

	if recordIdx >= len(page)/base.RecordLength {
		return nil, fmt.Errorf("unexpected out of bounds index %d, %d, %d, %d", i, pageIdx, recordIdx, len(page))
	}

	recordBytes := page[recordIdx*base.RecordLength : (recordIdx+1)*base.RecordLength]

	// double check the record is not all 0s.  this could occur if we read empty buffer space past the final
	// record in the final page
	allZeros := true
	for _, b := range recordBytes {
		if b != 0 {
			allZeros = false
			break
		}
	}
	if allZeros {
		return nil, fmt.Errorf("unexpected zero value record %d, %d, %d, %d", i, pageIdx, recordIdx, len(page))
	}

	return base.UnmarshalRecord(recordBytes), nil
}

func (r *indexReader) Find(ctx context.Context, id common.ID) (*common.Record, int, error) { // jpe test :(
	// recordsPerPage := objectsPerPage(v0.RecordLength, r.pageSizeBytes)
	// totalPages := totalPages(r.totalRecords, recordsPerPage)

	// // jpe pass min/max ids in here for better accuracy? do this: 64 bit ids
	// // guess the page assuming linear distribution of traceids
	// page := estimatePage(totalPages, id)

	return nil, 0, nil
}

func (r *indexReader) getPage(ctx context.Context, pageIdx int) ([]byte, error) {
	page, ok := r.pageCache[pageIdx]
	if ok {
		return page.data, nil
	}

	pageBuffer := make([]byte, r.pageSizeBytes)
	_, err := r.r.ReadAt(ctx, pageBuffer, int64(pageIdx*r.pageSizeBytes))
	if err != nil {
		return nil, err
	}

	page, err = unmarshalPageFromBytes(pageBuffer)
	if err != nil {
		return nil, err
	}

	r.pageCache[pageIdx] = page

	return page.data, nil
}

func estimatePage(totalPages int, id common.ID) int {
	// find the first non-zero byte
	firstIDByte := byte(0x00)
	for _, b := range id {
		if b != 0x00 {
			firstIDByte = b
			break
		}
	}

	place := float64(firstIDByte) / float64((maxByte - minByte))
	guessPage := int(math.Floor(place * float64(totalPages)))

	return guessPage
}
