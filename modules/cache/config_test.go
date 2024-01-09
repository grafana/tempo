package cache

import (
	"errors"
	"testing"

	"github.com/grafana/tempo/modules/cache/memcached"
	"github.com/grafana/tempo/modules/cache/redis"
	"github.com/grafana/tempo/pkg/cache"
	"github.com/stretchr/testify/require"
)

func TestConfigValidation(t *testing.T) {
	tcs := []struct {
		name     string
		cfg      *Config
		expected error
	}{
		{
			name: "no caching is valid",
			cfg:  &Config{},
		},
		{
			name: "valid config",
			cfg: &Config{
				Caches: []CacheConfig{
					{
						Role:            []cache.Role{cache.RoleBloom},
						MemcachedConfig: &memcached.Config{},
					},
					{
						Role:        []cache.Role{cache.RoleParquetColumnIdx},
						RedisConfig: &redis.Config{},
					},
				},
			},
		},
		{
			name: "invalid - duplicate roles",
			cfg: &Config{
				Caches: []CacheConfig{
					{
						Role:            []cache.Role{cache.RoleBloom},
						MemcachedConfig: &memcached.Config{},
					},
					{
						Role:        []cache.Role{cache.RoleBloom},
						RedisConfig: &redis.Config{},
					},
				},
			},
			expected: errors.New("role bloom is claimed by more than one cache"),
		},
		{
			name: "invalid - no roles",
			cfg: &Config{
				Caches: []CacheConfig{
					{
						MemcachedConfig: &memcached.Config{},
					},
				},
			},
			expected: errors.New("configured caches require a valid role"),
		},
		{
			name: "invalid - none",
			cfg: &Config{
				Caches: []CacheConfig{
					{
						Role:            []cache.Role{cache.RoleNone},
						MemcachedConfig: &memcached.Config{},
					},
				},
			},
			expected: errors.New("role none is not a valid role"),
		},
		{
			name: "invalid - both caches configged",
			cfg: &Config{
				Caches: []CacheConfig{
					{
						Role:            []cache.Role{cache.RoleBloom},
						MemcachedConfig: &memcached.Config{},
						RedisConfig:     &redis.Config{},
					},
				},
			},
			expected: errors.New("cache config for role [bloom] has both memcached and redis configs"),
		},
		{
			name: "invalid - no caches configged",
			cfg: &Config{
				Caches: []CacheConfig{
					{
						Role: []cache.Role{cache.RoleBloom},
					},
				},
			},
			expected: errors.New("cache config for role [bloom] has neither memcached nor redis configs"),
		},
		{
			name: "invalid - non-existent role",
			cfg: &Config{
				Caches: []CacheConfig{
					{
						Role:            []cache.Role{cache.Role("foo")},
						MemcachedConfig: &memcached.Config{},
					},
				},
			},
			expected: errors.New("role foo is not a valid role"),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			require.Equal(t, tc.expected, err)
		})
	}
}

func TestCacheConfigName(t *testing.T) {
	tcs := []struct {
		cfg      *CacheConfig
		expected string
	}{
		{
			cfg: &CacheConfig{
				Role: []cache.Role{cache.RoleBloom},
			},
			expected: "bloom",
		},
		{
			cfg: &CacheConfig{
				Role: []cache.Role{cache.RoleBloom, cache.RoleParquetColumnIdx},
			},
			expected: "bloom|parquet-column-idx",
		},
		{
			cfg:      &CacheConfig{},
			expected: "",
		},
	}

	for _, tc := range tcs {
		actual := tc.cfg.Name()
		require.Equal(t, tc.expected, actual)
	}
}
