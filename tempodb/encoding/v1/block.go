package v1

import (
	"context"
	"io"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
)

// These methods control the layout of the block in the backend.  Nothing has changed between v0 and v1 here
// so we will just passthrough

// NameObjects returns v0 name
func NameObjects() string {
	return v0.NameObjects
}

// NameIndex returns v0 name
func NameIndex() string {
	return v0.NameIndex
}

// BloomName returns v0 name
func BloomName(shard int) string {
	return v0.BloomName(shard)
}

// WriteBlockMeta writes the bloom filter, meta and index to the passed in backend.Writer
func WriteBlockMeta(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, records []*common.Record, b *common.ShardedBloomFilter) error {
	return v0.WriteBlockMeta(ctx, w, meta, records, b)
}

// WriteBlockData writes the data object from an io.Reader to the backend.Writer
func WriteBlockData(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, r io.Reader, size int64) error {
	return v0.WriteBlockData(ctx, w, meta, r, size)
}

// AppendBlockData appends the bytes passed to the block data
func AppendBlockData(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, tracker backend.AppendTracker, buffer []byte) (backend.AppendTracker, error) {
	return v0.AppendBlockData(ctx, w, meta, tracker, buffer)
}
