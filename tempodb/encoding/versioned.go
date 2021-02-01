package encoding

import (
	"context"
	"io"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
)

// versionedEncoding has a whole bunch of versioned functionality.  This is
//  currently quite sloppy and could easily be tightened up to just a few methods
//  but it is what it is for now!
type versionedEncoding interface {
	newBufferedAppender(writer io.Writer, indexDownsample int, totalObjectsEstimate int) common.Appender
	newPagedFinder(indexReader common.IndexReader, ra io.ReaderAt, combiner common.ObjectCombiner) common.Finder
	newPagedIterator(chunkSizeBytes uint32, indexBytes []byte, ra io.ReaderAt) (common.Iterator, error)

	writeBlockMeta(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, records []*common.Record, b *common.ShardedBloomFilter) error
	writeBlockData(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, r io.Reader, size int64) error
	appendBlockData(ctx context.Context, w backend.Writer, meta *backend.BlockMeta, tracker backend.AppendTracker, buffer []byte) (backend.AppendTracker, error)

	newIndexReaderBytes(indexBytes []byte) (common.IndexReader, error)
	newIndexReaderRecords(records []*common.Record) common.IndexReader

	nameIndex() string
	nameObjects() string
	nameBloom(shard int) string
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
func (v v0Encoding) newPagedIterator(chunkSizeBytes uint32, indexBytes []byte, ra io.ReaderAt) (common.Iterator, error) {
	reader, err := v0.NewIndexReaderBytes(indexBytes)
	if err != nil {
		return nil, err
	}

	return v0.NewPagedIterator(chunkSizeBytes, reader, v0.NewPageReader(ra)), nil
}
func (v v0Encoding) newPagedFinder(indexReader common.IndexReader, ra io.ReaderAt, combiner common.ObjectCombiner) common.Finder {
	return v0.NewPagedFinder(indexReader, v0.NewPageReader(ra), combiner)
}
func (v v0Encoding) newIndexReaderBytes(indexBytes []byte) (common.IndexReader, error) {
	return v0.NewIndexReaderBytes(indexBytes)
}
func (v v0Encoding) newIndexReaderRecords(records []*common.Record) common.IndexReader {
	return v0.NewIndexReaderRecords(records)
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
func (v v0Encoding) nameObjects() string {
	return v0.NameObjects
}
func (v v0Encoding) nameIndex() string {
	return v0.NameIndex
}
func (v v0Encoding) nameBloom(shard int) string {
	return v0.BloomName(shard)
}
