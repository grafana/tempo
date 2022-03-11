package encoding

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/opentracing/opentracing-go"
	willf_bloom "github.com/willf/bloom"
	"go.uber.org/atomic"

	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
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

	nameBloom := bloomName(shardKey)
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

	indexReaderAt := backend.NewContextReader(b.meta, nameIndex, b.reader, false)
	indexReader, err := b.encoding.NewIndexReader(indexReaderAt, int(b.meta.IndexPageSize), int(b.meta.TotalRecords))
	if err != nil {
		return nil, fmt.Errorf("error building index reader (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}

	ra := backend.NewContextReader(b.meta, nameObjects, b.reader, false)
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
	ra := backend.NewContextReader(b.meta, nameObjects, b.reader, false)
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

// PartialIterator returns an Iterator that iterates over the a subset of pages in the block from the backend
func (b *BackendBlock) PartialIterator(chunkSizeBytes uint32, startPage int, totalPages int) (Iterator, error) {
	// read index
	ra := backend.NewContextReader(b.meta, nameObjects, b.reader, false)
	dataReader, err := b.encoding.NewDataReader(ra, b.meta.Encoding)
	if err != nil {
		return nil, fmt.Errorf("failed to create dataReader (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}

	reader, err := b.NewIndexReader()
	if err != nil {
		return nil, err
	}

	return newPartialPagedIterator(chunkSizeBytes, reader, dataReader, b.encoding.NewObjectReaderWriter(), startPage, totalPages), nil
}

func (b *BackendBlock) NewIndexReader() (common.IndexReader, error) {
	indexReaderAt := backend.NewContextReader(b.meta, nameIndex, b.reader, false)
	reader, err := b.encoding.NewIndexReader(indexReaderAt, int(b.meta.IndexPageSize), int(b.meta.TotalRecords))
	if err != nil {
		return nil, fmt.Errorf("failed to create index reader (%s, %s): %w", b.meta.TenantID, b.meta.BlockID, err)
	}

	return reader, nil
}

func (b *BackendBlock) BlockMeta() *backend.BlockMeta {
	return b.meta
}

var _ Finder = (*BackendBlock)(nil)

func (b *BackendBlock) FindTraceByID(ctx context.Context, id common.ID) (*tempopb.Trace, error) {
	obj, err := b.Find(ctx, id)
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

var _ Searcher = (*BackendBlock)(nil)

func (b *BackendBlock) Search(ctx context.Context, req *tempopb.SearchRequest, opts ...SearchOption) (resp *tempopb.SearchResponse, err error) {

	// Options
	opt := SearchOptions{
		chunkSizeBytes: 10_000_000,
	}
	for _, o := range opts {
		o(&opt)
	}

	decoder, err := model.NewObjectDecoder(b.meta.DataEncoding)
	if err != nil {
		return nil, fmt.Errorf("failed to create NewDecoder: %w", err)
	}

	// Iterator
	var iter Iterator
	if opt.totalPages > 0 {
		iter, err = b.PartialIterator(opt.chunkSizeBytes, opt.startPage, opt.totalPages)
	} else {
		iter, err = b.Iterator(opt.chunkSizeBytes)
	}
	if err != nil {
		return nil, err
	}
	if opt.prefetchTraceCount > 0 {
		iter = NewPrefetchIterator(ctx, iter, opt.prefetchTraceCount)
	}
	defer iter.Close()

	respMtx := sync.Mutex{}
	resp = &tempopb.SearchResponse{
		Metrics: &tempopb.SearchMetrics{},
	}

	var searchErr error
	wg := boundedwaitgroup.New(5)
	done := atomic.Bool{}
	for {
		id, obj, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error iterating %s, %w", b.meta.BlockID, err)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			isDone, err := search(decoder, opt.maxBytes, id, obj, &respMtx, req, resp)
			if isDone {
				done.Store(true)
			}

			if err != nil {
				respMtx.Lock()
				searchErr = err
				respMtx.Unlock()
			}
		}()

		if done.Load() {
			break
		}
	}
	wg.Wait()

	if err != nil {
		return nil, err
	}
	if searchErr != nil {
		return nil, searchErr
	}
	return resp, nil
}

func search(decoder model.ObjectDecoder, maxBytes int, id common.ID, obj []byte, respMtx *sync.Mutex, req *tempopb.SearchRequest, resp *tempopb.SearchResponse) (bool, error) {
	respMtx.Lock()
	resp.Metrics.InspectedTraces++
	resp.Metrics.InspectedBytes += uint64(len(obj))
	respMtx.Unlock()

	if maxBytes > 0 && len(obj) > maxBytes {
		respMtx.Lock()
		resp.Metrics.SkippedTraces++
		respMtx.Unlock()
		return false, nil
	}

	metadata, err := decoder.Matches(id, obj, req)

	respMtx.Lock()
	defer respMtx.Unlock()
	if err != nil {
		return false, err
	}
	if metadata == nil {
		return false, nil // No match, keep going.
	}

	// Found a match
	resp.Traces = append(resp.Traces, metadata)
	return len(resp.Traces) >= int(req.Limit), nil
}
