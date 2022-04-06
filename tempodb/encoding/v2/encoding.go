package v2

import (
	"context"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

const VersionString = "v2"

// v2Encoding
type Encoding struct{}

func (v Encoding) Version() string {
	return VersionString
}

func (v Encoding) NewCompactor() common.Compactor {
	return NewCompactor()
}

func (v Encoding) OpenBackendBlock(meta *backend.BlockMeta, r backend.Reader) (common.BackendBlock, error) {
	return NewBackendBlock(meta, r)
}

func (v Encoding) CopyBlock(ctx context.Context, meta *backend.BlockMeta, from backend.Reader, to backend.Writer) error {
	return CopyBlock(ctx, meta, from, to)
}
