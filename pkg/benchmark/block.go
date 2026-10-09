package benchmark

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/google/uuid"

	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/backend/local"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
	"github.com/grafana/tempo/v3/tempodb/encoding/vparquet4"
	"github.com/grafana/tempo/v3/tempodb/encoding/vparquet5"
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

// warmBlock streams the block's objects once, pulling them into the OS page
// cache. The block is read-only and shared by every case, so its cold-read cost
// is paid here, once before the run, instead of landing on whichever case
// first touches each object. It reads through the raw reader: the counted and
// simulated readers charge every request, and warming is not one to bill.
func warmBlock(ctx context.Context, raw backend.RawReader, meta *backend.BlockMeta) error {
	keyPath := backend.KeyPathForBlock(uuid.UUID(meta.BlockID), meta.TenantID)

	// The data object's name belongs to the encoding that wrote the block, so
	// a block no encoding claims cannot be warmed, and would fail to open.
	var dataName string
	switch meta.Version {
	case vparquet4.VersionString:
		dataName = vparquet4.DataFileName
	case vparquet5.VersionString:
		dataName = vparquet5.DataFileName
	default:
		return fmt.Errorf("block %s is %s, which no supported encoding wrote", meta.BlockID, meta.Version)
	}

	names := make([]string, 0, int(meta.BloomShardCount)+2)
	names = append(names, dataName, common.NameIndex)
	for shard := range int(meta.BloomShardCount) {
		names = append(names, common.BloomName(shard))
	}

	for _, name := range names {
		if err := readThrough(ctx, raw, name, keyPath); err != nil {
			return fmt.Errorf("warming %s of block %s: %w", name, meta.BlockID, err)
		}
	}
	return nil
}

// readThrough streams one object and keeps none of it, so warming costs
// constant memory whatever the block's size.
func readThrough(ctx context.Context, raw backend.RawReader, name string, keyPath backend.KeyPath) error {
	rc, _, err := raw.Read(ctx, name, keyPath, nil)
	if err != nil {
		return err
	}
	defer rc.Close()
	_, err = io.Copy(io.Discard, rc)
	return err
}
