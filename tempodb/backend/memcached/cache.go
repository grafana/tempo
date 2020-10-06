package memcached

import (
	"context"
	"time"

	"github.com/cortexproject/cortex/pkg/chunk/cache"
	"github.com/go-kit/kit/log"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/google/uuid"
)

const (
	typeBloom = "bloom"
	typeIndex = "index"
)

type Config struct {
	ClientConfig cache.MemcachedClientConfig `yaml:",inline"`

	TTL        time.Duration `yaml:"ttl"`
	WritesOnly bool          `yaml:"writes_only"`
}

type readerWriter struct {
	nextReader backend.Reader
	nextWriter backend.Writer
	client     *cache.Memcached
	logger     log.Logger
	writesOnly bool
}

func New(nextReader backend.Reader, nextWriter backend.Writer, cfg *Config, logger log.Logger) (backend.Reader, backend.Writer, error) {
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
		writesOnly: cfg.WritesOnly,
	}

	return rw, rw, nil
}

// Reader
func (r *readerWriter) Tenants() ([]string, error) {
	return r.nextReader.Tenants()
}

func (r *readerWriter) Blocks(tenantID string) ([]uuid.UUID, error) {
	return r.nextReader.Blocks(tenantID)
}

func (r *readerWriter) BlockMeta(blockID uuid.UUID, tenantID string) (*encoding.BlockMeta, error) {
	return r.nextReader.BlockMeta(blockID, tenantID)
}

func (r *readerWriter) Bloom(blockID uuid.UUID, tenantID string) ([]byte, error) {
	key := key(blockID, tenantID, typeBloom)
	val := r.get(key)
	if val != nil {
		return val, nil
	}

	val, err := r.nextReader.Bloom(blockID, tenantID)
	if err != nil {
		r.set(context.Background(), key, val)
	}

	return val, err
}

func (r *readerWriter) Index(blockID uuid.UUID, tenantID string) ([]byte, error) {
	key := key(blockID, tenantID, typeIndex)
	val := r.get(key)
	if val != nil {
		return val, nil
	}

	val, err := r.nextReader.Index(blockID, tenantID)
	if err != nil {
		r.set(context.Background(), key, val)
	}

	return val, err
}

func (r *readerWriter) Object(blockID uuid.UUID, tenantID string, start uint64, buffer []byte) error {
	return r.nextReader.Object(blockID, tenantID, start, buffer)
}

func (r *readerWriter) Shutdown() {
	r.nextReader.Shutdown()
	r.client.Stop()
}

// Writer
func (r *readerWriter) Write(ctx context.Context, meta *encoding.BlockMeta, bBloom []byte, bIndex []byte, objectFilePath string) error {
	r.set(ctx, key(meta.BlockID, meta.TenantID, typeBloom), bBloom)
	r.set(ctx, key(meta.BlockID, meta.TenantID, typeIndex), bIndex)

	return r.nextWriter.Write(ctx, meta, bBloom, bIndex, objectFilePath)
}

func (r *readerWriter) WriteBlockMeta(ctx context.Context, tracker backend.AppendTracker, meta *encoding.BlockMeta, bBloom []byte, bIndex []byte) error {
	r.set(ctx, key(meta.BlockID, meta.TenantID, typeBloom), bBloom)
	r.set(ctx, key(meta.BlockID, meta.TenantID, typeIndex), bIndex)

	return r.nextWriter.WriteBlockMeta(ctx, tracker, meta, bBloom, bIndex)
}

func (r *readerWriter) AppendObject(ctx context.Context, tracker backend.AppendTracker, meta *encoding.BlockMeta, bObject []byte) (backend.AppendTracker, error) {
	return r.nextWriter.AppendObject(ctx, tracker, meta, bObject)
}

func (r *readerWriter) get(key string) []byte {
	found, vals, _ := r.client.Fetch(context.Background(), []string{key})
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
