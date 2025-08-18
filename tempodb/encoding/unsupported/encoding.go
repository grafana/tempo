package unsupported

import (
	"context"
	"io/fs"
	"time"

	"github.com/grafana/tempo/pkg/util"
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

func (v Encoding) CompactionSupported() bool {
	return false
}

func (v Encoding) OpenBlock(meta *backend.BlockMeta, r backend.Reader) (common.BackendBlock, error) {
	return Block{meta: meta}, nil
}

func (v Encoding) CopyBlock(ctx context.Context, meta *backend.BlockMeta, from backend.Reader, to backend.Writer) error {
	return util.ErrUnsupported
}

func (v Encoding) MigrateBlock(ctx context.Context, fromMeta, toMeta *backend.BlockMeta, from backend.Reader, to backend.Writer) error {
	return util.ErrUnsupported
}

func (v Encoding) CreateBlock(ctx context.Context, cfg *common.BlockConfig, meta *backend.BlockMeta, i common.Iterator, r backend.Reader, to backend.Writer) (*backend.BlockMeta, error) {
	return nil, util.ErrUnsupported
}

// OpenWALBlock opens an existing appendable block
func (v Encoding) OpenWALBlock(filename, path string, ingestionSlack, additionalStartSlack time.Duration) (b common.WALBlock, warning error, err error) {
	return nil, nil, util.ErrUnsupported
}

// CreateWALBlock creates a new appendable block
func (v Encoding) CreateWALBlock(meta *backend.BlockMeta, filepath, dataEncoding string, ingestionSlack time.Duration) (common.WALBlock, error) {
	return nil, util.ErrUnsupported
}

func (v Encoding) OwnsWALBlock(entry fs.DirEntry) bool {
	return ownsWALBlock(entry)
}
