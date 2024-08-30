package cache

import (
	"context"
	"errors"
	"flag"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	instr "github.com/grafana/dskit/instrument"
	"github.com/grafana/gomemcache/memcache"
	"github.com/grafana/tempo/pkg/util/math"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MemcachedConfig is config to make a Memcached
type MemcachedConfig struct {
	Expiration time.Duration `yaml:"expiration"`

	BatchSize   int `yaml:"batch_size"`
	Parallelism int `yaml:"parallelism"`
}

// RegisterFlagsWithPrefix adds the flags required to config this to the given FlagSet
func (cfg *MemcachedConfig) RegisterFlagsWithPrefix(prefix, description string, f *flag.FlagSet) {
	f.DurationVar(&cfg.Expiration, prefix+"memcached.expiration", 0, description+"How long keys stay in the memcache.")
	f.IntVar(&cfg.BatchSize, prefix+"memcached.batchsize", 1024, description+"How many keys to fetch in each batch.")
	f.IntVar(&cfg.Parallelism, prefix+"memcached.parallelism", 100, description+"Maximum active requests to memcache.")
}

// Memcached type caches chunks in memcached
type Memcached struct {
	cfg         MemcachedConfig
	memcache    MemcachedClient
	name        string
	maxItemSize int

	requestDuration *instr.HistogramCollector

	wg      sync.WaitGroup
	inputCh chan *work
	quit    chan struct{}

	logger log.Logger
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

	if cfg.BatchSize == 0 || cfg.Parallelism == 0 {
		return c
	}

	c.inputCh = make(chan *work)
	c.quit = make(chan struct{})
	c.wg.Add(cfg.Parallelism)

	for i := 0; i < cfg.Parallelism; i++ {
		go func() {
			defer c.wg.Done()
			for {
				select {
				case <-c.quit:
					return
				case input := <-c.inputCh:
					res := &result{
						batchID: input.batchID,
					}
					res.found, res.bufs, res.missed = c.fetch(input.ctx, input.keys)
					// No-one will be reading from resultCh if we were asked to quit
					// during the fetch, so check again before writing to it.
					select {
					case <-c.quit:
						return
					case input.resultCh <- res:
					}
				}
			}
		}()
	}

	return c
}

type work struct {
	keys     []string
	ctx      context.Context
	resultCh chan<- *result
	batchID  int // For ordering results.
}

type result struct {
	found   []string
	bufs    [][]byte
	missed  []string
	batchID int // For ordering results.
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
	if c.cfg.BatchSize == 0 {
		found, bufs, missed = c.fetch(ctx, keys)
		return
	}
	_ = measureRequest(ctx, "Memcache.GetBatched", c.requestDuration, memcacheStatusCode, func(ctx context.Context) error {
		found, bufs, missed = c.fetchKeysBatched(ctx, keys)
		return nil
	})
	return
}

// FetchKey gets a single key from the cache
func (c *Memcached) FetchKey(ctx context.Context, key string) (buf []byte, found bool) {
	const method = "Memcache.Get"
	var item *memcache.Item
	err := measureRequest(ctx, method, c.requestDuration, memcacheStatusCode, func(_ context.Context) error {
		var err error
		item, err = c.memcache.Get(key)
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
		return buf, false
	}
	return item.Value, true
}

func (c *Memcached) fetch(ctx context.Context, keys []string) (found []string, bufs [][]byte, missed []string) {
	var items map[string]*memcache.Item
	const method = "Memcache.GetMulti"
	err := measureRequest(ctx, method, c.requestDuration, memcacheStatusCode, func(_ context.Context) error {
		var err error
		items, err = c.memcache.GetMulti(keys)
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

func (c *Memcached) fetchKeysBatched(ctx context.Context, keys []string) (found []string, bufs [][]byte, missed []string) {
	resultsCh := make(chan *result)
	batchSize := c.cfg.BatchSize

	go func() {
		for i, j := 0, 0; i < len(keys); i += batchSize {
			batchKeys := keys[i:math.Min(i+batchSize, len(keys))]
			select {
			case <-c.quit:
				return
			case c.inputCh <- &work{
				keys:     batchKeys,
				ctx:      ctx,
				resultCh: resultsCh,
				batchID:  j,
			}:
			}
			j++
		}
	}()

	// Read all values from this channel to avoid blocking upstream.
	numResults := len(keys) / batchSize
	if len(keys)%batchSize != 0 {
		numResults++
	}

	// We need to order found by the input keys order.
	results := make([]*result, numResults)
loopResults:
	for i := 0; i < numResults; i++ {
		select {
		case <-c.quit:
			break loopResults
		case result := <-resultsCh:
			results[result.batchID] = result
		}
	}

	for _, result := range results {
		if result == nil {
			continue
		}
		found = append(found, result.found...)
		bufs = append(bufs, result.bufs...)
		missed = append(missed, result.missed...)
	}

	return
}

// Store stores the key in the cache.
func (c *Memcached) Store(ctx context.Context, keys []string, bufs [][]byte) {
	for i := range keys {
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

func measureRequest(ctx context.Context, method string, col instr.Collector, toStatusCode func(error) string, f func(context.Context) error) error {
	start := time.Now()
	col.Before(ctx, method, start)
	err := f(ctx)
	col.After(ctx, method, toStatusCode(err), start)
	return err
}

// Stop does nothing.
func (c *Memcached) Stop() {
	if c.quit == nil {
		return
	}

	select {
	case <-c.quit:
	default:
		close(c.quit)
	}
	c.wg.Wait()
}

func (c *Memcached) MaxItemSize() int {
	return c.maxItemSize
}
