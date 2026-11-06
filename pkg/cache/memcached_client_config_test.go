package cache

import (
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestNewMemcachedClientAppliesConfig(t *testing.T) {
	tests := []struct {
		name               string
		cfg                MemcachedClientConfig
		wantTimeout        time.Duration
		wantConnectTimeout time.Duration
		wantMaxIdleConns   int
		wantHeadroom       float64
		wantRecentlyUsed   time.Duration
	}{
		{
			name: "connect timeout falls back to timeout",
			cfg: MemcachedClientConfig{
				Timeout:        200 * time.Millisecond,
				MaxIdleConns:   16,
				UpdateInterval: time.Minute,
			},
			wantTimeout:        200 * time.Millisecond,
			wantConnectTimeout: 200 * time.Millisecond,
			wantMaxIdleConns:   16,
			wantHeadroom:       0,
		},
		{
			name: "explicit values applied",
			cfg: MemcachedClientConfig{
				Timeout:                        100 * time.Millisecond,
				ConnectTimeout:                 50 * time.Millisecond,
				MaxIdleConns:                   42,
				MinIdleConnsHeadroomPercentage: -1,
				RecentlyUsedConnsThreshold:     10 * time.Minute,
				UpdateInterval:                 time.Minute,
			},
			wantTimeout:        100 * time.Millisecond,
			wantConnectTimeout: 50 * time.Millisecond,
			wantMaxIdleConns:   42,
			wantHeadroom:       -1,
			wantRecentlyUsed:   10 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewMemcachedClient(tt.cfg, "test", prometheus.NewRegistry(), log.NewNopLogger())

			mc, ok := c.(*memcachedClient)
			require.True(t, ok)
			defer mc.Stop()
			defer mc.Close()

			require.Equal(t, tt.wantTimeout, mc.Timeout)
			require.Equal(t, tt.wantConnectTimeout, mc.ConnectTimeout)
			require.Equal(t, tt.wantMaxIdleConns, mc.MaxIdleConns)
			require.Equal(t, tt.wantHeadroom, mc.MinIdleConnsHeadroomPercentage)
			require.Equal(t, tt.wantRecentlyUsed, mc.RecentlyUsedConnsThreshold)
		})
	}
}
