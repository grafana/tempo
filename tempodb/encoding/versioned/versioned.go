package versioned

import (
	"context"
	"fmt"
	"io"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/bloom"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
)

// Current contains a string indicating the most recent block version
const Current = "v0"

// NewBackendBlock returns a encoding.BackendBlock for the given backend.BlockMeta
//  It is version aware.
func NewBackendBlock(meta *backend.BlockMeta) (encoding.BackendBlock, error) {
	if meta.Version == "v0" {
		return v0.NewBackendBlock(meta), nil
	}

	return nil, fmt.Errorf("%s is not a valid block version", meta.Version)
}

// NewBufferedAppender returns the most recent encoding.Appender
func NewBufferedAppender(writer io.Writer, indexDownsample int, totalObjectsEstimate int) encoding.Appender {
	return v0.NewBufferedAppender(writer, indexDownsample, totalObjectsEstimate)
}

// NewDedupingFinder returns the most recent encoding.Finder
func NewDedupingFinder(sortedRecords []*encoding.Record, ra io.ReaderAt, combiner encoding.ObjectCombiner) encoding.Finder {
	return v0.NewDedupingFinder(sortedRecords, ra, combiner)
}

// WriteBlockMeta calls the most recent WriteBlockMeta
func WriteBlockMeta(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, records []*encoding.Record, b bloom.ShardedBloomFilter) error {
	return v0.WriteBlockMeta(ctx, w, meta, records, b)
}

// WriteBlockData calls the most recent WriteBlockData
func WriteBlockData(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, r io.Reader, size int64) error {
	return v0.WriteBlockData(ctx, w, meta, r, size)
}
