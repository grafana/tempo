package v2

import (
	"fmt"

	"github.com/cespare/xxhash/v2"
)

type indexWriter struct {
	pageSizeBytes int
	recordRW      RecordReaderWriter
}

// NewIndexWriter returns an index writer that writes to the provided io.Writer.
// The index has not changed between v0 and v1.
func NewIndexWriter(pageSizeBytes int) IndexWriter {
	return &indexWriter{
		pageSizeBytes: pageSizeBytes,
		recordRW:      NewRecordReaderWriter(),
	}
}

// Write implements common.IndexWriter
func (w *indexWriter) Write(records []Record) ([]byte, error) {
	// we need to write a page at a time to an output byte slice
	//  first let's calculate how many pages we need
	recordsPerPage := objectsPerPage(w.recordRW.RecordLength(), w.pageSizeBytes, IndexHeaderLength)
	totalPages := totalPages(len(records), recordsPerPage)

	if recordsPerPage == 0 {
		return nil, fmt.Errorf("pageSize %d too small for one record", w.pageSizeBytes)
	}

	totalBytes := totalPages * w.pageSizeBytes
	indexBuffer := make([]byte, totalBytes)

	for currentPage := 0; currentPage < totalPages; currentPage++ {
		var pageRecords []Record

		if len(records) > recordsPerPage {
			pageRecords = records[:recordsPerPage]
			records = records[recordsPerPage:]
		} else {
			pageRecords = records[:]
			records = []Record{}
		}

		if len(pageRecords) == 0 {
			return nil, fmt.Errorf("unexpected 0 length records %d,%d,%d,%d", currentPage, recordsPerPage, w.pageSizeBytes, totalPages)
		}

		// header
		header := &indexHeader{}

		// page
		pageBuffer := indexBuffer[currentPage*w.pageSizeBytes : (currentPage+1)*w.pageSizeBytes]

		// write records and calculate crc
		pageData := pageBuffer[header.headerLength()+int(baseHeaderSize):]
		err := w.recordRW.MarshalRecordsToBuffer(pageRecords, pageData)
		if err != nil {
			return nil, err
		}

		h := xxhash.New()
		_, _ = h.Write(pageData)
		header.checksum = h.Sum64()

		_, err = marshalHeaderToPage(pageBuffer, header)
		if err != nil {
			return nil, err
		}
	}

	return indexBuffer, nil
}
