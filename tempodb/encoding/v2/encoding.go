package v2

import (
	"context"

	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

const VersionString = "v2"

// v2Encoding
type Encoding struct{}

func (v Encoding) Version() string {
	return VersionString
}

func (v Encoding) NewCompactor(opts common.CompactionOptions) common.Compactor {
	return NewCompactor(opts)
}

func (v Encoding) OpenBlock(meta *backend.BlockMeta, r backend.Reader) (common.BackendBlock, error) {
	return NewBackendBlock(meta, r)
}

func (v Encoding) CopyBlock(ctx context.Context, meta *backend.BlockMeta, from backend.Reader, to backend.Writer) error {
	return CopyBlock(ctx, meta, from, to)
}

func (v Encoding) CreateBlock(ctx context.Context, cfg *common.BlockConfig, meta *backend.BlockMeta, i common.Iterator, dec model.ObjectDecoder, to backend.Writer) (*backend.BlockMeta, error) {
	return CreateBlock(ctx, cfg, meta, i, dec, to)
}
