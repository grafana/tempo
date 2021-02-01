package v0

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

const (
	// NameObjects names the backend data object
	NameObjects = "data"
	// NameIndex names the backend index object
	NameIndex       = "index"
	nameBloomPrefix = "bloom-"
)

// BloomName returns the backend bloom name for the given shard
func BloomName(shard int) string {
	return nameBloomPrefix + strconv.Itoa(shard)
}

// WriteBlockMeta writes the bloom filter, meta and index to the passed in backend.Writer
func WriteBlockMeta(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, records []*common.Record, b *common.ShardedBloomFilter) error {
	index, err := marshalRecords(records)
	if err != nil {
		return err
	}

	blooms, err := b.WriteTo()
	if err != nil {
		return err
	}

	// index
	err = w.Write(ctx, NameIndex, meta.BlockID, meta.TenantID, index)
	if err != nil {
		return fmt.Errorf("unexpected error writing index %w", err)
	}

	// bloom
	for i, bloom := range blooms {
		err := w.Write(ctx, BloomName(i), meta.BlockID, meta.TenantID, bloom)
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

// WriteBlockData writes the data object from an io.Reader to the backend.Writer
func WriteBlockData(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, r io.Reader, size int64) error {
	return w.WriteReader(ctx, NameObjects, meta.BlockID, meta.TenantID, r, size)
}

// AppendBlockData appends the bytes passed to the block data
func AppendBlockData(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, tracker backend.AppendTracker, buffer []byte) (backend.AppendTracker, error) {
	return w.Append(ctx, NameObjects, meta.BlockID, meta.TenantID, tracker, buffer)
}
