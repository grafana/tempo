package redis

import (
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/grafana/tempo/pkg/cache"
)

type Config struct {
	ClientConfig cache.RedisConfig `yaml:",inline"`

	TTL time.Duration `yaml:"ttl"`
}

func NewClient(cfg *Config, cfgBackground *cache.BackgroundConfig, name string, logger log.Logger) cache.Cache {
	if cfg.ClientConfig.Timeout == 0 {
		cfg.ClientConfig.Timeout = 100 * time.Millisecond
	}
	if cfg.ClientConfig.Expiration == 0 {
		cfg.ClientConfig.Expiration = cfg.TTL
	}

	client := cache.NewRedisClient(&cfg.ClientConfig)
	c := cache.NewRedisCache(name, client, prometheus.DefaultRegisterer, logger)

	return cache.NewBackground(name, *cfgBackground, c, prometheus.DefaultRegisterer)
}
