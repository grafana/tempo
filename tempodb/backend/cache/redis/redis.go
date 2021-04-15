package redis

import (
	"time"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"

	cortex_cache "github.com/cortexproject/cortex/pkg/chunk/cache"
)

type Config struct {
	ClientConfig cortex_cache.RedisConfig `yaml:",inline"`

	TTL time.Duration `yaml:"ttl"`
}

func NewClient(cfg *Config, cfgBackground *cortex_cache.BackgroundConfig, logger log.Logger) cortex_cache.Cache {
	if cfg.ClientConfig.Timeout == 0 {
		cfg.ClientConfig.Timeout = 100 * time.Millisecond
	}
	if cfg.ClientConfig.Expiration == 0 {
		cfg.ClientConfig.Expiration = cfg.TTL
	}

	client := cortex_cache.NewRedisClient(&cfg.ClientConfig)
	cache := cortex_cache.NewRedisCache("tempo", client, logger)

	return cortex_cache.NewBackground("tempo", *cfgBackground, cache, prometheus.DefaultRegisterer)
}
