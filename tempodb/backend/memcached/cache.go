package memcached

import (
	"context"
	"strconv"
	"time"

	"github.com/cortexproject/cortex/pkg/chunk/cache"
	"github.com/go-kit/kit/log"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/google/uuid"
)

const (
	typeBloom = "bloom"
	typeIndex = "index"
)

type Config struct {
	ClientConfig cache.MemcachedClientConfig `yaml:",inline"`

	TTL time.Duration `yaml:"ttl"`
}

type readerWriter struct {
	nextReader backend.Reader
	nextWriter backend.Writer
	client     *cache.Memcached
	logger     log.Logger
}

func New(nextReader backend.Reader, nextWriter backend.Writer, cfg *Config, logger log.Logger) (backend.Reader, backend.Writer, error) {
	if cfg.ClientConfig.MaxIdleConns == 0 {
		cfg.ClientConfig.MaxIdleConns = 16
	}
	if cfg.ClientConfig.Timeout == 0 {
		cfg.ClientConfig.Timeout = 100 * time.Millisecond
	}
	if cfg.ClientConfig.UpdateInterval == 0 {
		cfg.ClientConfig.UpdateInterval = time.Minute
	}

	client := cache.NewMemcachedClient(cfg.ClientConfig, "tempo", prometheus.DefaultRegisterer, logger)
	memcachedCfg := cache.MemcachedConfig{
		Expiration:  cfg.TTL,
		BatchSize:   0, // we are currently only requesting one key at a time, which is bad.  we could restructure Find() to batch request all blooms at once
		Parallelism: 0,
	}

	rw := &readerWriter{
		client:     cache.NewMemcached(memcachedCfg, client, "tempo", prometheus.DefaultRegisterer, logger),
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

func (r *readerWriter) BlockMeta(ctx context.Context, blockID uuid.UUID, tenantID string) (*backend.BlockMeta, error) {
	return r.nextReader.BlockMeta(ctx, blockID, tenantID)
}

func (r *readerWriter) Bloom(ctx context.Context, blockID uuid.UUID, tenantID string, shardNum int) ([]byte, error) {
	key := bloomKey(blockID, tenantID, typeBloom, shardNum)
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
	key := key(blockID, tenantID, typeIndex)
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
func (r *readerWriter) Write(ctx context.Context, meta *backend.BlockMeta, bBloom [][]byte, bIndex []byte, objectFilePath string) error {
	for i, b := range bBloom {
		r.set(ctx, bloomKey(meta.BlockID, meta.TenantID, typeBloom, i), b)
	}
	r.set(ctx, key(meta.BlockID, meta.TenantID, typeIndex), bIndex)

	return r.nextWriter.Write(ctx, meta, bBloom, bIndex, objectFilePath)
}

func (r *readerWriter) WriteBlockMeta(ctx context.Context, tracker backend.AppendTracker, meta *backend.BlockMeta, bBloom [][]byte, bIndex []byte) error {
	for i, b := range bBloom {
		r.set(ctx, bloomKey(meta.BlockID, meta.TenantID, typeBloom, i), b)
	}
	r.set(ctx, key(meta.BlockID, meta.TenantID, typeIndex), bIndex)

	return r.nextWriter.WriteBlockMeta(ctx, tracker, meta, bBloom, bIndex)
}

func (r *readerWriter) AppendObject(ctx context.Context, tracker backend.AppendTracker, meta *backend.BlockMeta, bObject []byte) (backend.AppendTracker, error) {
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

func key(blockID uuid.UUID, tenantID string, t string) string {
	return blockID.String() + ":" + tenantID + ":" + t
}

func bloomKey(blockID uuid.UUID, tenantID string, t string, shardNum int) string {
	return blockID.String() + ":" + tenantID + ":" + t + strconv.Itoa(shardNum)
}
