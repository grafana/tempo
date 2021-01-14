package v0

import (
	"bytes"
	"context"
	"fmt"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/bloom"

	willf_bloom "github.com/willf/bloom"
)

type backendBlock struct {
	meta *backend.BlockMeta
}

// NewBackendBlock returns a block used for finding traces in the backend
func NewBackendBlock(meta *backend.BlockMeta) encoding.BackendBlock {
	return &backendBlock{
		meta: meta,
	}
}

// Find searches a block for the ID and returns an object if found.
func (b *backendBlock) Find(ctx context.Context, r backend.Reader, id encoding.ID, metrics *encoding.FindMetrics) ([]byte, error) {
	shardKey := bloom.ShardKeyForTraceID(id)
	blockID := b.meta.BlockID
	tenantID := b.meta.TenantID

	bloomBytes, err := r.Read(ctx, bloomName(shardKey), blockID, tenantID)
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

	indexBytes, err := r.Read(ctx, nameIndex, blockID, tenantID)
	metrics.IndexReads.Inc()
	metrics.IndexBytesRead.Add(int32(len(indexBytes)))
	if err != nil {
		return nil, fmt.Errorf("error reading index %w", err)
	}

	record, err := FindRecord(id, indexBytes) // todo: replace with backend.Finder
	if err != nil {
		return nil, fmt.Errorf("error finding record %w", err)
	}

	if record == nil {
		return nil, nil
	}

	objectBytes := make([]byte, record.Length)
	err = r.ReadRange(ctx, nameObjects, blockID, tenantID, record.Start, objectBytes)
	metrics.BlockReads.Inc()
	metrics.BlockBytesRead.Add(int32(len(objectBytes)))
	if err != nil {
		return nil, fmt.Errorf("error reading object %w", err)
	}

	iter := NewIterator(bytes.NewReader(objectBytes))
	var foundObject []byte
	for {
		iterID, iterObject, err := iter.Next()
		if iterID == nil {
			break
		}
		if err != nil {
			return nil, err
		}
		if bytes.Equal(iterID, id) {
			foundObject = iterObject
			break
		}
	}
	return foundObject, nil
}

// Iterator searches a block for the ID and returns an object if found.
func (b *backendBlock) Iterator(chunkSizeBytes uint32, r backend.Reader) (encoding.Iterator, error) {
	return NewBackendIterator(b.meta.TenantID, b.meta.BlockID, chunkSizeBytes, r)
}
