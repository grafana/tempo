package vparquet4

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/cache"
	"github.com/grafana/tempo/tempodb/backend"
	backend_v1 "github.com/grafana/tempo/tempodb/backend/v1"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

func CopyBlock(ctx context.Context, fromMeta, toMeta *backend_v1.BlockMeta, from backend.Reader, to backend.Writer) error {
	fromMetaBlockID, err := uuid.Parse(fromMeta.BlockId)
	if err != nil {
		return err
	}

	toMetaBlockID, err := uuid.Parse(toMeta.BlockId)
	if err != nil {
		return err
	}

	// Copy streams, efficient but can't cache.
	copyStream := func(name string) error {
		reader, size, err := from.StreamReader(ctx, name, fromMetaBlockID, fromMeta.TenantId)
		if err != nil {
			return fmt.Errorf("error reading %s: %w", name, err)
		}
		defer reader.Close()

		return to.StreamWriter(ctx, name, toMetaBlockID, toMeta.TenantId, reader, size)
	}

	// Read entire object and attempt to cache
	cpy := func(name string, cacheInfo *backend.CacheInfo) error {
		cacheInfo.Meta = fromMeta
		b, err := from.Read(ctx, name, fromMetaBlockID, fromMeta.TenantId, cacheInfo)
		if err != nil {
			return fmt.Errorf("error reading %s: %w", name, err)
		}

		cacheInfo.Meta = toMeta
		return to.Write(ctx, name, toMetaBlockID, toMeta.TenantId, b, cacheInfo)
	}

	// Data
	err = copyStream(DataFileName)
	if err != nil {
		return err
	}

	// Bloom
	cacheInfo := &backend.CacheInfo{Role: cache.RoleBloom}
	for i := 0; i < common.ValidateShardCount(int(fromMeta.BloomShardCount)); i++ {
		err = cpy(common.BloomName(i), cacheInfo)
		if err != nil {
			return err
		}
	}

	// Index (may not exist)
	err = cpy(common.NameIndex, &backend.CacheInfo{Role: cache.RoleTraceIDIdx})
	if err != nil && !errors.Is(err, backend.ErrDoesNotExist) {
		return err
	}

	// Meta
	err = to.WriteBlockMeta(ctx, toMeta)
	return err
}

func writeBlockMeta(ctx context.Context, w backend.Writer, meta *backend_v1.BlockMeta, bloom *common.ShardedBloomFilter, index *index) error {
	// bloom
	blooms, err := bloom.Marshal()
	if err != nil {
		return err
	}

	cacheInfo := &backend.CacheInfo{
		Meta: meta,
		Role: cache.RoleBloom,
	}

	metaBlockID, err := uuid.Parse(meta.BlockId)
	if err != nil {
		return err
	}

	for i, bloom := range blooms {
		nameBloom := common.BloomName(i)
		err := w.Write(ctx, nameBloom, metaBlockID, meta.TenantId, bloom, cacheInfo)
		if err != nil {
			return fmt.Errorf("unexpected error writing bloom-%d: %w", i, err)
		}
	}

	// Index
	i, err := index.Marshal()
	if err != nil {
		return err
	}
	err = w.Write(ctx, common.NameIndex, metaBlockID, meta.TenantId, i, &backend.CacheInfo{
		Meta: meta,
		Role: cache.RoleTraceIDIdx,
	})
	if err != nil {
		return err
	}

	// meta
	err = w.WriteBlockMeta(ctx, meta)
	if err != nil {
		return fmt.Errorf("unexpected error writing meta: %w", err)
	}

	return nil
}
