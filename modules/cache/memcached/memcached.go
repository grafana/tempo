package memcached

import (
	"flag"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/grafana/tempo/pkg/cache"
)

type Config struct {
	ClientConfig cache.MemcachedClientConfig `yaml:",inline"`

	TTL time.Duration `yaml:"ttl"`
}

// RegisterFlagsAndApplyDefaults applies the default values for this config.
func (cfg *Config) RegisterFlagsAndApplyDefaults(_ string, _ *flag.FlagSet) {
	// Size the idle connection pool for burst concurrency: idle connections are kept
	// per memcached server, and a too-small pool forces a storm of new dials at the
	// start of every request burst.
	cfg.ClientConfig.MaxIdleConns = 100
	// Never reap idle connections by default: keeping connection pools warm between
	// request bursts avoids dial storms and the tail latency they cause. Set a
	// positive headroom percentage to re-enable idle connection reaping.
	cfg.ClientConfig.MinIdleConnsHeadroomPercentage = -1
	cfg.ClientConfig.Timeout = 100 * time.Millisecond
	cfg.ClientConfig.UpdateInterval = time.Minute
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (cfg *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	cfg.RegisterFlagsAndApplyDefaults("", nil)

	type rawConfig Config
	return unmarshal((*rawConfig)(cfg))
}

func NewClient(cfg *Config, cfgBackground *cache.BackgroundConfig, name string, logger log.Logger) cache.Cache {
	client := cache.NewMemcachedClient(cfg.ClientConfig, name, prometheus.DefaultRegisterer, logger)
	memcachedCfg := cache.MemcachedConfig{
		Expiration: cfg.TTL,
	}
	c := cache.NewMemcached(memcachedCfg, client, name, cfg.ClientConfig.MaxItemSize, prometheus.DefaultRegisterer, logger)

	return cache.NewBackground(name, *cfgBackground, c, prometheus.DefaultRegisterer)
}
