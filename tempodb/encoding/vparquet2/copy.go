package vparquet2

import (
	"context"
	"errors"
	"fmt"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

func CopyBlock(ctx context.Context, fromMeta, toMeta *backend.BlockMeta, from backend.Reader, to backend.Writer) error {
	// Copy streams, efficient but can't cache.
	copyStream := func(name string) error {
		reader, size, err := from.StreamReader(ctx, name, fromMeta.BlockID, fromMeta.TenantID)
		if err != nil {
			return fmt.Errorf("error reading %s: %w", name, err)
		}
		defer reader.Close()

		return to.StreamWriter(ctx, name, toMeta.BlockID, toMeta.TenantID, reader, size)
	}

	// Read entire object and attempt to cache
	cpy := func(name string) error {
		b, err := from.Read(ctx, name, fromMeta.BlockID, fromMeta.TenantID, true)
		if err != nil {
			return fmt.Errorf("error reading %s: %w", name, err)
		}

		return to.Write(ctx, name, toMeta.BlockID, toMeta.TenantID, b, true)
	}

	// Data
	err := copyStream(DataFileName)
	if err != nil {
		return err
	}

	// Bloom
	for i := 0; i < common.ValidateShardCount(int(fromMeta.BloomShardCount)); i++ {
		err = cpy(common.BloomName(i))
		if err != nil {
			return err
		}
	}

	// Index (may not exist)
	err = cpy(common.NameIndex)
	if err != nil && !errors.Is(err, backend.ErrDoesNotExist) {
		return err
	}

	// Meta
	err = to.WriteBlockMeta(ctx, toMeta)
	return err
}

func writeBlockMeta(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, bloom *common.ShardedBloomFilter, index *index) error {
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

	// Index
	i, err := index.Marshal()
	if err != nil {
		return err
	}
	err = w.Write(ctx, common.NameIndex, meta.BlockID, meta.TenantID, i, true)
	if err != nil {
		return err
	}

	// meta
	err = w.WriteBlockMeta(ctx, meta)
	if err != nil {
		return fmt.Errorf("unexpected error writing meta %w", err)
	}

	return nil
}
