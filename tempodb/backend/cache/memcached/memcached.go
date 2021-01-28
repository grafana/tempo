package memcached

import (
	"context"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"

	cortex_cache "github.com/cortexproject/cortex/pkg/chunk/cache"
	"github.com/grafana/tempo/tempodb/backend/cache"
)

type Config struct {
	ClientConfig cortex_cache.MemcachedClientConfig `yaml:",inline"`

	TTL time.Duration `yaml:"ttl"`
}

type Client struct {
	client *cortex_cache.Memcached
}

func NewCache(cfg *Config, logger log.Logger) cache.Client {
	if cfg.ClientConfig.MaxIdleConns == 0 {
		cfg.ClientConfig.MaxIdleConns = 16
	}
	if cfg.ClientConfig.Timeout == 0 {
		cfg.ClientConfig.Timeout = 100 * time.Millisecond
	}
	if cfg.ClientConfig.UpdateInterval == 0 {
		cfg.ClientConfig.UpdateInterval = time.Minute
	}

	client := cortex_cache.NewMemcachedClient(cfg.ClientConfig, "tempo", prometheus.DefaultRegisterer, logger)
	memcachedCfg := cortex_cache.MemcachedConfig{
		Expiration:  cfg.TTL,
		BatchSize:   0, // we are currently only requesting one key at a time, which is bad.  we could restructure Find() to batch request all blooms at once
		Parallelism: 0,
	}
	return &Client{
		client: cortex_cache.NewMemcached(memcachedCfg, client, "tempo", prometheus.DefaultRegisterer, logger),
	}
}

// Store implements cache.Store
func (m *Client) Store(ctx context.Context, key string, val []byte) {
	m.client.Store(ctx, []string{key}, [][]byte{val})
}

// Fetch implements cache.Fetch
func (m *Client) Fetch(ctx context.Context, key string) []byte {
	found, vals, _ := m.client.Fetch(ctx, []string{key})
	if len(found) > 0 {
		return vals[0]
	}
	return nil
}

// Shutdown implements cache.Shutdown
func (m *Client) Shutdown() {
	m.client.Stop()
}
