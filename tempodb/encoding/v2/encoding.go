package v2

import (
	"context"
	"io/fs"
	"time"

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
	return CopyBlock(ctx, meta, meta, from, to)
}

func (v Encoding) MigrateBlock(ctx context.Context, fromMeta, toMeta *backend.BlockMeta, from backend.Reader, to backend.Writer) error {
	return CopyBlock(ctx, fromMeta, toMeta, from, to)
}

func (v Encoding) CreateBlock(ctx context.Context, cfg *common.BlockConfig, meta *backend.BlockMeta, i common.Iterator, _ backend.Reader, to backend.Writer) (*backend.BlockMeta, error) {
	return CreateBlock(ctx, cfg, meta, i, to)
}

// OpenWALBlock opens an existing appendable block
func (v Encoding) OpenWALBlock(filename string, path string, ingestionSlack time.Duration, additionalStartSlack time.Duration) (common.WALBlock, error, error) {
	return openWALBlock(filename, path, ingestionSlack, additionalStartSlack)
}

// CreateWALBlock creates a new appendable block
func (v Encoding) CreateWALBlock(meta *backend.BlockMeta, filepath, dataEncoding string, ingestionSlack time.Duration) (common.WALBlock, error) {
	// Default data encoding if needed
	if dataEncoding == "" {
		dataEncoding = model.CurrentEncoding
	}
	return createWALBlock(meta, filepath, dataEncoding, ingestionSlack)
}

func (v Encoding) OwnsWALBlock(entry fs.DirEntry) bool {
	return ownsWALBlock(entry)
}
