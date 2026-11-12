package cache

import (
	"errors"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

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
		{"value read timeout", fmt.Errorf("memcache: corrupt get result: %w", &net.OpError{Op: "read", Err: os.ErrDeadlineExceeded}), "timeout_value"},
		{"value read non-timeout", fmt.Errorf("memcache: corrupt get result: %w", errors.New("unexpected EOF")), "other"},
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

// TestMemcacheErrorReasonFromClient drives the vendored client against servers that stall, so a client
// update that changes where a timeout is reported fails here instead of silently moving errors between reasons.
func TestMemcacheErrorReasonFromClient(t *testing.T) {
	serve := func(t *testing.T, handle func(c net.Conn)) string {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		t.Cleanup(func() { _ = l.Close() })
		go func() {
			for {
				c, err := l.Accept()
				if err != nil {
					return
				}
				go func() {
					defer c.Close()
					buf := make([]byte, 256)
					_, _ = c.Read(buf) // the get request
					handle(c)
				}()
			}
		}()
		return l.Addr().String()
	}
	get := func(t *testing.T, addr string) error {
		c := memcache.New(addr)
		c.Timeout = 100 * time.Millisecond
		_, err := c.Get("key")
		require.Error(t, err)
		return err
	}

	t.Run("no response", func(t *testing.T) {
		addr := serve(t, func(net.Conn) { time.Sleep(time.Second) })
		require.Equal(t, "timeout", memcacheErrorReason(get(t, addr)))
	})

	t.Run("response stalls in the value", func(t *testing.T) {
		addr := serve(t, func(c net.Conn) {
			_, _ = c.Write([]byte("VALUE key 0 1000\r\n0123456789"))
			time.Sleep(time.Second)
		})
		require.Equal(t, "timeout_value", memcacheErrorReason(get(t, addr)))
	})
}
