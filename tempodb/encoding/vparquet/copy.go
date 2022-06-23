package vparquet

import (
	"context"
	"fmt"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/pkg/errors"
)

func CopyBlock(ctx context.Context, meta *backend.BlockMeta, from backend.Reader, to backend.Writer) error {
	blockID := meta.BlockID
	tenantID := meta.TenantID

	// Copy streams, efficient but can't cache.
	copyStream := func(name string) error {
		reader, size, err := from.StreamReader(ctx, name, blockID, tenantID)
		if err != nil {
			return errors.Wrapf(err, "error reading %s", name)
		}
		defer reader.Close()

		return to.StreamWriter(ctx, name, blockID, tenantID, reader, size)
	}

	// Read entire object and attempt to cache
	copy := func(name string) error {
		b, err := from.Read(ctx, name, blockID, tenantID, true)
		if err != nil {
			return errors.Wrapf(err, "error reading %s", name)
		}

		return to.Write(ctx, name, blockID, tenantID, b, true)
	}

	// Data
	err := copyStream(DataFileName)
	if err != nil {
		return err
	}

	// Bloom
	for i := 0; i < common.ValidateShardCount(int(meta.BloomShardCount)); i++ {
		err = copy(common.BloomName(i))
		if err != nil {
			return err
		}
	}

	// Meta
	err = to.WriteBlockMeta(ctx, meta)
	return err
}

func writeBlockMeta(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, bloom *common.ShardedBloomFilter) error {

	// bloom
	blooms, err := bloom.Marshal()
	if err != nil {
		return err
	}
	for i, bloom := range blooms {
		nameBloom := common.BloomName(i)
		err := w.Write(ctx, nameBloom, meta.BlockID, meta.TenantID, bloom, true)
		if err != nil {
			return fmt.Errorf("unexpected error writing bloom-%d %w", i, err)
		}
	}

	// meta
	err = w.WriteBlockMeta(ctx, meta)
	if err != nil {
		return fmt.Errorf("unexpected error writing meta %w", err)
	}

	return nil
}
