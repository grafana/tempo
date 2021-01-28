package cache

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
)

type readerWriter struct {
	nextReader backend.Reader
	nextWriter backend.Writer
	cache      Client
}

type Client interface {
	Fetch(ctx context.Context, key string) []byte
	Store(ctx context.Context, key string, val []byte)
	Shutdown()
}

func NewCache(nextReader backend.Reader, nextWriter backend.Writer, cache Client) (backend.Reader, backend.Writer, error) {
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
	val := r.cache.Fetch(ctx, key)
	if val != nil {
		return val, nil
	}

	val, err := r.nextReader.Read(ctx, name, blockID, tenantID)
	if err == nil {
		r.cache.Store(ctx, key, val)
	}

	return val, err
}

// ReadRange implements backend.Reader
func (r *readerWriter) ReadRange(ctx context.Context, name string, blockID uuid.UUID, tenantID string, offset uint64, buffer []byte) error {
	return r.nextReader.ReadRange(ctx, name, blockID, tenantID, offset, buffer)
}

// Shutdown implements backend.Reader
func (r *readerWriter) Shutdown() {
	r.nextReader.Shutdown()
	r.cache.Shutdown()
}

// Write implements backend.Writer
func (r *readerWriter) Write(ctx context.Context, name string, blockID uuid.UUID, tenantID string, buffer []byte) error {
	r.cache.Store(ctx, key(blockID, tenantID, name), buffer)

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
