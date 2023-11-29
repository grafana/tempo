package cache

import (
	"errors"
	"fmt"
	"strings"

	"github.com/grafana/tempo/modules/cache/memcached"
	"github.com/grafana/tempo/modules/cache/redis"
	"github.com/grafana/tempo/pkg/cache"
)

type Config struct {
	Background *cache.BackgroundConfig `yaml:"background"`
	Caches     []CacheConfig           `yaml:"caches"`
}

type CacheConfig struct { // nolint: revive
	Role            []cache.Role      `yaml:"roles"`
	MemcachedConfig *memcached.Config `yaml:"memcached"`
	RedisConfig     *redis.Config     `yaml:"redis"`
}

// Validate validates the config.
func (cfg *Config) Validate() error {
	claimedRoles := map[cache.Role]struct{}{}
	allRoles := allRoles()

	for _, cacheCfg := range cfg.Caches {
		if cacheCfg.MemcachedConfig != nil && cacheCfg.RedisConfig != nil {
			return fmt.Errorf("cache config for role %s has both memcached and redis configs", cacheCfg.Role)
		}

		if cacheCfg.MemcachedConfig == nil && cacheCfg.RedisConfig == nil {
			return fmt.Errorf("cache config for role %s has neither memcached nor redis configs", cacheCfg.Role)
		}

		if len(cacheCfg.Role) == 0 {
			return errors.New("configured caches require a valid role")
		}

		// check that all roles are unique
		for _, role := range cacheCfg.Role {
			if _, ok := allRoles[role]; !ok {
				return fmt.Errorf("role %s is not a valid role", role)
			}

			if _, ok := claimedRoles[role]; ok {
				return fmt.Errorf("role %s is claimed by more than one cache", role)
			}

			claimedRoles[role] = struct{}{}
		}
	}

	return nil
}

// Name returns a string representation of the roles claimed by this cache.
func (cfg *CacheConfig) Name() string {
	stringRoles := make([]string, len(cfg.Role))
	for i, role := range cfg.Role {
		stringRoles[i] = string(role)
	}
	return strings.Join(stringRoles, "|")
}

func allRoles() map[cache.Role]struct{} {
	all := []cache.Role{
		cache.RoleBloom,
		cache.RoleParquetFooter,
		cache.RoleParquetColumnIdx,
		cache.RoleParquetOffsetIdx,
		cache.RoleTraceIDIdx,
	}

	roles := map[cache.Role]struct{}{}
	for _, role := range all {
		roles[role] = struct{}{}
	}

	return roles
}
