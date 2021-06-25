package encoding

import (
	"bytes"
	"context"
	"fmt"

	"github.com/opentracing/opentracing-go"
	willf_bloom "github.com/willf/bloom"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// BackendBlock represents a block already in the backend.
type BackendBlock struct {
	encoding VersionedEncoding

	meta   *backend.BlockMeta
	reader backend.Reader
}

// NewBackendBlock returns a BackendBlock for the given backend.BlockMeta
//  It is version aware.
func NewBackendBlock(meta *backend.BlockMeta, r backend.Reader) (*BackendBlock, error) {
	encoding, err := FromVersion(meta.Version)
	if err != nil {
		return nil, err
	}

	return &BackendBlock{
		encoding: encoding,
		meta:     meta,
		reader:   r,
	}, nil
}

// Find searches a block for the ID and returns an object if found.
func (b *BackendBlock) Find(ctx context.Context, id common.ID) ([]byte, error) {
	var err error
	span, ctx := opentracing.StartSpanFromContext(ctx, "BackendBlock.Find")
	defer func() {
		if err != nil {
			span.SetTag("error", true)
		}
		span.Finish()
	}()

	span.SetTag("block", b.meta.BlockID.String())

	shardKey := common.ShardKeyForTraceID(id, int(b.meta.BloomShardCount))
	blockID := b.meta.BlockID
	tenantID := b.meta.TenantID

	bloomBytes, err := b.reader.Read(ctx, bloomName(shardKey), backend.KeyPathForBlock(blockID, tenantID))
	if err != nil {
		return nil, fmt.Errorf("error retrieving bloom (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}

	filter := &willf_bloom.BloomFilter{}
	_, err = filter.ReadFrom(bytes.NewReader(bloomBytes))
	if err != nil {
		return nil, fmt.Errorf("error parsing bloom (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}

	if !filter.Test(id) {
		return nil, nil
	}

	indexReaderAt := backend.NewContextReader(b.meta, nameIndex, b.reader)
	indexReader, err := b.encoding.NewIndexReader(indexReaderAt, int(b.meta.IndexPageSize), int(b.meta.TotalRecords))
	if err != nil {
		return nil, fmt.Errorf("error building index reader (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}

	ra := backend.NewContextReader(b.meta, nameObjects, b.reader)
	dataReader, err := b.encoding.NewDataReader(ra, b.meta.Encoding)
	if err != nil {
		return nil, fmt.Errorf("error building page reader (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}
	defer dataReader.Close()

	// passing nil for objectCombiner here.  this is fine b/c a backend block should never have dupes
	finder := NewPagedFinder(indexReader, dataReader, nil, b.encoding.NewObjectReaderWriter(), b.meta.DataEncoding)
	objectBytes, err := finder.Find(ctx, id)

	if err != nil {
		return nil, fmt.Errorf("error using pageFinder (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}

	return objectBytes, nil
}

// Iterator returns an Iterator that iterates over the objects in the block from the backend
func (b *BackendBlock) Iterator(chunkSizeBytes uint32) (Iterator, error) {
	// read index
	ra := backend.NewContextReader(b.meta, nameObjects, b.reader)
	dataReader, err := b.encoding.NewDataReader(ra, b.meta.Encoding)
	if err != nil {
		return nil, fmt.Errorf("failed to create dataReader (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}

	reader, err := b.NewIndexReader()
	if err != nil {
		return nil, err
	}

	return newPagedIterator(chunkSizeBytes, reader, dataReader, b.encoding.NewObjectReaderWriter()), nil
}

func (b *BackendBlock) NewIndexReader() (common.IndexReader, error) {
	indexReaderAt := backend.NewContextReader(b.meta, nameIndex, b.reader)
	reader, err := b.encoding.NewIndexReader(indexReaderAt, int(b.meta.IndexPageSize), int(b.meta.TotalRecords))
	if err != nil {
		return nil, fmt.Errorf("failed to create index reader (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}

	return reader, nil
}

func (b *BackendBlock) BlockMeta() *backend.BlockMeta {
	return b.meta
}
