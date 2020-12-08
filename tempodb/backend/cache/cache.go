package cache

import (
	"context"
	"strconv"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
)

const (
	typeBloom = "bloom"
	typeIndex = "index"
)

type readerWriter struct {
	nextReader backend.Reader
	nextWriter backend.Writer
	cache      Cache
}

type Cache interface {
	Fetch(ctx context.Context, key string) []byte
	Store(ctx context.Context, key string, val []byte)
	Shutdown()
}

func NewCache(nextReader backend.Reader, nextWriter backend.Writer, cache Cache) (backend.Reader, backend.Writer, error) {
	rw := &readerWriter{
		cache:      cache,
		nextReader: nextReader,
		nextWriter: nextWriter,
	}

	return rw, rw, nil
}

// Reader
func (r *readerWriter) Tenants(ctx context.Context) ([]string, error) {
	return r.nextReader.Tenants(ctx)
}

func (r *readerWriter) Blocks(ctx context.Context, tenantID string) ([]uuid.UUID, error) {
	return r.nextReader.Blocks(ctx, tenantID)
}

func (r *readerWriter) BlockMeta(ctx context.Context, blockID uuid.UUID, tenantID string) (*encoding.BlockMeta, error) {
	return r.nextReader.BlockMeta(ctx, blockID, tenantID)
}

func (r *readerWriter) Bloom(ctx context.Context, blockID uuid.UUID, tenantID string, shardNum int) ([]byte, error) {
	key := bloomKey(blockID, tenantID, shardNum)
	val := r.cache.Fetch(ctx, key)
	if val != nil {
		return val, nil
	}

	val, err := r.nextReader.Bloom(ctx, blockID, tenantID, shardNum)
	if err == nil {
		r.cache.Store(ctx, key, val)
	}

	return val, err
}

func (r *readerWriter) Index(ctx context.Context, blockID uuid.UUID, tenantID string) ([]byte, error) {
	key := key(blockID, tenantID)
	val := r.cache.Fetch(ctx, key)
	if val != nil {
		return val, nil
	}

	val, err := r.nextReader.Index(ctx, blockID, tenantID)
	if err == nil {
		r.cache.Store(ctx, key, val)
	}

	return val, err
}

func (r *readerWriter) Object(ctx context.Context, blockID uuid.UUID, tenantID string, start uint64, buffer []byte) error {
	return r.nextReader.Object(ctx, blockID, tenantID, start, buffer)
}

// Writer
func (r *readerWriter) Write(ctx context.Context, meta *encoding.BlockMeta, bBloom [][]byte, bIndex []byte, objectFilePath string) error {
	for i, b := range bBloom {
		r.cache.Store(ctx, bloomKey(meta.BlockID, meta.TenantID, i), b)
	}
	r.cache.Store(ctx, key(meta.BlockID, meta.TenantID), bIndex)

	return r.nextWriter.Write(ctx, meta, bBloom, bIndex, objectFilePath)
}

func (r *readerWriter) WriteBlockMeta(ctx context.Context, tracker backend.AppendTracker, meta *encoding.BlockMeta, bBloom [][]byte, bIndex []byte) error {
	for i, b := range bBloom {
		r.cache.Store(ctx, bloomKey(meta.BlockID, meta.TenantID, i), b)
	}
	r.cache.Store(ctx, key(meta.BlockID, meta.TenantID), bIndex)

	return r.nextWriter.WriteBlockMeta(ctx, tracker, meta, bBloom, bIndex)
}

func (r *readerWriter) AppendObject(ctx context.Context, tracker backend.AppendTracker, meta *encoding.BlockMeta, bObject []byte) (backend.AppendTracker, error) {
	return r.nextWriter.AppendObject(ctx, tracker, meta, bObject)
}

func (r *readerWriter) Shutdown() {
	r.nextReader.Shutdown()
	r.cache.Shutdown()
}

func key(blockID uuid.UUID, tenantID string) string {
	return blockID.String() + ":" + tenantID + ":" + typeIndex
}

func bloomKey(blockID uuid.UUID, tenantID string, shardNum int) string {
	return blockID.String() + ":" + tenantID + ":" + typeBloom + strconv.Itoa(shardNum)
}
