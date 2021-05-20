package encoding

import (
	"context"
	"fmt"
	"strconv"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/pkg/errors"
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
func writeBlockMeta(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, indexBytes []byte, b *common.ShardedBloomFilter) error {
	blooms, err := b.Marshal()
	if err != nil {
		return err
	}

	// index
	err = w.Write(ctx, nameIndex, meta.BlockID, meta.TenantID, indexBytes)
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

// appendBlockData appends the bytes passed to the block data
func appendBlockData(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, tracker backend.AppendTracker, buffer []byte) (backend.AppendTracker, error) {
	return w.Append(ctx, nameObjects, meta.BlockID, meta.TenantID, tracker, buffer)
}

// CopyBlock copies a block from one backend to another.   It is done at a low level, all encoding/formatting is preserved.
func CopyBlock(ctx context.Context, meta *backend.BlockMeta, src backend.Reader, dest backend.Writer) error {
	blockID := meta.BlockID
	tenantID := meta.TenantID

	copy := func(name string) error {
		reader, size, err := src.ReadReader(ctx, name, blockID, tenantID)
		if err != nil {
			return errors.Wrapf(err, "error reading %s", name)
		}
		defer reader.Close()

		return dest.WriteReader(ctx, name, blockID, tenantID, reader, size)
	}

	// Data
	err := copy(nameObjects)
	if err != nil {
		return err
	}

	// Bloom
	for i := 0; i < common.ValidateShardCount(int(meta.BloomShardCount)); i++ {
		err = copy(bloomName(i))
		if err != nil {
			return err
		}
	}

	// Index
	err = copy(nameIndex)
	if err != nil {
		return err
	}

	// Meta
	err = dest.WriteBlockMeta(ctx, meta)
	return err
}
