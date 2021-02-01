package v0

import (
	"bytes"
	"context"
	"fmt"

	"github.com/opentracing/opentracing-go"
	willf_bloom "github.com/willf/bloom"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type BackendBlock struct {
	meta       *backend.BlockMeta
	pageReader common.PageReader
	reader     backend.Reader
}

// NewBackendBlock returns a block used for finding traces in the backend
func NewBackendBlock(meta *backend.BlockMeta, r backend.Reader) *BackendBlock {
	readerAt := backend.NewBackendReaderAt(meta, nameObjects, r)
	pageReader := NewPageReader(readerAt) // jpe - remove readerat shim?

	return &BackendBlock{
		meta:       meta,
		reader:     r,
		pageReader: pageReader,
	}
}

// Find searches a block for the ID and returns an object if found.
func (b *BackendBlock) Find(ctx context.Context, id common.ID, metrics *common.FindMetrics) ([]byte, error) { // jpe drop find metrics?  pagedfinder kind of ruins them
	var err error
	span, ctx := opentracing.StartSpanFromContext(ctx, "BackendBlock.Find")
	defer func() {
		if err != nil {
			span.SetTag("error", true)
		}
		span.Finish()
	}()

	span.SetTag("block", b.meta.BlockID.String())

	shardKey := common.ShardKeyForTraceID(id)
	blockID := b.meta.BlockID
	tenantID := b.meta.TenantID

	bloomBytes, err := b.reader.Read(ctx, bloomName(shardKey), blockID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving bloom %w", err)
	}

	filter := &willf_bloom.BloomFilter{}
	_, err = filter.ReadFrom(bytes.NewReader(bloomBytes))
	if err != nil {
		return nil, fmt.Errorf("error parsing bloom %w", err)
	}

	metrics.BloomFilterReads.Inc()
	metrics.BloomFilterBytesRead.Add(int32(len(bloomBytes)))
	if !filter.Test(id) {
		return nil, nil
	}

	indexBytes, err := b.reader.Read(ctx, nameIndex, blockID, tenantID)
	metrics.IndexReads.Inc()
	metrics.IndexBytesRead.Add(int32(len(indexBytes)))
	if err != nil {
		return nil, fmt.Errorf("error reading index %w", err)
	}

	indexReader, err := NewIndexReaderBytes(indexBytes)
	if err != nil {
		return nil, fmt.Errorf("error building index reader %w", err)
	}

	finder := NewPagedFinder(indexReader, b.pageReader, nil) // jpe : nil ok?  take combiner? return slice of slices and let something else combine?
	objectBytes, err := finder.Find(id)

	if err != nil {
		return nil, fmt.Errorf("error finding using pagedFinder %w", err)
	}

	return objectBytes, nil
}

// Iterator returns an Iterator that iterates over the objects in the block from the backend
func (b *BackendBlock) Iterator(chunkSizeBytes uint32) (common.Iterator, error) {
	// read index
	indexBytes, err := b.reader.Read(context.TODO(), nameIndex, b.meta.BlockID, b.meta.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to read index while building iterator: %w", err)
	}

	indexReader, err := NewIndexReaderBytes(indexBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create index reader building iterator: %w", err)
	}

	return NewPagedIterator(chunkSizeBytes, indexReader, b.pageReader), nil
}
