package v2

import (
	"bytes"
	"context"
	"fmt"
	"sort"

	"github.com/cespare/xxhash"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/base"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/opentracing/opentracing-go"
)

type indexReader struct {
	r backend.ContextReader

	pageSizeBytes int
	totalRecords  int

	pageCache map[int]*page // indexReader is not concurrency safe, but since it is currently used within one request it is fine.
}

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

// At implements common.indexReader
func (r *indexReader) At(ctx context.Context, i int) (*common.Record, error) {
	if i < 0 || i >= r.totalRecords {
		return nil, nil
	}

	recordsPerPage := objectsPerPage(base.RecordLength, r.pageSizeBytes, IndexHeaderLength)
	if recordsPerPage == 0 {
		return nil, fmt.Errorf("page %d is too small for one record", r.pageSizeBytes)
	}
	pageIdx := i / recordsPerPage
	recordIdx := i % recordsPerPage

	page, err := r.getPage(ctx, pageIdx)
	if err != nil {
		return nil, err
	}

	if recordIdx >= len(page.data)/base.RecordLength {
		return nil, fmt.Errorf("unexpected out of bounds index %d, %d, %d, %d", i, pageIdx, recordIdx, len(page.data))
	}

	recordBytes := page.data[recordIdx*base.RecordLength : (recordIdx+1)*base.RecordLength]

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

	return base.UnmarshalRecord(recordBytes), nil
}

// Find implements common.indexReader
func (r *indexReader) Find(ctx context.Context, id common.ID) (*common.Record, int, error) {
	// with a linear distribution of trace ids we can actually do much better than a normal
	// binary search.  unfortunately there are edge cases which make this perform far worse.
	// for instance consider a set of trace ids what with 90% 64 bit ids and 10% 128 bit ids.
	span, ctx := opentracing.StartSpanFromContext(ctx, "indexReader.Find")
	defer span.Finish()

	var err error
	i := sort.Search(r.totalRecords, func(i int) bool {
		if err != nil { // if we get an error somewhere then just force bail the search
			return true
		}

		var record *common.Record
		record, err = r.At(ctx, i)
		if err != nil {
			return true
		}

		return bytes.Compare(record.ID, id) >= 0
	})

	if err != nil {
		return nil, -1, err
	}

	var record *common.Record
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
