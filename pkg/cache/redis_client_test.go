package cache

import (
	"context"
	"strings"
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
		})
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

			client := NewRedisClient(&RedisConfig{
				Expiration:  time.Minute,
				Timeout:     100 * time.Millisecond,
				Endpoint:    redisServer.Addr(),
				MaxItemSize: tc.maxItemSize,
			}, "test", prometheus.NewRegistry())

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
	}

	return NewRedisClient(cfg, "test", prometheus.NewRegistry()), nil
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

	return NewRedisClient(cfg, "test", prometheus.NewRegistry()), nil
}
