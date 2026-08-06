package memcached

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v2"

	"github.com/grafana/tempo/pkg/cache"
)

func TestConfigAppliesDefaultsOnUnmarshal(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want cache.MemcachedClientConfig
	}{
		{
			name: "empty config gets defaults",
			yaml: `{}`,
			want: cache.MemcachedClientConfig{
				MaxIdleConns:                   100,
				MinIdleConnsHeadroomPercentage: -1,
				Timeout:                        100 * time.Millisecond,
				UpdateInterval:                 time.Minute,
			},
		},
		{
			name: "explicit values override defaults",
			yaml: `
host: memcached.example.com
max_idle_conns: 32
min_idle_conns_headroom_percentage: 50
timeout: 250ms
connect_timeout: 50ms
update_interval: 5m
`,
			want: cache.MemcachedClientConfig{
				Host:                           "memcached.example.com",
				MaxIdleConns:                   32,
				MinIdleConnsHeadroomPercentage: 50,
				Timeout:                        250 * time.Millisecond,
				ConnectTimeout:                 50 * time.Millisecond,
				UpdateInterval:                 5 * time.Minute,
			},
		},
		{
			name: "partial config keeps remaining defaults",
			yaml: `timeout: 500ms`,
			want: cache.MemcachedClientConfig{
				MaxIdleConns:                   100,
				MinIdleConnsHeadroomPercentage: -1,
				Timeout:                        500 * time.Millisecond,
				UpdateInterval:                 time.Minute,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg Config
			require.NoError(t, yaml.UnmarshalStrict([]byte(tt.yaml), &cfg))
			require.Equal(t, tt.want, cfg.ClientConfig)
		})
	}
}
