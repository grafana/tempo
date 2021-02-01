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
	Find(ctx context.Context, id common.ID) ([]byte, error) // jpe - interface goes, move BackendBlock up to this level
	// Iterator returns an iterator that can be used to examine every object in the block
	Iterator(chunkSizeBytes uint32) (common.Iterator, error)
}

// NewBackendBlock returns a BackendBlock for the given backend.BlockMeta
//  It is version aware.
func NewBackendBlock(meta *backend.BlockMeta, r backend.Reader) (BackendBlock, error) {
	if meta.Version == "v0" {
		return v0.NewBackendBlock(meta, r), nil
	}

	return nil, fmt.Errorf("%s is not a valid block version", meta.Version)
}

// versionedEncoding has a whole bunch of versioned functionality.  This is
//  currently quite sloppy and could easily be tightened up to just a few methods
//  but it is what it is for now!
type versionedEncoding interface {
	newBufferedAppender(writer io.Writer, indexDownsample int, totalObjectsEstimate int) common.Appender
	newPagedFinder(sortedRecords []*common.Record, ra io.ReaderAt, combiner common.ObjectCombiner) common.Finder

	writeBlockMeta(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, records []*common.Record, b *common.ShardedBloomFilter) error
	writeBlockData(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, r io.Reader, size int64) error
	appendBlockData(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, tracker backend.AppendTracker, buffer []byte) (backend.AppendTracker, error)
}

// latestEncoding is used by Compactor and Complete block
func latestEncoding() versionedEncoding {
	return v0Encoding{}
}

// v0Encoding
type v0Encoding struct{}

func (v v0Encoding) newBufferedAppender(writer io.Writer, indexDownsample int, totalObjectsEstimate int) common.Appender {
	return v0.NewBufferedAppender(writer, indexDownsample, totalObjectsEstimate)
}
func (v v0Encoding) newPagedFinder(sortedRecords []*common.Record, ra io.ReaderAt, combiner common.ObjectCombiner) common.Finder {
	return v0.NewPagedFinder(v0.NewIndexReaderRecords(sortedRecords), v0.NewPageReader(ra), combiner)
}
func (v v0Encoding) writeBlockMeta(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, records []*common.Record, b *common.ShardedBloomFilter) error {
	return v0.WriteBlockMeta(ctx, w, meta, records, b)
}
func (v v0Encoding) writeBlockData(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, r io.Reader, size int64) error {
	return v0.WriteBlockData(ctx, w, meta, r, size)
}
func (v v0Encoding) appendBlockData(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, tracker backend.AppendTracker, buffer []byte) (backend.AppendTracker, error) {
	return v0.AppendBlockData(ctx, w, meta, tracker, buffer)
}
