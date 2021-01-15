package encoding

import (
	"bytes"
	"context"
	"fmt"

	"github.com/opentracing/opentracing-go"
	willf_bloom "github.com/willf/bloom"
	"go.uber.org/atomic"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/bloom"
)

// FindMetrics is a threadsafe struct for tracking metrics related to a parallelized query
type FindMetrics struct {
	BloomFilterReads     *atomic.Int32
	BloomFilterBytesRead *atomic.Int32
	IndexReads           *atomic.Int32
	IndexBytesRead       *atomic.Int32
	BlockReads           *atomic.Int32
	BlockBytesRead       *atomic.Int32
}

func NewFindMetrics() FindMetrics {
	return FindMetrics{
		BloomFilterReads:     atomic.NewInt32(0),
		BloomFilterBytesRead: atomic.NewInt32(0),
		IndexReads:           atomic.NewInt32(0),
		IndexBytesRead:       atomic.NewInt32(0),
		BlockReads:           atomic.NewInt32(0),
		BlockBytesRead:       atomic.NewInt32(0),
	}
}

// BackendBlock defines an object that can find traces
type BackendBlock interface {
	Find(ctx context.Context, r backend.Reader, id ID, metrics *FindMetrics) ([]byte, error)
}

type backendBlock struct {
	meta *backend.BlockMeta
}

// NewBackendBlock returns a block used for finding traces in the backend
func NewBackendBlock(meta *backend.BlockMeta) BackendBlock {
	return &backendBlock{
		meta: meta,
	}
}

// Find searches a block for the ID and returns an object if found.
func (b *backendBlock) Find(ctx context.Context, r backend.Reader, id ID, metrics *FindMetrics) ([]byte, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "BackendBlock.Find")
	defer span.Finish()

	span.SetTag("block", b.meta.BlockID.String())

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
