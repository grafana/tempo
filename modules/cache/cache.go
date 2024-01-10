package cache

import (
	"context"
	"fmt"

	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/modules/cache/memcached"
	"github.com/grafana/tempo/modules/cache/redis"
	"github.com/grafana/tempo/pkg/cache"
	"github.com/grafana/tempo/pkg/usagestats"
	"github.com/prometheus/statsd_exporter/pkg/level"

	"github.com/go-kit/log"
)

var (
	statMemcached = usagestats.NewInt("cache_memcached")
	statRedis     = usagestats.NewInt("cache_redis")
)

type provider struct {
	services.Service

	caches map[cache.Role]cache.Cache
}

// NewProvider creates a new cache provider with the given config.
func NewProvider(cfg *Config, logger log.Logger) (cache.Provider, error) {
	p := &provider{
		caches: map[cache.Role]cache.Cache{},
	}

	err := cfg.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid cache config: %w", err)
	}

	statMemcached.Set(0)
	statRedis.Set(0)

	for _, cacheCfg := range cfg.Caches {
		var c cache.Cache

		if cacheCfg.MemcachedConfig != nil {
			level.Info(logger).Log("msg", "configuring memcached client", "roles", cacheCfg.Name())

			statMemcached.Add(1)
			c = memcached.NewClient(cacheCfg.MemcachedConfig, cfg.Background, cacheCfg.Name(), logger)
		}

		if cacheCfg.RedisConfig != nil {
			level.Info(logger).Log("msg", "configuring redis client", "roles", cacheCfg.Name())

			statRedis.Add(1)
			c = redis.NewClient(cacheCfg.RedisConfig, cfg.Background, cacheCfg.Name(), logger)
		}

		// add this cache for all claimed roles
		for _, role := range cacheCfg.Role {
			p.caches[role] = c
		}
	}

	p.Service = services.NewIdleService(p.starting, p.stopping)
	return p, nil
}

// CacheFor is used to retrieve a cache for a given role.
func (p *provider) CacheFor(role cache.Role) cache.Cache {
	return p.caches[role]
}

// AddCache is used to add a cache for a given role. It currently
// only exists to add the legacy cache in tempodb. It should
// likely be removed in the future.
func (p *provider) AddCache(role cache.Role, c cache.Cache) error {
	if _, ok := p.caches[role]; ok {
		return fmt.Errorf("cache for role %s already exists", role)
	}

	p.caches[role] = c

	return nil
}

func (p *provider) starting(_ context.Context) error {
	return nil
}

func (p *provider) stopping(_ error) error {
	// we can only stop a cache once (or they panic). use this map
	// to track which caches we've stopped.
	stopped := map[cache.Cache]struct{}{}

	for _, c := range p.caches {
		if _, ok := stopped[c]; ok {
			continue
		}

		stopped[c] = struct{}{}
		c.Stop()
	}

	return nil
}
