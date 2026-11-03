package cache

import (
	"errors"
	"net"
	"testing"
	"testing/synctest"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestMemcachedRefreshesOnOpenCircuitBreakerWithCooldown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const first = "127.0.0.1:11211"
		const second = "127.0.0.2:11211"
		const third = "127.0.0.3:11211"

		c := NewMemcachedClient(MemcachedClientConfig{
			Addresses:      first,
			ConsistentHash: true,
			UpdateInterval: time.Hour,
			CBFailures:     1,
		}, "refresh-test", prometheus.NewRegistry(), log.NewNopLogger()).(*memcachedClient)
		defer c.Stop()
		defer c.Close()

		c.DialTimeout = func(string, string, time.Duration) (net.Conn, error) {
			return nil, errors.New("dial failed")
		}
		selected := func() string {
			addr, err := c.serverList.PickServer("key")
			require.NoError(t, err)
			return addr.String()
		}
		openBreaker := func(address string) {
			for range 2 {
				_, err := c.dialViaCircuitBreaker("tcp", address, 0)
				require.Error(t, err)
			}
		}

		synctest.Wait() // Ensure the update loop has initialized its clock and ticker.
		require.Equal(t, first, selected())
		// The initial discovery counts as a refresh for the cooldown.
		c.addresses = []string{second}
		time.Sleep(time.Second)
		openBreaker(first)
		synctest.Wait()
		require.Equal(t, second, selected())

		// Another breaker opening during the cooldown does not force a lookup.
		c.addresses = []string{third}
		openBreaker(second)
		synctest.Wait()
		require.Equal(t, second, selected())
		time.Sleep(time.Second)
		synctest.Wait()
		require.Equal(t, second, selected()) // Suppressed refreshes are not deferred.

		// Once the cooldown expires, a new open transition refreshes the selector.
		openBreaker(third)
		synctest.Wait()
		require.Equal(t, third, selected())
	})
}
