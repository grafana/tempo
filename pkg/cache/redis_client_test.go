package cache

import (
	"context"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestRedisClient(t *testing.T) {
	single, err := mockRedisClientSingle()
	require.Nil(t, err)
	defer single.Close()

	cluster, err := mockRedisClientCluster()
	require.Nil(t, err)
	defer cluster.Close()

	ctx := context.Background()

	tests := []struct {
		name   string
		client *RedisClient
	}{
		{
			name:   "single redis client",
			client: single,
		},
		{
			name:   "cluster redis client",
			client: cluster,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys := []string{"key1", "key2", "key3"}
			bufs := [][]byte{[]byte("data1"), []byte("data2"), []byte("data3")}
			miss := []string{"miss1", "miss2"}

			// set values
			err := tt.client.MSet(ctx, keys, bufs)
			require.Nil(t, err)

			// get keys
			values, err := tt.client.MGet(ctx, keys)
			require.Nil(t, err)
			require.Len(t, values, len(keys))
			for i, value := range values {
				require.Equal(t, values[i], value)
			}

			// get missing keys
			values, err = tt.client.MGet(ctx, miss)
			require.Nil(t, err)
			require.Len(t, values, len(miss))
			for _, value := range values {
				require.Nil(t, value)
			}

			// delete keys
			err = tt.client.Del(ctx, keys)
			require.Nil(t, err)

			// verify deleted
			values, err = tt.client.MGet(ctx, keys)
			require.Nil(t, err)
			require.Len(t, values, len(keys))
			for _, value := range values {
				require.Nil(t, value, "expected key to be absent after Del")
			}

			// delete empty slice should be a no-op
			err = tt.client.Del(ctx, []string{})
			require.Nil(t, err)

			// large multi-key Del cross-slot smoke test: in real clusters this
			// would touch multiple slots and must complete without CROSSSLOT.
			many := make([]string, 0, 50)
			manyVals := make([][]byte, 0, 50)
			for i := 0; i < 50; i++ {
				many = append(many, "many-"+strconv.Itoa(i))
				manyVals = append(manyVals, []byte(strconv.Itoa(i)))
			}
			require.Nil(t, tt.client.MSet(ctx, many, manyVals))
			require.Nil(t, tt.client.Del(ctx, many))
			gotMany, err := tt.client.MGet(ctx, many)
			require.Nil(t, err)
			for _, v := range gotMany {
				require.Nil(t, v)
			}
		})
	}
}

// TestRedisClient_MSet_Cluster_AllKeysReadable codifies the invariant that
// every key written through MSet must subsequently be readable via MGet on a
// cluster client. The original v8 implementation used TxPipeline (MULTI/EXEC),
// which is rejected with CROSSSLOT for multi-slot writes in a real cluster —
// this test guards against a regression on the simpler Pipeline path.
func TestRedisClient_MSet_Cluster_AllKeysReadable(t *testing.T) {
	cluster, err := mockRedisClientCluster()
	require.Nil(t, err)
	defer cluster.Close()

	ctx := context.Background()
	const n = 64
	keys := make([]string, n)
	vals := make([][]byte, n)
	for i := 0; i < n; i++ {
		keys[i] = "xs-key-" + strconv.Itoa(i)
		vals[i] = []byte("xs-val-" + strconv.Itoa(i))
	}

	require.NoError(t, cluster.MSet(ctx, keys, vals))

	got, err := cluster.MGet(ctx, keys)
	require.NoError(t, err)
	require.Len(t, got, n)
	for i := range keys {
		require.Equal(t, vals[i], got[i], "key %q must be readable after MSet", keys[i])
	}
}

func TestRedisClient_MSet_MaxItemSize(t *testing.T) {
	const itemSize = 100

	tests := []struct {
		name        string
		maxItemSize int
		keys        []string
		values      [][]byte
		wantStored  []bool // parallel to keys; true = expected present in redis
		wantSkipped int    // expected value of the skipped counter
	}{
		{
			name:        "mixed batch skips oversized item",
			maxItemSize: itemSize,
			keys:        []string{"small1", "oversized", "small2"},
			values:      [][]byte{[]byte("ok"), make([]byte, itemSize+1), []byte("also ok")},
			wantStored:  []bool{true, false, true},
			wantSkipped: 1,
		},
		{
			name:        "boundary item exactly at the cap is stored and one byte over is skipped",
			maxItemSize: itemSize,
			keys:        []string{"at-cap", "over-cap-by-1"},
			values:      [][]byte{make([]byte, itemSize), make([]byte, itemSize+1)},
			wantStored:  []bool{true, false},
			wantSkipped: 1,
		},
		{
			name:        "all oversized batch skips every item and counter increments per item",
			maxItemSize: itemSize,
			keys:        []string{"big1", "big2", "big3"},
			values:      [][]byte{make([]byte, itemSize+1), make([]byte, itemSize+50), make([]byte, itemSize*10)},
			wantStored:  []bool{false, false, false},
			wantSkipped: 3,
		},
		{
			name:        "zero MaxItemSize disables the cap",
			maxItemSize: 0,
			keys:        []string{"larger-than-cap"},
			values:      [][]byte{make([]byte, itemSize*100)},
			wantStored:  []bool{true},
			wantSkipped: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			redisServer, err := miniredis.Run()
			require.NoError(t, err)
			defer redisServer.Close()

			client, err := NewRedisClient(&RedisConfig{
				Expiration:  time.Minute,
				Timeout:     100 * time.Millisecond,
				Endpoint:    redisServer.Addr(),
				SingleNode:  true,
				MaxItemSize: tc.maxItemSize,
			}, "test", prometheus.NewRegistry())
			require.NoError(t, err)

			require.NoError(t, client.MSet(context.Background(), tc.keys, tc.values))

			got, err := client.MGet(context.Background(), tc.keys)
			require.NoError(t, err)

			for i, key := range tc.keys {
				if tc.wantStored[i] {
					require.Equal(t, tc.values[i], got[i], "key %q must be stored", key)
				} else {
					require.Nil(t, got[i], "key %q must be skipped", key)
				}
			}
			require.Equal(t, float64(tc.wantSkipped), testutil.ToFloat64(client.skipped),
				"skipped counter must equal the number of items above the cap")
		})
	}
}

func mockRedisClientSingle() (*RedisClient, error) {
	redisServer, err := miniredis.Run()
	if err != nil {
		return nil, err
	}

	cfg := &RedisConfig{
		Expiration: time.Minute,
		Timeout:    100 * time.Millisecond,
		Endpoint: strings.Join([]string{
			redisServer.Addr(),
		}, ","),
		SingleNode: true,
	}

	return NewRedisClient(cfg, "test", prometheus.NewRegistry())
}

func mockRedisClientCluster() (*RedisClient, error) {
	redisServer1, err := miniredis.Run()
	if err != nil {
		return nil, err
	}

	redisServer2, err := miniredis.Run()
	if err != nil {
		return nil, err
	}

	cfg := &RedisConfig{
		Expiration: time.Minute,
		Timeout:    100 * time.Millisecond,
		Endpoint: strings.Join([]string{
			redisServer1.Addr(),
			redisServer2.Addr(),
		}, ","),
	}

	return NewRedisClient(cfg, "test", prometheus.NewRegistry())
}

// TestRedisClient_TimeoutBoundsHangingServer guards the wiring that bounds
// every Redis command by cfg.Timeout. go-redis v9 ignores context deadlines
// for socket I/O unless ContextTimeoutEnabled is set, and its DialTimeout /
// ReadTimeout / WriteTimeout fall back to 5s/3s/3s defaults when left at
// zero — so a forgotten mapping silently lets a configured 100ms turn into a
// multi-second stall. The server here accepts TCP connections but never
// writes a reply, which is the scenario the defaults mask.
func TestRedisClient_TimeoutBoundsHangingServer(t *testing.T) {
	tests := []struct {
		name       string
		singleNode bool
	}{
		{name: "single node", singleNode: true},
		{name: "cluster", singleNode: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			addr := newHangingTCPServer(t)

			cfg := &RedisConfig{
				Endpoint:   addr,
				Timeout:    100 * time.Millisecond,
				SingleNode: tc.singleNode,
				// Cluster only: keep the retry blast-radius small so a regression
				// to multi-second socket defaults is unambiguous in the upper
				// bound below rather than swallowed by MaxRedirects*ReadTimeout.
				MaxRedirects: 0,
			}
			client, err := NewRedisClient(cfg, "test-hanging-"+tc.name, prometheus.NewRegistry())
			require.NoError(t, err)
			defer client.Close()

			start := time.Now()
			err = client.Ping(context.Background())
			elapsed := time.Since(start)

			require.Error(t, err, "Ping against a hanging server must fail")
			// Headroom over cfg.Timeout=100ms accommodates scheduling jitter
			// and, in cluster mode, the initial CLUSTER SLOTS probe before
			// PING. Without the fix this elapsed ~3s (go-redis's default
			// ReadTimeout), so 1.5s is comfortably below the regression and
			// well above the legitimate timeout.
			require.Less(t, elapsed, 1500*time.Millisecond,
				"Ping must respect cfg.Timeout=100ms, took %v", elapsed)
		})
	}
}

// newHangingTCPServer returns the address of a TCP listener that accepts
// connections and holds them open without ever writing a byte back. Accepted
// connections are tracked so the test cleanup can close them and unblock any
// goroutines that may still be reading on the client side.
func newHangingTCPServer(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	var (
		mu    sync.Mutex
		conns []net.Conn
	)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			conns = append(conns, conn)
			mu.Unlock()
		}
	}()

	t.Cleanup(func() {
		_ = ln.Close()
		mu.Lock()
		defer mu.Unlock()
		for _, c := range conns {
			_ = c.Close()
		}
	})
	return ln.Addr().String()
}
