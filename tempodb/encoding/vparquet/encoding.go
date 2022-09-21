package vparquet

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

const VersionString = "vParquet"

type Encoding struct{}

func (v Encoding) Version() string {
	return VersionString
}

func (v Encoding) NewCompactor(opts common.CompactionOptions) common.Compactor {
	return NewCompactor(opts)
}

func (v Encoding) OpenBlock(meta *backend.BlockMeta, r backend.Reader) (common.BackendBlock, error) {
	return newBackendBlock(meta, r), nil
}

func (v Encoding) CopyBlock(ctx context.Context, meta *backend.BlockMeta, from backend.Reader, to backend.Writer) error {
	return CopyBlock(ctx, meta, from, to)
}

func (v Encoding) CreateBlock(ctx context.Context, cfg *common.BlockConfig, meta *backend.BlockMeta, i common.Iterator, dec model.ObjectDecoder, r backend.Reader, to backend.Writer) (*backend.BlockMeta, error) {
	return CreateBlock(ctx, cfg, meta, i, dec, r, to)
}

// OpenAppendBlock opens an existing appendable block
func (v Encoding) OpenAppendBlock(filename string, path string, ingestionSlack time.Duration, additionalStartSlack time.Duration, fn common.RangeFunc) (common.AppendBlock, error, error) {
	panic("unsupported")
}

// CreateAppendBlock creates a new appendable block
func (v Encoding) CreateAppendBlock(id uuid.UUID, tenantID string, filepath string, e backend.Encoding, dataEncoding string, ingestionSlack time.Duration) (common.AppendBlock, error) {
	panic("unsupported")
}
