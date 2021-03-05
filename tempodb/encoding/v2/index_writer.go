package v2

import (
	"fmt"

	"github.com/grafana/tempo/tempodb/encoding/base"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type indexWriter struct {
	pageSizeBytes  int
	recordsPerPage int
}

// NewIndexWriter returns an index writer that writes to the provided io.Writer.
// The index has not changed between v0 and v1.
func NewIndexWriter(pageSizeBytes int) common.IndexWriter {
	return &indexWriter{
		pageSizeBytes: pageSizeBytes,
	}
}

// Write implements common.IndexWriter
func (w *indexWriter) Write(records []*common.Record) ([]byte, error) {
	// we need to write a page at a time to an output byte slice
	//  first let's calculate how many pages we need
	recordsPerPage := objectsPerPage(base.RecordLength, w.pageSizeBytes)
	totalPages := totalPages(len(records), recordsPerPage)

	if recordsPerPage == 0 {
		return nil, fmt.Errorf("pageSize %d too small for one record", w.pageSizeBytes)
	}

	totalBytes := totalPages * w.pageSizeBytes
	indexBuffer := make([]byte, totalBytes)

	for currentPage := 0; currentPage < totalPages; currentPage++ {
		var pageRecords []*common.Record

		if len(records) > recordsPerPage {
			pageRecords = records[:recordsPerPage]
			records = records[recordsPerPage:]
		} else {
			pageRecords = records[:]
			records = []*common.Record{}
		}

		if len(pageRecords) == 0 {
			return nil, fmt.Errorf("unexpected 0 length records %d,%d,%d,%d", currentPage, recordsPerPage, w.pageSizeBytes, totalPages)
		}

		writePageData := func(b []byte) error { // jpe does this need to exist?  change to marshalHeaderOnly and return the rest of the page?
			return base.MarshalRecordsToBuffer(pageRecords, b)
		}

		pageBuffer := indexBuffer[currentPage*w.pageSizeBytes : (currentPage+1)*w.pageSizeBytes]
		_, err := marshalPageToBuffer(writePageData, pageBuffer)
		if err != nil {
			return nil, err
		}
	}

	return indexBuffer, nil
}
