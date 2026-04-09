package cache

import (
	"errors"
	"testing"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/v2/modules/cache/memcached"
	"github.com/grafana/tempo/v2/modules/cache/redis"
	"github.com/grafana/tempo/v2/pkg/cache"
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

func TestMaxItemSize(t *testing.T) {
	tcs := []struct {
		cfg      *CacheConfig
		expected int
	}{
		{
			cfg: &CacheConfig{
				Role: []cache.Role{cache.RoleBloom, cache.RoleParquetColumnIdx},
				MemcachedConfig: &memcached.Config{
					ClientConfig: cache.MemcachedClientConfig{
						MaxItemSize: 123,
					},
				},
			},
			expected: 123,
		},
		{
			cfg: &CacheConfig{
				Role:        []cache.Role{cache.RoleBloom, cache.RoleFrontendSearch},
				RedisConfig: &redis.Config{},
			},
			expected: 0, // redis does not support max item size
		},
	}

	for _, tc := range tcs {
		t.Run("", func(t *testing.T) {
			p, err := NewProvider(&Config{
				Caches:     []CacheConfig{*tc.cfg},
				Background: &cache.BackgroundConfig{},
			}, log.NewNopLogger())
			require.NoError(t, err)

			cache := p.CacheFor(cache.RoleBloom)
			require.Equal(t, tc.expected, cache.MaxItemSize())
		})
	}
}
