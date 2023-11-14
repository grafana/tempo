package memcached

import (
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/grafana/tempo/pkg/cache"
)

type Config struct {
	ClientConfig cache.MemcachedClientConfig `yaml:",inline"`

	TTL time.Duration `yaml:"ttl"`
}

func NewClient(cfg *Config, cfgBackground *cache.BackgroundConfig, name string, logger log.Logger) cache.Cache {
	if cfg.ClientConfig.MaxIdleConns == 0 {
		cfg.ClientConfig.MaxIdleConns = 16
	}
	if cfg.ClientConfig.Timeout == 0 {
		cfg.ClientConfig.Timeout = 100 * time.Millisecond
	}
	if cfg.ClientConfig.UpdateInterval == 0 {
		cfg.ClientConfig.UpdateInterval = time.Minute
	}

	client := cache.NewMemcachedClient(cfg.ClientConfig, name, prometheus.DefaultRegisterer, logger)
	memcachedCfg := cache.MemcachedConfig{
		Expiration:  cfg.TTL,
		BatchSize:   0, // we are currently only requesting one key at a time, which is bad.  we could restructure Find() to batch request all blooms at once
		Parallelism: 0,
	}
	c := cache.NewMemcached(memcachedCfg, client, name, prometheus.DefaultRegisterer, logger)

	return cache.NewBackground(name, *cfgBackground, c, prometheus.DefaultRegisterer)
}
