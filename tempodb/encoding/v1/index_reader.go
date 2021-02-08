package v1

import (
	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
)

// NewIndexReader returns an index reader for a byte slice of marshalled
// ordered records.
// The index has not changed between v0 and v1.
func NewIndexReader(index []byte) (common.IndexReader, error) {
	return v0.NewIndexReader(index)
}
