package cache

import (
	"context"
	"io"

	cortex_cache "github.com/cortexproject/cortex/pkg/chunk/cache"
	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
)

type readerWriter struct {
	nextReader backend.Reader
	nextWriter backend.Writer
	cache      cortex_cache.Cache
}

func NewCache(nextReader backend.Reader, nextWriter backend.Writer, cache cortex_cache.Cache) (backend.Reader, backend.Writer, error) {
	rw := &readerWriter{
		cache:      cache,
		nextReader: nextReader,
		nextWriter: nextWriter,
	}

	return rw, rw, nil
}

// Tenants implements backend.Reader
func (r *readerWriter) Tenants(ctx context.Context) ([]string, error) {
	return r.nextReader.Tenants(ctx)
}

// Blocks implements backend.Reader
func (r *readerWriter) Blocks(ctx context.Context, tenantID string) ([]uuid.UUID, error) {
	return r.nextReader.Blocks(ctx, tenantID)
}

// BlockMeta implements backend.Reader
func (r *readerWriter) BlockMeta(ctx context.Context, blockID uuid.UUID, tenantID string) (*backend.BlockMeta, error) {
	return r.nextReader.BlockMeta(ctx, blockID, tenantID)
}

// Read implements backend.Reader
func (r *readerWriter) Read(ctx context.Context, name string, blockID uuid.UUID, tenantID string) ([]byte, error) {
	key := key(blockID, tenantID, name)
	found, vals, _ := r.cache.Fetch(ctx, []string{key})
	if len(found) > 0 {
		return vals[0], nil
	}

	val, err := r.nextReader.Read(ctx, name, blockID, tenantID)
	if err == nil {
		r.cache.Store(ctx, []string{key}, [][]byte{val})
	}

	return val, err
}

func (r *readerWriter) ReadReader(ctx context.Context, name string, blockID uuid.UUID, tenantID string) (io.ReadCloser, int64, error) {
	panic("ReadReader is not yet supported for cache")
}

// ReadRange implements backend.Reader
func (r *readerWriter) ReadRange(ctx context.Context, name string, blockID uuid.UUID, tenantID string, offset uint64, buffer []byte) error {
	return r.nextReader.ReadRange(ctx, name, blockID, tenantID, offset, buffer)
}

// Shutdown implements backend.Reader
func (r *readerWriter) Shutdown() {
	r.nextReader.Shutdown()
	r.cache.Stop()
}

// Write implements backend.Writer
func (r *readerWriter) Write(ctx context.Context, name string, blockID uuid.UUID, tenantID string, buffer []byte) error {
	r.cache.Store(ctx, []string{key(blockID, tenantID, name)}, [][]byte{buffer})

	return r.nextWriter.Write(ctx, name, blockID, tenantID, buffer)
}

// Write implements backend.Writer
func (r *readerWriter) WriteReader(ctx context.Context, name string, blockID uuid.UUID, tenantID string, data io.Reader, size int64) error {
	return r.nextWriter.WriteReader(ctx, name, blockID, tenantID, data, size)
}

// WriteBlockMeta implements backend.Writer
func (r *readerWriter) WriteBlockMeta(ctx context.Context, meta *backend.BlockMeta) error {
	return r.nextWriter.WriteBlockMeta(ctx, meta)
}

// Append implements backend.Writer
func (r *readerWriter) Append(ctx context.Context, name string, blockID uuid.UUID, tenantID string, tracker backend.AppendTracker, buffer []byte) (backend.AppendTracker, error) {
	return r.nextWriter.Append(ctx, name, blockID, tenantID, tracker, buffer)
}

// CloseAppend implements backend.Writer
func (r *readerWriter) CloseAppend(ctx context.Context, tracker backend.AppendTracker) error {
	return r.nextWriter.CloseAppend(ctx, tracker)
}

func key(blockID uuid.UUID, tenantID string, name string) string {
	return blockID.String() + ":" + tenantID + ":" + name
}
