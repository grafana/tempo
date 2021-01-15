package encoding

import (
	"context"
	"fmt"
	"io"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
)

// Current contains a string indicating the most recent block version
const Current = "v0"

// BackendBlock defines an object that can be used to interact with a block in object storage
type BackendBlock interface {
	// Find searches for a given ID and returns the object if exists
	Find(ctx context.Context, r backend.Reader, id common.ID, metrics *common.FindMetrics) ([]byte, error)
	// Iterator returns an iterator that can be used to examine every object in the block
	Iterator(chunkSizeBytes uint32, r backend.Reader) (common.Iterator, error)
}

// newBackendBlock returns a BackendBlock for the given backend.BlockMeta
//  It is version aware.
func NewBackendBlock(meta *backend.BlockMeta) (BackendBlock, error) {
	if meta.Version == "v0" {
		return v0.NewBackendBlock(meta), nil
	}

	return nil, fmt.Errorf("%s is not a valid block version", meta.Version)
}

// newBufferedAppender returns the most recent Appender
func newBufferedAppender(writer io.Writer, indexDownsample int, totalObjectsEstimate int) common.Appender {
	return v0.NewBufferedAppender(writer, indexDownsample, totalObjectsEstimate)
}

// newDedupingFinder returns the most recent Finder
func newDedupingFinder(sortedRecords []*common.Record, ra io.ReaderAt, combiner common.ObjectCombiner) common.Finder {
	return v0.NewDedupingFinder(sortedRecords, ra, combiner)
}

// writeBlockMeta calls the most recent WriteBlockMeta
func writeBlockMeta(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, records []*common.Record, b *common.ShardedBloomFilter) error {
	return v0.WriteBlockMeta(ctx, w, meta, records, b)
}

// writeBlockData calls the most recent WriteBlockData
func writeBlockData(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, r io.Reader, size int64) error {
	return v0.WriteBlockData(ctx, w, meta, r, size)
}

// appendBlockData calls the most recent AppendBlockData
func appendBlockData(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, tracker backend.AppendTracker, buffer []byte) (backend.AppendTracker, error) {
	return v0.AppendBlockData(ctx, w, meta, tracker, buffer)
}
