package v0

import (
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type indexWriter struct {
	r common.RecordReaderWriter
}

// NewIndexWriter returns an index writer
func NewIndexWriter() common.IndexWriter {
	return &indexWriter{
		r: NewRecordReaderWriter(),
	}
}

// Write implements common.IndexWriter
func (w *indexWriter) Write(records []common.Record) ([]byte, error) {
	return w.r.MarshalRecords(records)
}
