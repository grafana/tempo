package v2

import (
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v1 "github.com/grafana/tempo/tempodb/encoding/v1"
)

// NewPageReader constructs a v1 PageReader that handles compression
func NewPageReader(r backend.ContextReader, encoding backend.Encoding) (common.PageReader, error) {
	return v1.NewPageReader(r, encoding)
}
