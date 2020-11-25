package redis

import (
	"context"
	"strconv"
	"time"

	"github.com/cortexproject/cortex/pkg/chunk/cache"
	"github.com/go-kit/kit/log"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"

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
	val := r.get(ctx, key)
	if val != nil {
		return val, nil
	}

	val, err := r.nextReader.Bloom(ctx, blockID, tenantID, shardNum)
	if err == nil {
		r.set(ctx, key, val)
	}

	return val, err
}

func (r *readerWriter) Index(ctx context.Context, blockID uuid.UUID, tenantID string) ([]byte, error) {
	key := key(blockID, tenantID)
	val := r.get(ctx, key)
	if val != nil {
		return val, nil
	}

	val, err := r.nextReader.Index(ctx, blockID, tenantID)
	if err == nil {
		r.set(ctx, key, val)
	}

	return val, err
}

func (r *readerWriter) Object(ctx context.Context, blockID uuid.UUID, tenantID string, start uint64, buffer []byte) error {
	return r.nextReader.Object(ctx, blockID, tenantID, start, buffer)
}

func (r *readerWriter) Shutdown() {
	r.nextReader.Shutdown()
	r.client.Stop()
}

// Writer
func (r *readerWriter) Write(ctx context.Context, meta *encoding.BlockMeta, bBloom [][]byte, bIndex []byte, objectFilePath string) error {
	for i, b := range bBloom {
		r.set(ctx, bloomKey(meta.BlockID, meta.TenantID, i), b)
	}
	r.set(ctx, key(meta.BlockID, meta.TenantID), bIndex)

	return r.nextWriter.Write(ctx, meta, bBloom, bIndex, objectFilePath)
}

func (r *readerWriter) WriteBlockMeta(ctx context.Context, tracker backend.AppendTracker, meta *encoding.BlockMeta, bBloom [][]byte, bIndex []byte) error {
	for i, b := range bBloom {
		r.set(ctx, bloomKey(meta.BlockID, meta.TenantID, i), b)
	}
	r.set(ctx, key(meta.BlockID, meta.TenantID), bIndex)

	return r.nextWriter.WriteBlockMeta(ctx, tracker, meta, bBloom, bIndex)
}

func (r *readerWriter) AppendObject(ctx context.Context, tracker backend.AppendTracker, meta *encoding.BlockMeta, bObject []byte) (backend.AppendTracker, error) {
	return r.nextWriter.AppendObject(ctx, tracker, meta, bObject)
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

func key(blockID uuid.UUID, tenantID string) string {
	return blockID.String() + ":" + tenantID + ":" + typeIndex
}

func bloomKey(blockID uuid.UUID, tenantID string, shardNum int) string {
	return blockID.String() + ":" + tenantID + typeBloom + strconv.Itoa(shardNum)
}
