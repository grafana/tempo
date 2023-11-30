package v2

import (
	"context"
	"fmt"

	"github.com/grafana/tempo/pkg/cache"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// writeBlockMeta writes the bloom filter, meta and index to the passed in backend.Writer
func writeBlockMeta(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, indexBytes []byte, b *common.ShardedBloomFilter) error {
	blooms, err := b.Marshal()
	if err != nil {
		return err
	}

	// index
	err = w.Write(ctx, common.NameIndex, meta.BlockID, meta.TenantID, indexBytes, nil)
	if err != nil {
		return fmt.Errorf("unexpected error writing index: %w", err)
	}

	cacheInfo := &backend.CacheInfo{
		Meta: meta,
		Role: cache.RoleBloom,
	}
	// bloom
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

// appendBlockData appends the bytes passed to the block data
func appendBlockData(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, tracker backend.AppendTracker, buffer []byte) (backend.AppendTracker, error) {
	return w.Append(ctx, common.NameObjects, meta.BlockID, meta.TenantID, tracker, buffer)
}

// CopyBlock copies a block from one backend to another.   It is done at a low level, all encoding/formatting is preserved.
func CopyBlock(ctx context.Context, srcMeta, destMeta *backend.BlockMeta, src backend.Reader, dest backend.Writer) error {
	// Copy streams, efficient but can't cache.
	copyStream := func(name string) error {
		reader, size, err := src.StreamReader(ctx, name, srcMeta.BlockID, srcMeta.TenantID)
		if err != nil {
			return fmt.Errorf("error reading %s: %w", name, err)
		}
		defer reader.Close()

		return dest.StreamWriter(ctx, name, destMeta.BlockID, destMeta.TenantID, reader, size)
	}

	cacheInfo := &backend.CacheInfo{
		Role: cache.RoleBloom,
	}

	// Read entire object and attempt to cache
	cpyBloom := func(name string) error {
		cacheInfo.Meta = srcMeta
		b, err := src.Read(ctx, name, srcMeta.BlockID, srcMeta.TenantID, cacheInfo)
		if err != nil {
			return fmt.Errorf("error reading %s: %w", name, err)
		}

		cacheInfo.Meta = destMeta
		return dest.Write(ctx, name, destMeta.BlockID, destMeta.TenantID, b, cacheInfo)
	}

	// Data
	err := copyStream(common.NameObjects)
	if err != nil {
		return err
	}

	// Bloom
	for i := 0; i < common.ValidateShardCount(int(srcMeta.BloomShardCount)); i++ {
		err = cpyBloom(common.BloomName(i))
		if err != nil {
			return err
		}
	}

	// Index
	err = copyStream(common.NameIndex)
	if err != nil {
		return err
	}

	// Meta
	err = dest.WriteBlockMeta(ctx, destMeta)
	return err
}
