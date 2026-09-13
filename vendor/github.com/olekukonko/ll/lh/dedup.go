package lh

import (
	"bytes"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/olekukonko/ll/lx"
)

// shardCount determines the number of shards for the dedup handler.
// Must be a power of 2 for efficient modulo via bitwise AND.
const shardCount = 32

// Dedup is a log handler that suppresses duplicate entries within a TTL window.
// It wraps another handler and filters out repeated log entries that match
// within the deduplication period.
type Dedup struct {
	next         lx.Handler
	ttl          time.Duration
	cleanupEvery time.Duration
	keyFn        lx.Deduper
	maxKeys      int
	shards       [shardCount]dedupShard // value array; take &shards[i] when locking
	done         chan struct{}
	wg           sync.WaitGroup
	once         sync.Once
}

type dedupShard struct {
	mu   sync.Mutex
	seen map[uint64]int64 // key -> expiry unix-nano timestamp
}

// DedupOpt configures a Dedup handler.
type DedupOpt func(*Dedup)

// WithDedupKeyFunc customizes how deduplication keys are generated.
func WithDedupKeyFunc(fn func(*lx.Entry) uint64) DedupOpt {
	return func(d *Dedup) {
		d.keyFn = dedupKeyFunc(fn)
	}
}

type dedupKeyFunc func(*lx.Entry) uint64

func (f dedupKeyFunc) Calculate(e *lx.Entry) uint64 {
	return f(e)
}

// WithDedupCleanupInterval sets how often expired deduplication keys are purged.
func WithDedupCleanupInterval(every time.Duration) DedupOpt {
	return func(d *Dedup) {
		if every > 0 {
			d.cleanupEvery = every
		}
	}
}

// WithDedupMaxKeys sets a soft limit on tracked deduplication keys.
func WithDedupMaxKeys(max int) DedupOpt {
	return func(d *Dedup) {
		if max > 0 {
			d.maxKeys = max
		}
	}
}

// WithDedupIgnore specifies fields to ignore in the default key function.
func WithDedupIgnore(fields ...string) DedupOpt {
	return func(d *Dedup) {
		if dd, ok := d.keyFn.(*defaultDedup); ok {
			if dd.ignoreFields == nil {
				dd.ignoreFields = make(map[string]struct{}, len(fields))
			}
			for _, f := range fields {
				dd.ignoreFields[f] = struct{}{}
			}
		}
	}
}

// NewDedup creates a deduplicating handler wrapper.
func NewDedup(next lx.Handler, ttl time.Duration, opts ...DedupOpt) *Dedup {
	if ttl <= 0 {
		ttl = 2 * time.Second
	}
	d := &Dedup{
		next:         next,
		ttl:          ttl,
		cleanupEvery: time.Minute,
		keyFn:        NewDefaultDedup(),
		done:         make(chan struct{}),
	}
	// Pre-allocate each shard's map to avoid growth allocations at startup.
	for i := 0; i < len(d.shards); i++ {
		d.shards[i].seen = make(map[uint64]int64, 64)
	}
	for _, opt := range opts {
		opt(d)
	}
	d.wg.Add(1)
	go d.cleanupLoop()
	return d
}

// Handle processes a log entry, suppressing duplicates within the TTL window.
func (d *Dedup) Handle(e *lx.Entry) error {
	// Guard against nil keyFn — pass through if not configured.
	if d.keyFn == nil {
		return d.next.Handle(e)
	}

	now := time.Now().UnixNano()
	key := d.keyFn.Calculate(e)

	// Bitwise AND is safe because shardCount is a power of 2.
	shard := &d.shards[key&(shardCount-1)]

	shard.mu.Lock()
	exp, ok := shard.seen[key]
	if ok && now < exp {
		shard.mu.Unlock()
		return nil // duplicate within TTL — suppress
	}

	// Opportunistic per-shard cleanup when the shard is getting full.
	if d.maxKeys > 0 {
		limitPerShard := d.maxKeys / shardCount
		if limitPerShard > 0 && len(shard.seen) >= limitPerShard {
			d.cleanupShardLocked(shard, now)
		}
	}

	shard.seen[key] = now + d.ttl.Nanoseconds()
	shard.mu.Unlock()

	return d.next.Handle(e)
}

// getShardIndex returns the shard index for a given key.
// Uses bitwise AND since shardCount is a power of 2.
func (d *Dedup) getShardIndex(key uint64) int {
	return int(key & (shardCount - 1))
}

// Close stops the cleanup goroutine and closes the underlying handler.
func (d *Dedup) Close() error {
	var err error
	d.once.Do(func() {
		close(d.done)
		d.wg.Wait()
		if c, ok := d.next.(interface{ Close() error }); ok {
			err = c.Close()
		}
	})
	return err
}

// cleanupLoop runs periodically to purge expired deduplication keys.
func (d *Dedup) cleanupLoop() {
	defer d.wg.Done()
	ticker := time.NewTicker(d.cleanupEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			now := time.Now().UnixNano()
			for i := 0; i < len(d.shards); i++ {
				shard := &d.shards[i]
				shard.mu.Lock()
				d.cleanupShardLocked(shard, now)
				shard.mu.Unlock()
			}
		case <-d.done:
			return
		}
	}
}

// cleanupShardLocked removes expired keys from a shard (caller must hold lock).
func (d *Dedup) cleanupShardLocked(shard *dedupShard, now int64) {
	for k, exp := range shard.seen {
		if now > exp {
			delete(shard.seen, k)
		}
	}
}

// defaultDedup implements the default deduplication key calculation.
type defaultDedup struct {
	ignoreFields map[string]struct{}
}

// NewDefaultDedup creates a new default deduplication key generator.
func NewDefaultDedup() lx.Deduper {
	return &defaultDedup{ignoreFields: make(map[string]struct{})}
}

// Calculate generates a deduplication key from level, message, namespace, and
// fields.  Fields are sorted before hashing so that identical entries always
// produce the same key regardless of Go map iteration order.
func (d *defaultDedup) Calculate(e *lx.Entry) uint64 {
	h := xxhash.New()
	zero := []byte{0}

	h.Write([]byte(e.Level.String()))
	h.Write(zero)
	h.Write([]byte(e.Message))
	h.Write(zero)
	h.Write([]byte(e.Namespace))
	h.Write(zero)

	if len(e.Fields) > 0 {
		m := e.Fields.Map()
		keys := make([]string, 0, len(m))
		for k := range m {
			if _, excluded := d.ignoreFields[k]; !excluded {
				keys = append(keys, k)
			}
		}
		// Sort keys to guarantee a deterministic hash across calls.
		// Without this, Go's random map iteration order means two identical
		// entries can hash to different values and bypass deduplication.
		sort.Strings(keys)

		buf := dedupBufPool.Get().(*bytes.Buffer)
		buf.Reset()
		defer dedupBufPool.Put(buf)

		for _, k := range keys {
			buf.WriteString(k)
			buf.WriteByte('=')
			fmt.Fprint(buf, m[k])
			buf.WriteByte(0)
		}
		h.Write(buf.Bytes())
	}

	return h.Sum64()
}

var dedupBufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}
