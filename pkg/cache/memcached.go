package cache

import (
	"context"
	"errors"
	"flag"
	"os"
	"strconv"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	instr "github.com/grafana/dskit/instrument"
	"github.com/grafana/gomemcache/memcache"
	"github.com/grafana/tempo/pkg/pool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MemcachedConfig is config to make a Memcached
type MemcachedConfig struct {
	Expiration time.Duration `yaml:"expiration"`
}

// RegisterFlagsWithPrefix adds the flags required to config this to the given FlagSet
func (cfg *MemcachedConfig) RegisterFlagsWithPrefix(prefix, description string, f *flag.FlagSet) {
	f.DurationVar(&cfg.Expiration, prefix+"memcached.expiration", 0, description+"How long keys stay in the memcache.")
}

var cacheBufAllocator *allocator

func init() {
	bktSize := intFromEnv("CACHE_BKT_SIZE", 40*1024)
	numBuckets := intFromEnv("CACHE_NUM_BUCKETS", 100)
	minBucket := intFromEnv("CACHE_MIN_BUCKET", 0)

	cacheBufAllocator = &allocator{
		pool: pool.New("cache", minBucket, numBuckets, bktSize),
	}
}

// Memcached type caches chunks in memcached
type Memcached struct {
	cfg             MemcachedConfig
	memcache        MemcachedClient
	name            string
	maxItemSize     int
	requestDuration *instr.HistogramCollector
	logger          log.Logger
}

// NewMemcached makes a new Memcached.
func NewMemcached(cfg MemcachedConfig, client MemcachedClient, name string, maxItemSize int, reg prometheus.Registerer, logger log.Logger) *Memcached {
	c := &Memcached{
		cfg:         cfg,
		memcache:    client,
		name:        name,
		maxItemSize: maxItemSize,
		logger:      logger,
		requestDuration: instr.NewHistogramCollector(
			promauto.With(reg).NewHistogramVec(prometheus.HistogramOpts{
				Namespace: "tempo",
				Name:      "memcache_request_duration_seconds",
				Help:      "Total time spent in seconds doing memcache requests.",
				// Memcached requests are very quick: smallest bucket is 16us, biggest is 1s
				Buckets:                         prometheus.ExponentialBuckets(0.000016, 4, 8),
				NativeHistogramBucketFactor:     1.1,
				NativeHistogramMaxBucketNumber:  100,
				NativeHistogramMinResetDuration: 1 * time.Hour,
				ConstLabels:                     prometheus.Labels{"name": name},
			}, []string{"method", "status_code"}),
		),
	}
	return c
}

func memcacheStatusCode(err error) string {
	// See https://godoc.org/github.com/bradfitz/gomemcache/memcache#pkg-variables
	if errors.Is(err, memcache.ErrCacheMiss) {
		return "404"
	}
	if errors.Is(err, memcache.ErrMalformedKey) {
		return "400"
	}
	if err != nil {
		return "500"
	}
	return "200"
}

// Fetch gets keys from the cache. The keys that are found must be in the order of the keys requested.
func (c *Memcached) Fetch(ctx context.Context, keys []string) (found []string, bufs [][]byte, missed []string) {
	select {
	case <-ctx.Done():
		missed = keys
		return
	default:
	}

	var items map[string]*memcache.Item
	const method = "Memcache.GetMulti"

	err := measureRequest(ctx, method, c.requestDuration, memcacheStatusCode, func(_ context.Context) error {
		var err error
		items, err = c.memcache.GetMulti(ctx, keys, memcache.WithAllocator(cacheBufAllocator))
		if err != nil {
			level.Error(c.logger).Log("msg", "Failed to get keys from memcached", "err", err)
		}
		return err
	})
	if err != nil {
		return found, bufs, keys
	}
	for _, key := range keys {
		item, ok := items[key]
		if ok {
			found = append(found, key)
			bufs = append(bufs, item.Value)
		} else {
			missed = append(missed, key)
		}
	}
	return
}

// FetchKey gets a single key from the cache
func (c *Memcached) FetchKey(ctx context.Context, key string) ([]byte, bool) {
	select {
	case <-ctx.Done():
		return nil, false
	default:
	}

	const method = "Memcache.Get"
	var item *memcache.Item
	err := measureRequest(ctx, method, c.requestDuration, memcacheStatusCode, func(_ context.Context) error {
		var err error
		item, err = c.memcache.Get(key, memcache.WithAllocator(cacheBufAllocator))
		if err != nil {
			if errors.Is(err, memcache.ErrCacheMiss) {
				level.Debug(c.logger).Log("msg", "Failed to get key from memcached", "err", err, "key", key)
			} else {
				level.Error(c.logger).Log("msg", "Error getting key from memcached", "err", err, "key", key)
			}
		}
		return err
	})
	if err != nil {
		return nil, false
	}
	return item.Value, true
}

// Store stores the key in the cache.
func (c *Memcached) Store(ctx context.Context, keys []string, bufs [][]byte) {
	for i := range keys {
		select {
		case <-ctx.Done():
			return
		default:
		}

		err := measureRequest(ctx, "Memcache.Put", c.requestDuration, memcacheStatusCode, func(_ context.Context) error {
			item := memcache.Item{
				Key:        keys[i],
				Value:      bufs[i],
				Expiration: int32(c.cfg.Expiration.Seconds()),
			}
			return c.memcache.Set(&item)
		})
		if err != nil {
			level.Error(c.logger).Log("msg", "failed to put to memcached", "name", c.name, "err", err)
		}
	}
}

func (c *Memcached) Stop() {
	c.memcache.Close()
}

func (c *Memcached) MaxItemSize() int {
	return c.maxItemSize
}

func (c *Memcached) Release(buf []byte) {
	cacheBufAllocator.Put(&buf)
}

// allocator is a thin wrapper around pool.Pool to satisfy the memcache allocator interface
type allocator struct {
	pool *pool.Pool
}

func (a *allocator) Get(sz int) *[]byte {
	buffer := a.pool.Get(sz)
	return &buffer
}

func (a *allocator) Put(b *[]byte) {
	a.pool.Put(*b)
}

func intFromEnv(env string, defaultValue int) int {
	// get the value from the environment
	val, ok := os.LookupEnv(env)
	if !ok {
		return defaultValue
	}

	// try to parse the value
	intVal, err := strconv.Atoi(val)
	if err != nil {
		panic("failed to parse " + env + " as int")
	}

	return intVal
}
