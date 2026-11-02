package cache

import (
	"errors"
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/go-kit/log"
	"github.com/grafana/gomemcache/memcache"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sony/gobreaker"
	"github.com/stretchr/testify/require"
)

func TestMemcacheErrorReason(t *testing.T) {
	addr := &net.TCPAddr{IP: net.IPv4(10, 0, 0, 1), Port: 11311}

	tcs := []struct {
		name     string
		err      error
		expected string
	}{
		{"breaker open", fmt.Errorf("memcache GetMulti: %w", gobreaker.ErrOpenState), "circuit_breaker_open"},
		{"breaker half open", gobreaker.ErrTooManyRequests, "circuit_breaker_open"},
		{"read timeout", &net.OpError{Op: "read", Err: os.ErrDeadlineExceeded}, "timeout"},
		{"wrapped deadline", fmt.Errorf("memcache GetMulti: %w", os.ErrDeadlineExceeded), "timeout"},
		{"connect timeout", &memcache.ConnectTimeoutError{Addr: addr}, "timeout"},
		{"server error", fmt.Errorf("%w: out of memory storing object", memcache.ErrServerError), "other"},
		{"other", errors.New("boom"), "other"},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, memcacheErrorReason(tc.err))
		})
	}
}

func TestMemcachedCountError(t *testing.T) {
	c := NewMemcached(MemcachedConfig{}, nil, "test", 0, prometheus.NewRegistry(), log.NewNopLogger())

	c.countError("Memcache.Get", nil)
	c.countError("Memcache.Get", memcache.ErrCacheMiss)
	c.countError("Memcache.Get", memcache.ErrMalformedKey)
	c.countError("Memcache.Get", os.ErrDeadlineExceeded)
	c.countError("Memcache.Get", os.ErrDeadlineExceeded)
	c.countError("Memcache.Put", gobreaker.ErrOpenState)

	require.Equal(t, 2.0, testutil.ToFloat64(c.requestErrors.WithLabelValues("Memcache.Get", "timeout")))
	require.Equal(t, 1.0, testutil.ToFloat64(c.requestErrors.WithLabelValues("Memcache.Put", "circuit_breaker_open")))
	require.Equal(t, 0.0, testutil.ToFloat64(c.requestErrors.WithLabelValues("Memcache.Get", "other")))
}
