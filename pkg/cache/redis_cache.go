package cache

import (
	"context"
	"errors"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/go-redis/redis/v8"
	instr "github.com/grafana/dskit/instrument"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel/attribute"

	util_log "github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/pkg/util/spanlogger"
)

// RedisCache type caches chunks in redis
type RedisCache struct {
	name            string
	redis           *RedisClient
	logger          log.Logger
	requestDuration *instr.HistogramCollector
}

// NewRedisCache creates a new RedisCache
func NewRedisCache(name string, redisClient *RedisClient, reg prometheus.Registerer, logger log.Logger) *RedisCache {
	util_log.WarnExperimentalUse("Redis cache")
	cache := &RedisCache{
		name:   name,
		redis:  redisClient,
		logger: logger,
		requestDuration: instr.NewHistogramCollector(
			promauto.With(reg).NewHistogramVec(prometheus.HistogramOpts{
				Namespace:                       "tempo",
				Name:                            "rediscache_request_duration_seconds",
				Help:                            "Total time spent in seconds doing Redis requests.",
				Buckets:                         prometheus.ExponentialBuckets(0.000016, 4, 8),
				NativeHistogramBucketFactor:     1.1,
				NativeHistogramMaxBucketNumber:  100,
				NativeHistogramMinResetDuration: 1 * time.Hour,
				ConstLabels:                     prometheus.Labels{"name": name},
			}, []string{"method", "status_code"}),
		),
	}
	if err := cache.redis.Ping(context.Background()); err != nil {
		level.Error(logger).Log("msg", "error connecting to redis", "name", name, "err", err)
	}
	return cache
}

func redisStatusCode(err error) string {
	// TODO: Figure out if there are more error types returned by Redis
	switch err {
	case nil:
		return "200"
	default:
		return "500"
	}
}

// Fetch gets keys from the cache. The keys that are found must be in the order of the keys requested.
func (c *RedisCache) Fetch(ctx context.Context, keys []string) (found []string, bufs [][]byte, missed []string) {
	const method = "RedisCache.MGet"
	var items [][]byte
	// Run a tracked request, using c.requestDuration to monitor requests.
	err := instr.CollectedRequest(ctx, method, c.requestDuration, redisStatusCode, func(ctx context.Context) error {
		log, _ := spanlogger.New(ctx, method)
		defer log.End()
		log.SetAttributes(attribute.Int("keys requested", len(keys)))

		var err error
		items, err = c.redis.MGet(ctx, keys)
		if err != nil {
			// nolint:errcheck
			log.Error(err)
			level.Error(c.logger).Log("msg", "failed to get from redis", "name", c.name, "err", err)
			return err
		}

		log.SetAttributes(attribute.Int("keys found", len(items)))

		return nil
	})
	if err != nil {
		return found, bufs, keys
	}

	for i, key := range keys {
		if items[i] != nil {
			found = append(found, key)
			bufs = append(bufs, items[i])
		} else {
			missed = append(missed, key)
		}
	}

	return
}

// Fetch gets a single keys from the cache
func (c *RedisCache) FetchKey(ctx context.Context, key string) (buf []byte, found bool) {
	const method = "RedisCache.Get"
	// Run a tracked request, using c.requestDuration to monitor requests.
	err := instr.CollectedRequest(ctx, method, c.requestDuration, redisStatusCode, func(ctx context.Context) error {
		log, _ := spanlogger.New(ctx, method)
		defer log.Finish()
		var err error
		buf, err = c.redis.Get(ctx, key)
		if err != nil {
			// nolint:errcheck
			log.Error(err)
			if errors.Is(err, redis.Nil) {
				level.Debug(c.logger).Log("msg", "failed to get key from redis", "name", c.name, "err", err, "key", key)
			} else {
				level.Error(c.logger).Log("msg", "error requesting key from redis", "name", c.name, "err", err, "key", key)
			}

			return err
		}

		log.LogFields(otlog.String("key found", key))

		return nil
	})
	if err != nil {
		return buf, false
	}

	return buf, true
}

// Store stores the key in the cache.
func (c *RedisCache) Store(ctx context.Context, keys []string, bufs [][]byte) {
	err := c.redis.MSet(ctx, keys, bufs)
	if err != nil {
		level.Error(c.logger).Log("msg", "failed to put to redis", "name", c.name, "err", err)
	}
}

// Stop stops the redis client.
func (c *RedisCache) Stop() {
	_ = c.redis.Close()
}

// redis doesn't have a max item size. todo: add
func (c *RedisCache) MaxItemSize() int {
	return 0
}
