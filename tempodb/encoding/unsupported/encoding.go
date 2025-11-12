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

func (v Encoding) NewCompactor(_ common.CompactionOptions) common.Compactor {
	return nil
}

func (v Encoding) CompactionSupported() bool {
	return false
}

func (v Encoding) OpenBlock(meta *backend.BlockMeta, _ backend.Reader) (common.BackendBlock, error) {
	return Block{meta: meta}, nil
}

func (v Encoding) CopyBlock(context.Context, *backend.BlockMeta, backend.Reader, backend.Writer) error {
	return util.ErrUnsupported
}

func (v Encoding) MigrateBlock(context.Context, *backend.BlockMeta, *backend.BlockMeta, backend.Reader, backend.Writer) error {
	return util.ErrUnsupported
}

func (v Encoding) CreateBlock(context.Context, *common.BlockConfig, *backend.BlockMeta, common.Iterator, backend.Reader, backend.Writer) (*backend.BlockMeta, error) {
	return nil, util.ErrUnsupported
}

// OpenWALBlock opens an existing appendable block
func (v Encoding) OpenWALBlock(string, string, time.Duration, time.Duration) (b common.WALBlock, warning error, err error) {
	return nil, nil, util.ErrUnsupported
}

// CreateWALBlock creates a new appendable block
func (v Encoding) CreateWALBlock(*backend.BlockMeta, string, string, time.Duration) (common.WALBlock, error) {
	return nil, util.ErrUnsupported
}

func (v Encoding) OwnsWALBlock(entry fs.DirEntry) bool {
	return ownsWALBlock(entry)
}
