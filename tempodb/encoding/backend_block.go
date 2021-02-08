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
	encoding versionedEncoding

	meta   *backend.BlockMeta
	reader backend.Reader
}

// NewBackendBlock returns a BackendBlock for the given backend.BlockMeta
//  It is version aware.
func NewBackendBlock(meta *backend.BlockMeta, r backend.Reader) (*BackendBlock, error) {
	var encoding versionedEncoding

	switch meta.Version {
	case "v0":
		encoding = v0Encoding{}
	case "v1":
		encoding = v1Encoding{}
	default:
		return nil, fmt.Errorf("%s is not a valid block version", meta.Version)
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

	shardKey := common.ShardKeyForTraceID(id)
	blockID := b.meta.BlockID
	tenantID := b.meta.TenantID

	bloomBytes, err := b.reader.Read(ctx, b.encoding.nameBloom(shardKey), blockID, tenantID)
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

	indexBytes, err := b.reader.Read(ctx, b.encoding.nameIndex(), blockID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("error reading index (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}

	indexReader, err := b.encoding.newIndexReader(indexBytes)
	if err != nil {
		return nil, fmt.Errorf("error building index reader (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}

	ra := backend.NewReaderAt(b.meta, b.encoding.nameObjects(), b.reader)
	pageReader, err := b.encoding.newPageReader(ra, b.meta.Encoding)
	if err != nil {
		return nil, fmt.Errorf("error building page reader (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}

	// passing nil for objectCombiner here.  this is fine b/c a backend block should never have dupes
	finder := b.encoding.newPagedFinder(indexReader, pageReader, nil)
	objectBytes, err := finder.Find(id)

	if err != nil {
		return nil, fmt.Errorf("error using pageFinder (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}

	return objectBytes, nil
}

// Iterator returns an Iterator that iterates over the objects in the block from the backend
func (b *BackendBlock) Iterator(chunkSizeBytes uint32) (common.Iterator, error) {
	// read index
	indexBytes, err := b.reader.Read(context.TODO(), b.encoding.nameIndex(), b.meta.BlockID, b.meta.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to read index (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}

	ra := backend.NewReaderAt(b.meta, b.encoding.nameObjects(), b.reader)
	pageReader, err := b.encoding.newPageReader(ra, b.meta.Encoding)
	if err != nil {
		return nil, fmt.Errorf("failed to create pageReader (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}

	iterator, err := b.encoding.newPagedIterator(chunkSizeBytes, indexBytes, pageReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create iterator (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}

	return iterator, nil
}
