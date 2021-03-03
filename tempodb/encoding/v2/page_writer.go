package v2

import (
	"io"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v1 "github.com/grafana/tempo/tempodb/encoding/v1"
)

// NewPageWriter creates a v0 page writer.  This page writer
// writes raw bytes only
func NewPageWriter(writer io.Writer, encoding backend.Encoding) (common.PageWriter, error) {
	return v1.NewPageWriter(writer, encoding)
}
