package v2

import (
	"bytes"
	"context"
	"fmt"

	"github.com/grafana/tempo/pkg/sort"
	"go.opentelemetry.io/otel"

	"github.com/cespare/xxhash"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

var tracer = otel.Tracer("tempodb/encoding/v2")

type indexReader struct {
	r        backend.ContextReader
	recordRW RecordReaderWriter

	pageSizeBytes int
	totalRecords  int

	pageCache map[int]*page // indexReader is not concurrency safe, but since it is currently used within one request it is fine.
}

// NewIndexReader returns an index reader for a byte slice of marshalled
// ordered records.
// The index has not changed between v0 and v1.
func NewIndexReader(r backend.ContextReader, pageSizeBytes int, totalRecords int) (IndexReader, error) {
	return &indexReader{
		r:        r,
		recordRW: NewRecordReaderWriter(),

		pageSizeBytes: pageSizeBytes,
		totalRecords:  totalRecords,

		pageCache: map[int]*page{},
	}, nil
}

// At implements IndexReader
func (r *indexReader) At(ctx context.Context, i int) (*Record, error) {
	if i < 0 || i >= r.totalRecords {
		return nil, nil
	}

	recordLength := r.recordRW.RecordLength()

	recordsPerPage := objectsPerPage(recordLength, r.pageSizeBytes, IndexHeaderLength)
	if recordsPerPage == 0 {
		return nil, fmt.Errorf("page %d is too small for one record", r.pageSizeBytes)
	}
	pageIdx := i / recordsPerPage
	recordIdx := i % recordsPerPage

	page, err := r.getPage(ctx, pageIdx)
	if err != nil {
		return nil, err
	}

	if recordIdx >= len(page.data)/recordLength {
		return nil, fmt.Errorf("unexpected out of bounds index %d, %d, %d, %d", i, pageIdx, recordIdx, len(page.data))
	}

	recordBytes := page.data[recordIdx*recordLength : (recordIdx+1)*recordLength]

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
		return nil, fmt.Errorf("unexpected zero value record %d, %d, %d, %d", i, pageIdx, recordIdx, len(page.data))
	}

	record := r.recordRW.UnmarshalRecord(recordBytes)
	return &record, nil
}

// Find implements IndexReader
func (r *indexReader) Find(ctx context.Context, id common.ID) (*Record, int, error) {
	// with a linear distribution of trace ids we can actually do much better than a normal
	// binary search.  unfortunately there are edge cases which make this perform far worse.
	// for instance consider a set of trace ids what with 90% 64 bit ids and 10% 128 bit ids.
	ctx, span := tracer.Start(ctx, "indexReader.Find")
	defer span.End()

	i, err := sort.SearchWithErrors(r.totalRecords, func(i int) (bool, error) {
		record, err := r.At(ctx, i)
		if err != nil {
			return true, err
		}

		return bytes.Compare(record.ID, id) >= 0, nil
	})
	if err != nil {
		return nil, -1, err
	}

	var record *Record
	if i >= 0 && i < r.totalRecords {
		record, err = r.At(ctx, i)
		if err != nil {
			return nil, -1, err
		}
		return record, i, nil
	}
	return nil, -1, nil
}

func (r *indexReader) getPage(ctx context.Context, pageIdx int) (*page, error) {
	page, ok := r.pageCache[pageIdx]
	if ok {
		return page, nil
	}

	pageBuffer := make([]byte, r.pageSizeBytes)
	_, err := r.r.ReadAt(ctx, pageBuffer, int64(pageIdx*r.pageSizeBytes))
	if err != nil {
		return nil, err
	}

	page, err = unmarshalPageFromBytes(pageBuffer, &indexHeader{})
	if err != nil {
		return nil, err
	}

	// checksum
	h := xxhash.New()
	_, _ = h.Write(page.data)
	if page.header.(*indexHeader).checksum != h.Sum64() {
		return nil, fmt.Errorf("mismatched checksum: %d", pageIdx)
	}

	r.pageCache[pageIdx] = page

	return page, nil
}
