package encoding

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
)

const (
	// nameObjects names the backend data object
	nameObjects = "data"
	// nameIndex names the backend index object
	nameIndex = "index"
	// nameBloomPrefix is the prefix used to build the bloom shards
	nameBloomPrefix = "bloom-"
)

// bloomName returns the backend bloom name for the given shard
func bloomName(shard int) string {
	return nameBloomPrefix + strconv.Itoa(shard)
}

// writeBlockMeta writes the bloom filter, meta and index to the passed in backend.Writer
func writeBlockMeta(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, records []*common.Record, b *common.ShardedBloomFilter) error {
	index, err := v0.MarshalRecords(records)
	if err != nil {
		return err
	}

	blooms, err := b.WriteTo()
	if err != nil {
		return err
	}

	// index
	err = w.Write(ctx, nameIndex, meta.BlockID, meta.TenantID, index)
	if err != nil {
		return fmt.Errorf("unexpected error writing index %w", err)
	}

	// bloom
	for i, bloom := range blooms {
		err := w.Write(ctx, bloomName(i), meta.BlockID, meta.TenantID, bloom)
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

// writeBlockData writes the data object from an io.Reader to the backend.Writer
func writeBlockData(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, r io.Reader, size int64) error {
	return w.WriteReader(ctx, nameObjects, meta.BlockID, meta.TenantID, r, size)
}

// appendBlockData appends the bytes passed to the block data
func appendBlockData(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, tracker backend.AppendTracker, buffer []byte) (backend.AppendTracker, error) {
	return w.Append(ctx, nameObjects, meta.BlockID, meta.TenantID, tracker, buffer)
}
