package cache

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"

	dstls "github.com/grafana/dskit/crypto/tls"
	"github.com/grafana/dskit/flagext"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/redis/go-redis/v9"
)

const (
	// redisScanCount is the COUNT hint passed to SCAN while evicting a block's
	// keys by prefix. COUNT is only a hint for how many keys SCAN examines per
	// call, not a cap on the result; a larger value trades fewer round trips for
	// more work — and, since Redis is single-threaded, higher latency for other
	// clients — per call. 20000 favors throughput for this low-frequency,
	// compaction-driven scan.
	redisScanCount = 20000
	// redisDeleteBatch caps how many keys are sent in a single DEL while draining
	// a scan, bounding command size on large blocks.
	redisDeleteBatch = 1000
)

// globPrefixReplacer escapes the glob metacharacters that Redis SCAN MATCH
// interprets (* ? [ ] and the \ escape itself), so a prefix built from a tenant
// ID is matched literally rather than as a pattern. strings.Replacer performs a
// single non-overlapping pass, so the backslashes it inserts are not re-escaped.
var globPrefixReplacer = strings.NewReplacer(
	`\`, `\\`,
	`*`, `\*`,
	`?`, `\?`,
	`[`, `\[`,
	`]`, `\]`,
)

// redisScanner enumerates keys with SCAN. On a cluster it is a single master node
// (SCAN has no cluster-wide form); on a single node it is the client itself.
// Satisfied by *redis.Client, *redis.ClusterClient, and the wrapped
// UniversalClient.
type redisScanner interface {
	Scan(ctx context.Context, cursor uint64, match string, count int64) *redis.ScanCmd
}

// redisDeleter deletes keys with DEL. On a cluster this MUST be the
// *redis.ClusterClient, never a per-master node client: DEL is a multi-key
// command and a Redis Cluster node rejects a single DEL whose keys span more than
// one hash slot with CROSSSLOT. The cluster client splits a multi-key DEL into
// per-slot subcommands (osscluster: executeMultiShard); a node client sends it
// as-is. A block's keys are spread across every slot a master owns — the whole
// key, not just the block ID, is hashed — so deleting a scanned batch through a
// node client would fail. Satisfied by both *redis.Client and
// *redis.ClusterClient.
type redisDeleter interface {
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

// RedisConfig defines how a RedisCache should be constructed.
type RedisConfig struct {
	Endpoint   string         `yaml:"endpoint"`
	Timeout    time.Duration  `yaml:"timeout"`
	Expiration time.Duration  `yaml:"expiration"`
	DB         int            `yaml:"db"`
	PoolSize   int            `yaml:"pool_size"`
	Username   string         `yaml:"username"`
	Password   flagext.Secret `yaml:"password"`

	// SingleNode opts into a single-node Redis client. Default (false) targets
	// a Redis Cluster — the YAML key is inverted so its Go zero value matches
	// the desired default and a missing field can never silently flip routing.
	SingleNode bool `yaml:"single_node"`

	TLSEnabled bool               `yaml:"tls_enabled"`
	TLS        dstls.ClientConfig `yaml:",inline"`

	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`

	// Cluster-only routing options. Ignored on the single-node path.
	RouteByLatency bool `yaml:"route_by_latency"`
	RouteRandomly  bool `yaml:"route_randomly"`
	ReadOnly       bool `yaml:"read_only"`
	MaxRedirects   int  `yaml:"max_redirects"`
	MinIdleConns   int  `yaml:"min_idle_conns"`

	// MaxItemSize is the maximum size in bytes of an item stored in Redis.
	// Items larger than this are not stored. A value of 0 disables the limit.
	MaxItemSize int `yaml:"max_item_size"`
}

// RegisterFlagsWithPrefix adds the flags required to config this to the given FlagSet
func (cfg *RedisConfig) RegisterFlagsWithPrefix(prefix, description string, f *flag.FlagSet) {
	f.StringVar(&cfg.Endpoint, prefix+"redis.endpoint", "", description+"Redis Server endpoint to use for caching. A comma-separated list of endpoints for Redis Cluster. If empty, no redis will be used.")
	f.DurationVar(&cfg.Timeout, prefix+"redis.timeout", 500*time.Millisecond, description+"Maximum time to wait before giving up on redis requests.")
	f.DurationVar(&cfg.Expiration, prefix+"redis.expiration", 0, description+"How long keys stay in the redis.")
	f.IntVar(&cfg.DB, prefix+"redis.db", 0, description+"Single-node only: database index. Ignored in cluster mode, which requires DB 0.")
	f.IntVar(&cfg.PoolSize, prefix+"redis.pool-size", 0, description+"Maximum number of connections in the pool.")
	f.StringVar(&cfg.Username, prefix+"redis.username", "", description+"Username to use when connecting to redis (utilizes Redis 6+ ACL-based AUTH)")
	f.Var(&cfg.Password, prefix+"redis.password", description+"Password to use when connecting to redis.")
	f.BoolVar(&cfg.SingleNode, prefix+"redis.single-node", false, description+"Connect to a single Redis node instead of a Redis Cluster.")
	f.BoolVar(&cfg.TLSEnabled, prefix+"redis.tls-enabled", false, description+"Enable connecting to redis with TLS.")
	cfg.TLS.RegisterFlagsWithPrefix(prefix+"redis.", f)
	f.DurationVar(&cfg.ConnMaxIdleTime, prefix+"redis.conn-max-idle-time", 0, description+"Close connections after remaining idle for this duration. If the value is zero, then idle connections are not closed.")
	f.DurationVar(&cfg.ConnMaxLifetime, prefix+"redis.conn-max-lifetime", 0, description+"Close connections older than this duration. If the value is zero, then the pool does not close connections based on age.")
	f.BoolVar(&cfg.RouteByLatency, prefix+"redis.route-by-latency", false, description+"Cluster only: route read-only commands to the node with the lowest measured latency.")
	f.BoolVar(&cfg.RouteRandomly, prefix+"redis.route-randomly", false, description+"Cluster only: route read-only commands to a random node.")
	f.BoolVar(&cfg.ReadOnly, prefix+"redis.read-only", false, description+"Cluster only: allow read-only commands on replica nodes. Reads may be stale.")
	// Cosmetic alignment with go-redis's own resolved default: go-redis remaps
	// MaxRedirects==0 to 3 internally (and -1 to "disable"), so a missing YAML
	// field already gets 3. Exposing 3 here just makes --help reflect the
	// effective default.
	f.IntVar(&cfg.MaxRedirects, prefix+"redis.max-redirects", 3, description+"Cluster only: maximum number of redirects to follow on MOVED/ASK responses. Set to -1 to disable retries.")
	f.IntVar(&cfg.MinIdleConns, prefix+"redis.min-idle-conns", 0, description+"Minimum number of idle connections to maintain in the pool. Useful to avoid the overhead of establishing new connections on demand.")
	f.IntVar(&cfg.MaxItemSize, prefix+"redis.max-item-size", 0, description+"The maximum size in bytes of an item stored in Redis. Items larger than this are not stored. A value of 0 disables the limit.")
}

type RedisClient struct {
	expiration time.Duration
	timeout    time.Duration
	rdb        redis.UniversalClient

	maxItemSizeBytes int
	skipped          prometheus.Counter
}

// NewRedisClient creates Redis client. It fails closed if cfg.TLSEnabled is
// set but the TLS configuration cannot be assembled — silently falling back
// to a cleartext connection would violate the operator's intent.
//
// Routing is explicit: cfg.SingleNode chooses redis.NewClient; otherwise
// redis.NewClusterClient. The previous UniversalClient-based selection
// inferred routing from address count, which silently picked single-node
// for cluster endpoints that expanded to a single host (e.g. AWS
// ElastiCache configuration endpoints).
func NewRedisClient(cfg *RedisConfig, name string, reg prometheus.Registerer) (*RedisClient, error) {
	var tlsCfg *tls.Config
	if cfg.TLSEnabled {
		t, err := cfg.TLS.GetTLSConfig()
		if err != nil {
			return nil, fmt.Errorf("redis: invalid TLS configuration: %w", err)
		}
		tlsCfg = t
	}

	// go-redis v9 ignores context deadlines for socket I/O unless
	// ContextTimeoutEnabled is set, and its DialTimeout/ReadTimeout/WriteTimeout
	// fall back to multi-second defaults (5s/3s/3s) when left at zero. Mapping
	// cfg.Timeout into all four socket-timeout fields keeps each network op
	// bounded even when the caller's context is background, and enabling
	// ContextTimeoutEnabled honors the wrapper's WithTimeout in Ping/MGet/etc.
	var rdb redis.UniversalClient
	if cfg.SingleNode {
		rdb = redis.NewClient(&redis.Options{
			Addr:                  cfg.Endpoint,
			Username:              cfg.Username,
			Password:              cfg.Password.String(),
			DB:                    cfg.DB,
			PoolSize:              cfg.PoolSize,
			MinIdleConns:          cfg.MinIdleConns,
			ConnMaxIdleTime:       cfg.ConnMaxIdleTime,
			ConnMaxLifetime:       cfg.ConnMaxLifetime,
			TLSConfig:             tlsCfg,
			ContextTimeoutEnabled: true,
			DialTimeout:           cfg.Timeout,
			ReadTimeout:           cfg.Timeout,
			WriteTimeout:          cfg.Timeout,
			PoolTimeout:           cfg.Timeout,
		})
	} else {
		cc := redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:                 strings.Split(cfg.Endpoint, ","),
			Username:              cfg.Username,
			Password:              cfg.Password.String(),
			PoolSize:              cfg.PoolSize,
			MinIdleConns:          cfg.MinIdleConns,
			ConnMaxIdleTime:       cfg.ConnMaxIdleTime,
			ConnMaxLifetime:       cfg.ConnMaxLifetime,
			RouteByLatency:        cfg.RouteByLatency,
			RouteRandomly:         cfg.RouteRandomly,
			ReadOnly:              cfg.ReadOnly,
			MaxRedirects:          cfg.MaxRedirects,
			TLSConfig:             tlsCfg,
			ContextTimeoutEnabled: true,
			DialTimeout:           cfg.Timeout,
			ReadTimeout:           cfg.Timeout,
			WriteTimeout:          cfg.Timeout,
			PoolTimeout:           cfg.Timeout,
		})
		// Wire up COMMAND INFO-driven routing so multi-key commands
		// (MGET/DEL) get sharded across slots instead of failing with
		// CROSSSLOT. Requires Redis 7+, which advertises the multi_shard
		// request_policy via COMMAND INFO tips.
		cc.GetResolver().SetFallbackResolver(cc.NewDynamicResolver())
		rdb = cc
	}

	return &RedisClient{
		expiration:       cfg.Expiration,
		timeout:          cfg.Timeout,
		rdb:              rdb,
		maxItemSizeBytes: cfg.MaxItemSize,
		skipped: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Namespace:   "tempo",
			Name:        "rediscache_client_set_skip_total",
			Help:        "Total number of skipped set operations because the value is larger than max-item-size.",
			ConstLabels: prometheus.Labels{"name": name},
		}),
	}, nil
}

func (c *RedisClient) Ping(ctx context.Context) error {
	var cancel context.CancelFunc
	if c.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	pong, err := c.rdb.Ping(ctx).Result()
	if err != nil {
		return err
	}
	if pong != "PONG" {
		return fmt.Errorf("redis: Unexpected PING response %q", pong)
	}
	return nil
}

func (c *RedisClient) MSet(ctx context.Context, keys []string, values [][]byte) error {
	var cancel context.CancelFunc
	if c.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	if len(keys) != len(values) {
		return fmt.Errorf("MSet the length of keys and values not equal, len(keys)=%d, len(values)=%d", len(keys), len(values))
	}

	// Pipeline (not TxPipeline / MULTI-EXEC) — cache writes are independent and
	// best-effort, and MULTI/EXEC rejects cross-slot keys in cluster mode.
	pipe := c.rdb.Pipeline()
	for i := range keys {
		// Skip hitting redis at all if the item is larger than the max allowed size.
		if c.maxItemSizeBytes > 0 && len(values[i]) > c.maxItemSizeBytes {
			c.skipped.Inc()
			continue
		}
		pipe.Set(ctx, keys[i], values[i], c.expiration)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (c *RedisClient) MGet(ctx context.Context, keys []string) ([][]byte, error) {
	var cancel context.CancelFunc
	if c.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	cmd := c.rdb.MGet(ctx, keys...)
	if err := cmd.Err(); err != nil {
		return nil, err
	}

	vals := cmd.Val()
	ret := make([][]byte, len(keys))
	for i, val := range vals {
		if val != nil {
			ret[i] = StringToBytes(val.(string))
		}
	}
	return ret, nil
}

func (c *RedisClient) Get(ctx context.Context, key string) ([]byte, error) {
	var cancel context.CancelFunc
	if c.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}
	cmd := c.rdb.Get(ctx, key)
	err := cmd.Err()
	if err != nil {
		return nil, err
	}

	return StringToBytes(cmd.Val()), nil
}

func (c *RedisClient) Del(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	var cancel context.CancelFunc
	if c.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	return c.rdb.Del(ctx, keys...).Err()
}

// RemoveByPrefix deletes every key whose name begins with prefix and returns the
// number of keys deleted. On a Redis Cluster the keyspace is enumerated on each
// master (SCAN has no cluster-wide form, and a block's keys are spread across
// slots because the whole key — not just the block ID — is hashed, so
// ForEachMaster covers the entire keyspace), while deletion is routed through the
// cluster client, which splits each multi-key DEL across slots; deleting through
// the per-master node client instead would fail with CROSSSLOT (see
// redisDeleter). On a single node both run on the one client. Unlike the other
// operations this is not bounded by c.timeout as a whole (a scan may legitimately
// take much longer than a single request); each underlying SCAN/DEL is still
// bounded by the client's socket timeouts, and the caller's context cancels the
// loop.
func (c *RedisClient) RemoveByPrefix(ctx context.Context, prefix string) (int, error) {
	pattern := globPrefixReplacer.Replace(prefix) + "*"

	if cc, ok := c.rdb.(*redis.ClusterClient); ok {
		var total atomic.Int64
		// ForEachMaster runs the callback concurrently across masters. Enumerate
		// on the node, but delete through cc so each DEL is split per slot.
		err := cc.ForEachMaster(ctx, func(ctx context.Context, node *redis.Client) error {
			n, err := scanAndDelete(ctx, node, cc, pattern)
			total.Add(int64(n))
			return err
		})
		return int(total.Load()), err
	}

	return scanAndDelete(ctx, c.rdb, c.rdb, pattern)
}

// scanAndDelete iterates a single node's keyspace with SCAN (via scanner) and
// deletes every key matching pattern (via deleter). It always drives the cursor
// to completion: SCAN returns an opaque cursor alongside a best-effort batch, so
// an empty batch paired with a non-zero cursor is normal (MATCH filters after
// COUNT elements are examined) and iteration must continue until the cursor
// returns to 0. Scanning and deletion are separate interfaces because on a
// cluster the scanner is a single master node while the deleter must be the
// slot-splitting cluster client (see redisDeleter).
func scanAndDelete(ctx context.Context, scanner redisScanner, deleter redisDeleter, pattern string) (int, error) {
	var (
		cursor  uint64
		pending []string
		deleted int
	)

	// flush drains pending in DEL commands of at most redisDeleteBatch keys, so a
	// block with many cached ranges does not produce one oversized command.
	flush := func() error {
		for len(pending) > 0 {
			n := min(redisDeleteBatch, len(pending))
			cnt, err := deleter.Del(ctx, pending[:n]...).Result()
			if err != nil {
				return err
			}
			deleted += int(cnt)
			pending = pending[n:]
		}
		return nil
	}

	for {
		if err := ctx.Err(); err != nil {
			return deleted, err
		}

		var (
			keys []string
			err  error
		)
		keys, cursor, err = scanner.Scan(ctx, cursor, pattern, redisScanCount).Result()
		if err != nil {
			return deleted, err
		}

		pending = append(pending, keys...)
		if len(pending) >= redisDeleteBatch {
			if err := flush(); err != nil {
				return deleted, err
			}
		}

		// Only a zero cursor signals the end of iteration; an empty batch does not.
		if cursor == 0 {
			break
		}
	}

	return deleted, flush()
}

func (c *RedisClient) Close() error {
	return c.rdb.Close()
}

// StringToBytes returns a byte slice aliasing s without a copy.
//
// The returned slice shares storage with s, so callers MUST NOT write to it —
// mutating the bytes would violate the immutability of the source string and
// is undefined behavior. Use this only when you immediately read the result;
// if you need to retain or modify the data, copy it first.
func StringToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
