package benchmark

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/google/uuid"

	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/backend/local"
)

// LoadLocalBlock opens the block at path, which must be a directory laid out as
// <bucket>/<tenant-id>/<block-id> because that is how backend.KeyPathForBlock
// addresses a block.
func LoadLocalBlock(ctx context.Context, path string) (*backend.BlockMeta, backend.Reader, error) {
	meta, raw, err := openLocalBlock(ctx, path)
	if err != nil {
		return nil, nil, err
	}
	return meta, backend.NewReader(raw), nil
}

// openLocalBlock returns the raw reader too, for callers that need to wrap it.
func openLocalBlock(ctx context.Context, path string) (*backend.BlockMeta, backend.RawReader, error) {
	path = filepath.Clean(path)

	blockDir, tenantID := filepath.Base(path), filepath.Base(filepath.Dir(path))
	bucket := filepath.Dir(filepath.Dir(path))

	blockID, err := uuid.Parse(blockDir)
	if err != nil {
		return nil, nil, fmt.Errorf("%q is not a block directory: its name must be a block ID: %w", path, err)
	}
	if tenantID == "." || tenantID == string(filepath.Separator) {
		return nil, nil, fmt.Errorf("%q has no tenant directory above the block", path)
	}

	rawR, _, _, err := local.New(&local.Config{Path: bucket})
	if err != nil {
		return nil, nil, err
	}

	meta, err := backend.NewReader(rawR).BlockMeta(ctx, blockID, tenantID)
	if err != nil {
		return nil, nil, fmt.Errorf("reading block meta for %s in tenant %s: %w", blockID, tenantID, err)
	}
	return meta, rawR, nil
}
