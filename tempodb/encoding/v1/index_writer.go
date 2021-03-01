package v1

import (
	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
)

// NewIndexWriter returns an index writer that writes to the provided io.Writer.
// The index has not changed between v0 and v1.
func NewIndexWriter() common.IndexWriter {
	return v0.NewIndexWriter()
}
