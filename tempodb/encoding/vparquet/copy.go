package vparquet

import (
	"context"
	"fmt"

	"github.com/grafana/tempo/pkg/cache"
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

	cacheInfo := &backend.CacheInfo{
		Role: cache.RoleBloom,
	}
	// Read entire object and attempt to cache
	cpyBloom := func(name string) error {
		cacheInfo.Meta = fromMeta
		b, err := from.Read(ctx, name, fromMeta.BlockID, fromMeta.TenantID, cacheInfo)
		if err != nil {
			return fmt.Errorf("error reading %s: %w", name, err)
		}

		cacheInfo.Meta = toMeta
		return to.Write(ctx, name, toMeta.BlockID, toMeta.TenantID, b, cacheInfo)
	}

	// Data
	err := copyStream(DataFileName)
	if err != nil {
		return err
	}

	// Bloom
	for i := 0; i < common.ValidateShardCount(int(fromMeta.BloomShardCount)); i++ {
		err = cpyBloom(common.BloomName(i))
		if err != nil {
			return err
		}
	}

	// Meta
	err = to.WriteBlockMeta(ctx, toMeta)
	return err
}

func writeBlockMeta(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, bloom *common.ShardedBloomFilter) error {
	// bloom
	blooms, err := bloom.Marshal()
	if err != nil {
		return err
	}
	cacheInfo := &backend.CacheInfo{
		Role: cache.RoleBloom,
		Meta: meta,
	}
	for i, bloom := range blooms {
		nameBloom := common.BloomName(i)
		err := w.Write(ctx, nameBloom, meta.BlockID, meta.TenantID, bloom, cacheInfo)
		if err != nil {
			return fmt.Errorf("unexpected error writing bloom-%d: %w", i, err)
		}
	}

	// meta
	err = w.WriteBlockMeta(ctx, meta)
	if err != nil {
		return fmt.Errorf("unexpected error writing meta: %w", err)
	}

	return nil
}
