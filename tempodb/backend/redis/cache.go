package redis

import (
	"context"
	"io"
	"time"

	"github.com/cortexproject/cortex/pkg/chunk/cache"
	"github.com/go-kit/kit/log"
	"github.com/grafana/tempo/tempodb/backend"

	"github.com/google/uuid"
)

const (
	typeBloom = "bloom"
	typeIndex = "index"
)

type Config struct {
	ClientConfig cache.RedisConfig `yaml:",inline"`

	TTL time.Duration `yaml:"ttl"`
}

type readerWriter struct {
	nextReader backend.Reader
	nextWriter backend.Writer
	client     *cache.RedisCache
	logger     log.Logger
}

func New(nextReader backend.Reader, nextWriter backend.Writer, cfg *Config, logger log.Logger) (backend.Reader, backend.Writer, error) {
	if cfg.ClientConfig.Timeout == 0 {
		cfg.ClientConfig.Timeout = 100 * time.Millisecond
	}
	if cfg.ClientConfig.Expiration == 0 {
		cfg.ClientConfig.Expiration = cfg.TTL
	}

	client := cache.NewRedisClient(&cfg.ClientConfig)

	rw := &readerWriter{
		client:     cache.NewRedisCache("tempo", client, logger),
		nextReader: nextReader,
		nextWriter: nextWriter,
		logger:     logger,
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
	val := r.get(ctx, key)
	if val != nil {
		return val, nil
	}

	val, err := r.nextReader.Read(ctx, name, blockID, tenantID)
	if err == nil {
		r.set(ctx, key, val)
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
	r.client.Stop()
}

// Write implements backend.Writer
func (r *readerWriter) Write(ctx context.Context, name string, blockID uuid.UUID, tenantID string, buffer []byte) error {
	r.set(ctx, key(blockID, tenantID, name), buffer)

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

func (r *readerWriter) get(ctx context.Context, key string) []byte {
	found, vals, _ := r.client.Fetch(ctx, []string{key})
	if len(found) > 0 {
		return vals[0]
	}
	return nil
}

func (r *readerWriter) set(ctx context.Context, key string, val []byte) {
	r.client.Store(ctx, []string{key}, [][]byte{val})
}

func key(blockID uuid.UUID, tenantID string, name string) string {
	return blockID.String() + ":" + tenantID + ":" + name
}
