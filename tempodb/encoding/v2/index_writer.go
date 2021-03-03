package v2

import (
	"github.com/grafana/tempo/tempodb/encoding/common"
	v1 "github.com/grafana/tempo/tempodb/encoding/v1"
)

// NewIndexWriter returns an index writer that writes to the provided io.Writer.
// The index has not changed between v0 and v1.
func NewIndexWriter() common.IndexWriter {
	return v1.NewIndexWriter()
}
