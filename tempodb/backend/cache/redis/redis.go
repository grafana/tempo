package redis

import (
	"context"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"

	cortex_cache "github.com/cortexproject/cortex/pkg/chunk/cache"
	"github.com/grafana/tempo/tempodb/backend/cache"
)

type Config struct {
	ClientConfig cortex_cache.RedisConfig `yaml:",inline"`

	TTL time.Duration `yaml:"ttl"`
}

type Client struct {
	client cortex_cache.Cache
}

func NewClient(cfg *Config, logger log.Logger) cache.Client {
	if cfg.ClientConfig.Timeout == 0 {
		cfg.ClientConfig.Timeout = 100 * time.Millisecond
	}
	if cfg.ClientConfig.Expiration == 0 {
		cfg.ClientConfig.Expiration = cfg.TTL
	}

	client := cortex_cache.NewRedisClient(&cfg.ClientConfig)
	cache := cortex_cache.NewRedisCache("tempo", client, logger)

	return &Client{
		client: cortex_cache.NewBackground("tempo", cortex_cache.BackgroundConfig{
			WriteBackGoroutines: 10,
			WriteBackBuffer:     10000,
		}, cache, prometheus.DefaultRegisterer),
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

// Stop implements cache.Stop
func (r *Client) Stop() {
	r.client.Stop()
}
