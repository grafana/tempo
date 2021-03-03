package v0

import (
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type indexWriter struct {
}

// NewIndexWriter returns an index writer
func NewIndexWriter() common.IndexWriter {
	return &indexWriter{}
}

// Write implements common.IndexWriter
func (w *indexWriter) Write(records []*common.Record) ([]byte, error) {
	return MarshalRecords(records)
}
