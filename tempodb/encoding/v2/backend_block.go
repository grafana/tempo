package v2

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/opentracing/opentracing-go"
	willf_bloom "github.com/willf/bloom"

	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// BackendBlock represents a block already in the backend.
type BackendBlock struct {
	meta   *backend.BlockMeta
	reader backend.Reader
}

var _ common.Finder = (*BackendBlock)(nil)
var _ common.Searcher = (*BackendBlock)(nil)

// NewBackendBlock returns a BackendBlock for the given backend.BlockMeta
//  It is version aware.
func NewBackendBlock(meta *backend.BlockMeta, r backend.Reader) (*BackendBlock, error) {

	return &BackendBlock{
		meta:   meta,
		reader: r,
	}, nil
}

// Find searches a block for the ID and returns an object if found.
func (b *BackendBlock) find(ctx context.Context, id common.ID) ([]byte, error) {
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

	nameBloom := common.BloomName(shardKey)
	bloomBytes, err := b.reader.Read(ctx, nameBloom, blockID, tenantID, true)
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

	indexReaderAt := backend.NewContextReader(b.meta, common.NameIndex, b.reader, false)
	indexReader, err := NewIndexReader(indexReaderAt, int(b.meta.IndexPageSize), int(b.meta.TotalRecords))
	if err != nil {
		return nil, fmt.Errorf("error building index reader (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}

	ra := backend.NewContextReader(b.meta, common.NameObjects, b.reader, false)
	dataReader, err := NewDataReader(ra, b.meta.Encoding)
	if err != nil {
		return nil, fmt.Errorf("error building page reader (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}
	defer dataReader.Close()

	// passing nil for objectCombiner here.  this is fine b/c a backend block should never have dupes
	finder := NewPagedFinder(indexReader, dataReader, nil, NewObjectReaderWriter(), b.meta.DataEncoding)
	objectBytes, err := finder.Find(ctx, id)

	if err != nil {
		return nil, fmt.Errorf("error using pageFinder (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}

	return objectBytes, nil
}

// Iterator returns an Iterator that iterates over the objects in the block from the backend
func (b *BackendBlock) Iterator(chunkSizeBytes uint32) (Iterator, error) {
	// read index
	ra := backend.NewContextReader(b.meta, common.NameObjects, b.reader, false)
	dataReader, err := NewDataReader(ra, b.meta.Encoding)
	if err != nil {
		return nil, fmt.Errorf("failed to create dataReader (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}

	reader, err := b.NewIndexReader()
	if err != nil {
		return nil, err
	}

	return newPagedIterator(chunkSizeBytes, reader, dataReader, NewObjectReaderWriter()), nil
}

// partialIterator returns an Iterator that iterates over the a subset of pages in the block from the backend
func (b *BackendBlock) partialIterator(chunkSizeBytes uint32, startPage int, totalPages int) (Iterator, error) {
	// read index
	ra := backend.NewContextReader(b.meta, common.NameObjects, b.reader, false)
	dataReader, err := NewDataReader(ra, b.meta.Encoding)
	if err != nil {
		return nil, fmt.Errorf("failed to create dataReader (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}

	reader, err := b.NewIndexReader()
	if err != nil {
		return nil, err
	}

	return newPartialPagedIterator(chunkSizeBytes, reader, dataReader, NewObjectReaderWriter(), startPage, totalPages), nil
}

func (b *BackendBlock) NewIndexReader() (common.IndexReader, error) {
	indexReaderAt := backend.NewContextReader(b.meta, common.NameIndex, b.reader, false)
	reader, err := NewIndexReader(indexReaderAt, int(b.meta.IndexPageSize), int(b.meta.TotalRecords))
	if err != nil {
		return nil, fmt.Errorf("failed to create index reader (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}

	return reader, nil
}

func (b *BackendBlock) BlockMeta() *backend.BlockMeta {
	return b.meta
}

func (b *BackendBlock) FindTraceByID(ctx context.Context, id common.ID) (*tempopb.Trace, error) {
	obj, err := b.find(ctx, id)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		// Not found in this block
		return nil, nil
	}

	dec, err := model.NewObjectDecoder(b.meta.DataEncoding)
	if err != nil {
		return nil, err
	}
	return dec.PrepareForRead(obj)
}

func (b *BackendBlock) Search(ctx context.Context, req *tempopb.SearchRequest, opt common.SearchOptions) (resp *tempopb.SearchResponse, err error) {

	decoder, err := model.NewObjectDecoder(b.meta.DataEncoding)
	if err != nil {
		return nil, fmt.Errorf("failed to create NewDecoder: %w", err)
	}

	// Iterator
	var iter Iterator
	if opt.TotalPages > 0 {
		iter, err = b.partialIterator(opt.ChunkSizeBytes, opt.StartPage, opt.TotalPages)
	} else {
		iter, err = b.Iterator(opt.ChunkSizeBytes)
	}
	if err != nil {
		return nil, err
	}
	if opt.PrefetchTraceCount > 0 {
		iter = NewPrefetchIterator(ctx, iter, opt.PrefetchTraceCount)
	}
	defer iter.Close()

	resp = &tempopb.SearchResponse{
		Metrics: &tempopb.SearchMetrics{},
	}

	for {
		id, obj, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error iterating %s, %w", b.meta.BlockID, err)
		}

		err = search(decoder, opt.MaxBytes, id, obj, req, resp)
		if err != nil {
			return nil, err
		}

		if len(resp.Traces) >= int(req.Limit) {
			break
		}
	}

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func search(decoder model.ObjectDecoder, maxBytes int, id common.ID, obj []byte, req *tempopb.SearchRequest, resp *tempopb.SearchResponse) error {
	resp.Metrics.InspectedTraces++
	resp.Metrics.InspectedBytes += uint64(len(obj))

	if maxBytes > 0 && len(obj) > maxBytes {
		resp.Metrics.SkippedTraces++
		return nil
	}

	metadata, err := decoder.Matches(id, obj, req)
	if err != nil {
		return err
	}

	if metadata != nil {
		// Found a match
		resp.Traces = append(resp.Traces, metadata)
	}

	return nil
}
