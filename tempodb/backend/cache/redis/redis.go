package redis

import (
	"context"
	"time"

	"github.com/go-kit/kit/log"

	cortex_cache "github.com/cortexproject/cortex/pkg/chunk/cache"
	"github.com/grafana/tempo/tempodb/backend/cache"
)

type Config struct {
	ClientConfig cortex_cache.RedisConfig `yaml:",inline"`

	TTL time.Duration `yaml:"ttl"`
}

type Client struct {
	client *cortex_cache.RedisCache
}

func NewCache(cfg *Config, logger log.Logger) cache.Client {
	if cfg.ClientConfig.Timeout == 0 {
		cfg.ClientConfig.Timeout = 100 * time.Millisecond
	}
	if cfg.ClientConfig.Expiration == 0 {
		cfg.ClientConfig.Expiration = cfg.TTL
	}

	client := cortex_cache.NewRedisClient(&cfg.ClientConfig)
	return &Client{
		client: cortex_cache.NewRedisCache("tempo", client, logger),
	}
}

// Store implements cache.Store
func (r *Client) Store(ctx context.Context, key string, val []byte) {
	r.client.Store(ctx, []string{key}, [][]byte{val})
}

// Fetch implements cache.Fetch
func (r *Client) Fetch(ctx context.Context, key string) []byte {
	found, vals, _ := r.client.Fetch(ctx, []string{key})
	if len(found) > 0 {
		return vals[0]
	}
	return nil
}

// Shutdown implements cache.Shutdown
func (r *Client) Shutdown() {
	r.client.Stop()
}
