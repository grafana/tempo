package memcached

import (
	"time"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"

	cortex_cache "github.com/cortexproject/cortex/pkg/chunk/cache"
)

type Config struct {
	ClientConfig cortex_cache.MemcachedClientConfig `yaml:",inline"`

	TTL time.Duration `yaml:"ttl"`
}

func NewClient(cfg *Config, cfgBackground *cortex_cache.BackgroundConfig, logger log.Logger) cortex_cache.Cache {
	if cfg.ClientConfig.MaxIdleConns == 0 {
		cfg.ClientConfig.MaxIdleConns = 16
	}
	if cfg.ClientConfig.Timeout == 0 {
		cfg.ClientConfig.Timeout = 100 * time.Millisecond
	}
	if cfg.ClientConfig.UpdateInterval == 0 {
		cfg.ClientConfig.UpdateInterval = time.Minute
	}
	if cfgBackground == nil {
		cfgBackground = &cortex_cache.BackgroundConfig{
			WriteBackGoroutines: 10,
			WriteBackBuffer:     1000,
		}
	}

	client := cortex_cache.NewMemcachedClient(cfg.ClientConfig, "tempo", prometheus.DefaultRegisterer, logger)
	memcachedCfg := cortex_cache.MemcachedConfig{
		Expiration:  cfg.TTL,
		BatchSize:   0, // we are currently only requesting one key at a time, which is bad.  we could restructure Find() to batch request all blooms at once
		Parallelism: 0,
	}
	cache := cortex_cache.NewMemcached(memcachedCfg, client, "tempo", prometheus.DefaultRegisterer, logger)

	return cortex_cache.NewBackground("tempo", *cfgBackground, cache, prometheus.DefaultRegisterer)
}
