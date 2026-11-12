package cache

import (
	"fmt"
	"net"
	"testing"

	"github.com/grafana/gomemcache/memcache"
	"github.com/stretchr/testify/require"
)

func pinnedAddresses(t *testing.T, selector *memcachedPinnedSelector) []string {
	t.Helper()
	var addresses []string
	require.NoError(t, selector.Each(func(addr net.Addr) error {
		addresses = append(addresses, addr.String())
		return nil
	}))
	return addresses
}

func TestCutServers(t *testing.T) {
	a := net.Addr(&staticAddr{network: "tcp", str: "a"})
	b := net.Addr(&staticAddr{network: "tcp", str: "b"})
	c := net.Addr(&staticAddr{network: "tcp", str: "c"})
	d := net.Addr(&staticAddr{network: "tcp", str: "d"})
	servers := []net.Addr{a, b, c, d}
	for _, tc := range []struct {
		name  string
		addrs []net.Addr
		start int
		count uint
		want  []net.Addr
	}{
		{"first", servers, 0, 3, []net.Addr{a, b, c}},
		{"second", servers, 1, 3, []net.Addr{b, c, d}},
		{"third wraps", servers, 2, 3, []net.Addr{c, d, a}},
		{"fourth wraps", servers, 3, 3, []net.Addr{d, a, b}},
		{"exact pool size", servers, 2, 4, []net.Addr{c, d, a, b}},
		{"requested three, discovered two", servers[:2], 1, 3, []net.Addr{a, b}},
		{"no servers", nil, 0, 3, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, cutServers(tc.addrs, tc.start, tc.count))
		})
	}
}

func TestMemcachedPinnedSelectorClampsAndRefreshes(t *testing.T) {
	selector := newMemcachedPinnedSelector("querier-3", 3)
	_, err := selector.PickServer("key")
	require.ErrorIs(t, err, memcache.ErrNoServers)
	require.Empty(t, pinnedAddresses(t, selector))

	servers := []string{"127.0.0.1:11211", "127.0.0.2:11211"}
	require.NoError(t, selector.SetServers(servers[1], servers[0]))
	require.Equal(t, servers, pinnedAddresses(t, selector))

	// discovery failure preserves the last good subset.
	require.Error(t, selector.SetServers("127.0.0.1:invalid-port"))
	require.Equal(t, servers, pinnedAddresses(t, selector))

	// only one server, so all requests should go to it.
	require.NoError(t, selector.SetServers(servers[1]))
	for range 5 {
		addr, err := selector.PickServer("key")
		require.NoError(t, err)
		require.Equal(t, servers[1], addr.String())
	}

	// clean
	require.NoError(t, selector.SetServers())
	_, err = selector.PickServer("key")
	require.ErrorIs(t, err, memcache.ErrNoServers)
	require.Empty(t, pinnedAddresses(t, selector))
}

func TestMemcachedPinnedSelector(t *testing.T) {
	count := 2
	selector := newMemcachedPinnedSelector("querier-3", uint(count))
	servers := []string{"127.0.0.1:11211", "127.0.0.2:11211", "127.0.0.3:11211", "127.0.0.4:11211"}
	require.NoError(t, selector.SetServers(servers...))
	selected := pinnedAddresses(t, selector)
	require.Len(t, selected, count)

	total := 3000
	counter := map[string]int{}
	for i := range total {
		key := fmt.Sprintf("probe-%02d", i)
		selectedAddr, err := selector.PickServer(key)
		require.NoError(t, err)
		counter[selectedAddr.String()]++
	}
	require.Len(t, counter, count)
	for _, v := range counter {
		// chance of false positive is 1 in 3 million runs
		require.InDelta(t, total/count, v, float64(v)*0.1)
	}
}

func TestMemcachedPinnedSelectorHostName(t *testing.T) {
	servers := []string{"127.0.0.1:11211", "127.0.0.2:11211", "127.0.0.3:11211", "127.0.0.4:11211"}
	total := 3000
	counter := map[string]int{}
	for i := range total {
		hostname := fmt.Sprintf("querier-%d", i)
		selector := newMemcachedPinnedSelector(hostname, 1)
		require.NoError(t, selector.SetServers(servers...))
		selected := pinnedAddresses(t, selector)
		require.Len(t, selected, 1)
		counter[selected[0]]++
	}
	require.Len(t, counter, len(servers))
	for _, v := range counter {
		// chance of false positive is 1 in 7 million runs
		require.InDelta(t, total/len(servers), v, float64(v)*0.2)
	}
}
