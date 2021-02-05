package v1

import (
	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
)

// NewPagedIterator returns a v0.backendIterator.  There are no changes
// to logic from the v0 iterator and all compression changes are handled in the pageReader
func NewPagedIterator(chunkSizeBytes uint32, indexReader common.IndexReader, pageReader common.PageReader) common.Iterator {
	return v0.NewPagedIterator(chunkSizeBytes, indexReader, pageReader)
}
