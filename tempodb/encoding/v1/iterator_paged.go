package v1

import (
	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
)

// jpe comment
// NewPagedIterator returns a backendIterator.  This iterator is used to iterate
//  through objects stored in object storage.
func NewPagedIterator(chunkSizeBytes uint32, indexReader common.IndexReader, pageReader common.PageReader) common.Iterator {
	return v0.NewPagedIterator(chunkSizeBytes, indexReader, pageReader)
}
