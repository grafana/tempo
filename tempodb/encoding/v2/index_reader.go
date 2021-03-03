package v2

import (
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v1 "github.com/grafana/tempo/tempodb/encoding/v1"
)

// NewIndexReader returns an index reader for a byte slice of marshalled
// ordered records.
// The index has not changed between v0 and v1.
func NewIndexReader(r backend.ContextReader) (common.IndexReader, error) {
	return v1.NewIndexReader(r)
}
