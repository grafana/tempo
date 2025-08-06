package unsupported

import (
	"context"
	"io/fs"
	"time"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

const VersionString = "Unsupported"

type Encoding struct{}

func (v Encoding) Version() string {
	return VersionString
}

func (v Encoding) NewCompactor(opts common.CompactionOptions) common.Compactor {
	return nil
}

func (v Encoding) OpenBlock(meta *backend.BlockMeta, r backend.Reader) (common.BackendBlock, error) {
	return Block{}, nil
}

func (v Encoding) CopyBlock(ctx context.Context, meta *backend.BlockMeta, from backend.Reader, to backend.Writer) error {
	return common.ErrUnsupported
}

func (v Encoding) MigrateBlock(ctx context.Context, fromMeta, toMeta *backend.BlockMeta, from backend.Reader, to backend.Writer) error {
	return common.ErrUnsupported
}

func (v Encoding) CreateBlock(ctx context.Context, cfg *common.BlockConfig, meta *backend.BlockMeta, i common.Iterator, r backend.Reader, to backend.Writer) (*backend.BlockMeta, error) {
	return nil, common.ErrUnsupported
}

// OpenWALBlock opens an existing appendable block
func (v Encoding) OpenWALBlock(filename, path string, ingestionSlack, additionalStartSlack time.Duration) (common.WALBlock, error, error) {
	return nil, common.ErrUnsupported, nil
}

// CreateWALBlock creates a new appendable block
func (v Encoding) CreateWALBlock(meta *backend.BlockMeta, filepath, dataEncoding string, ingestionSlack time.Duration) (common.WALBlock, error) {
	return nil, common.ErrUnsupported
}

func (v Encoding) OwnsWALBlock(entry fs.DirEntry) bool {
	return false
}
