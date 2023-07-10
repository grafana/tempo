package vparquet3

import (
	"context"
	"io/fs"
	"time"

	"github.com/google/uuid"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

const VersionString = "vParquet3"

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
	return CopyBlock(ctx, meta, meta, from, to)
}

func (v Encoding) MigrateBlock(ctx context.Context, fromMeta, toMeta *backend.BlockMeta, from backend.Reader, to backend.Writer) error {
	return CopyBlock(ctx, fromMeta, toMeta, from, to)
}

func (v Encoding) CreateBlock(ctx context.Context, cfg *common.BlockConfig, meta *backend.BlockMeta, i common.Iterator, r backend.Reader, to backend.Writer) (*backend.BlockMeta, error) {
	return CreateBlock(ctx, cfg, meta, i, r, to)
}

// OpenWALBlock opens an existing appendable block
func (v Encoding) OpenWALBlock(filename, path string, ingestionSlack, additionalStartSlack time.Duration) (common.WALBlock, error, error) {
	return openWALBlock(filename, path, ingestionSlack, additionalStartSlack)
}

// CreateWALBlock creates a new appendable block
func (v Encoding) CreateWALBlock(id uuid.UUID, tenantID, filepath string, e backend.Encoding, dataEncoding string, ingestionSlack time.Duration, dedicatedColumns backend.DedicatedColumns) (common.WALBlock, error) {
	return createWALBlock(id, tenantID, filepath, e, dataEncoding, ingestionSlack, dedicatedColumns)
}

func (v Encoding) OwnsWALBlock(entry fs.DirEntry) bool {
	return ownsWALBlock(entry)
}
