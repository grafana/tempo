package cache

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

// BenchmarkRedisClient_MGet_Cluster compares the previous per-key sequential
// fallback (the v8 behaviour) against the native MGet shipped with go-redis
// over a multi-node cluster client.
//
// miniredis does not enforce slots, so the speedup observed here reflects
// RTT/pipelining savings at the client and transport layer — which is the
// dominant cost in production cross-slot MGet as well.
func BenchmarkRedisClient_MGet_Cluster(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		c := newBenchClusterClient(b)
		keys, _ := prepopulateBenchKeys(b, c, n)

		b.Run(fmt.Sprintf("keys=%d/sequential_get", n), func(b *testing.B) {
			ctx := context.Background()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ret := make([][]byte, len(keys))
				for j, key := range keys {
					val, err := c.rdb.Get(ctx, key).Result()
					// Mirror the original v8 cluster fallback: treat misses
					// (redis.Nil) as nil entries rather than failing the bench.
					// miniredis does not implement cluster slot routing, so a
					// per-key Get against a fake 2-node cluster can land on a
					// node that doesn't hold the key.
					if errors.Is(err, redis.Nil) {
						continue
					}
					if err != nil {
						b.Fatalf("get %q: %v", key, err)
					}
					ret[j] = StringToBytes(val)
				}
			}
		})

		b.Run(fmt.Sprintf("keys=%d/native_mget", n), func(b *testing.B) {
			ctx := context.Background()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := c.MGet(ctx, keys); err != nil {
					b.Fatalf("mget: %v", err)
				}
			}
		})
	}
}

func newBenchClusterClient(b *testing.B) *RedisClient {
	b.Helper()
	s1, err := miniredis.Run()
	if err != nil {
		b.Fatalf("miniredis 1: %v", err)
	}
	b.Cleanup(s1.Close)
	s2, err := miniredis.Run()
	if err != nil {
		b.Fatalf("miniredis 2: %v", err)
	}
	b.Cleanup(s2.Close)

	cfg := &RedisConfig{
		// No TTL — long benchmark runs would otherwise let pre-populated keys expire.
		Expiration: 0,
		Timeout:    5 * time.Second,
		Endpoint:   strings.Join([]string{s1.Addr(), s2.Addr()}, ","),
	}
	c, err := NewRedisClient(cfg, "bench", prometheus.NewRegistry())
	if err != nil {
		b.Fatalf("NewRedisClient: %v", err)
	}
	b.Cleanup(func() { _ = c.Close() })
	return c
}

func prepopulateBenchKeys(b *testing.B, c *RedisClient, n int) ([]string, [][]byte) {
	b.Helper()
	keys := make([]string, n)
	values := make([][]byte, n)
	for i := range n {
		keys[i] = fmt.Sprintf("bench-key-%d", i)
		values[i] = fmt.Appendf(nil, "bench-value-%d", i)
	}
	if err := c.MSet(context.Background(), keys, values); err != nil {
		b.Fatalf("seed mset: %v", err)
	}
	// warm-up to avoid first-call latency leaking into measurements.
	if _, err := c.MGet(context.Background(), keys); err != nil {
		b.Fatalf("warmup mget: %v", err)
	}
	return keys, values
}
