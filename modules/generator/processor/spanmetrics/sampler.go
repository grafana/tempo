package spanmetrics

import (
	"math/rand/v2"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
)

// The sampler bounds how many spans per metric series run the full
// label-building and metric-update path. Spans that are not selected never
// reach the registry at all, and the selected ones carry a multiplier that
// scales their contributions back up.
//
// Selection is stratified, not random: each series' arrivals are cut into
// consecutive blocks of blockSize spans and exactly one span per block, chosen
// uniformly at random, is kept. Compared with keeping each span independently
// with probability 1/blockSize, this
//
//   - pins _count to within one block of the truth instead of leaving it with a
//     sqrt(samples) sampling error, and
//   - gives every kept span the same weight, which is what minimizes the
//     variance of the _sum and per-bucket estimates for a given number of
//     kept spans.
//
// The random position within each block is what keeps the estimates unbiased
// when arrival order correlates with latency (bursts of slow spans, retry
// storms). Picking a fixed position instead would be cheaper but would inherit
// any periodicity in the stream.
const (
	// samplerWindow is how often each series re-estimates its span rate and
	// resizes its blocks. Short enough to track a ramp within a collection
	// interval, long enough that the budget below is a meaningful sample.
	samplerWindow = 10 * time.Second

	// samplerShards splits the series map so concurrent pushes for one tenant
	// do not serialize on a single mutex. Must be a power of two.
	samplerShards   = 64
	samplerShardMsk = samplerShards - 1

	// samplerMaxSeriesPerShard is the size at which a shard sweeps out series
	// that have gone quiet, not a hard cap: a shard whose series are all active
	// keeps growing, bounded instead by the registry's own active-series limit.
	// Sweeping an active series is harmless anyway -- it restarts at blockSize
	// 1, which costs CPU for a window but never accuracy.
	samplerMaxSeriesPerShard = 16384
	samplerEvictAfter        = 5 * samplerWindow
)

// seriesSampler tracks a per-series span budget. It is safe for concurrent use.
type seriesSampler struct {
	// budget is the number of spans to keep per series per samplerWindow.
	budget   uint64
	windowMs int64

	shards [samplerShards]samplerShard
}

type samplerShard struct {
	mtx    sync.Mutex
	series map[uint64]*samplerSeries
	rng    *rand.Rand
	// evictAtMs is the next timestamp at which an over-capacity shard may
	// sweep. Without it a shard that stays over capacity would sweep on every
	// push.
	evictAtMs int64
	// seenSpans and keptSpans track how much the sampler is actually cutting.
	// Kept under the shard mutex rather than in atomics so the hot path pays
	// nothing extra for them.
	seenSpans uint64
	keptSpans uint64
}

type samplerSeries struct {
	// blockSize is the number of spans this series' arrivals are grouped into;
	// 1 keeps every span. nextBlockSize takes effect at the next block
	// boundary, so a block in flight always completes at the size it started
	// with and yields exactly one kept span.
	blockSize     uint64
	nextBlockSize uint64
	// pos is how far into the current block we are and pick is the position in
	// it that gets kept.
	pos  uint64
	pick uint64
	// seen counts spans since windowStartMs.
	seen          uint64
	windowStartMs int64
	lastSeenMs    int64
}

// newSeriesSampler returns a sampler that keeps at most maxSpansPerSecond spans
// per series per second, or nil if sampling is disabled.
func newSeriesSampler(maxSpansPerSecond int) *seriesSampler {
	if maxSpansPerSecond <= 0 {
		return nil
	}

	windowMs := samplerWindow.Milliseconds()
	budget := uint64(maxSpansPerSecond) * uint64(samplerWindow/time.Second)

	s := &seriesSampler{budget: budget, windowMs: windowMs}
	for i := range s.shards {
		s.shards[i].series = make(map[uint64]*samplerSeries)
		// Seeding per shard from the global source keeps the picks independent
		// across shards without sharing a locked global generator on the hot
		// path; the shard mutex already serializes access to this one.
		s.shards[i].rng = rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	}
	return s
}

// sample reports the multiplier to apply to the span's contributions, or 0 if
// the span should be skipped. A series is always sampled at the same rate
// regardless of key collisions, so a collision costs accuracy for the two
// series that share a key, never correctness: a kept span is still attributed
// to whatever series its real label set resolves to.
func (s *seriesSampler) sample(key uint64, nowMs int64) float64 {
	sh := &s.shards[key&samplerShardMsk]

	sh.mtx.Lock()
	defer sh.mtx.Unlock()

	series, ok := sh.series[key]
	if !ok {
		// A series is seen for the first time, or after an eviction: keep every
		// span until the first window boundary produces a rate estimate. The
		// burst guard below stops that from being unbounded.
		series = &samplerSeries{blockSize: 1, nextBlockSize: 1, windowStartMs: nowMs}
		sh.series[key] = series
		sh.maybeEvict(nowMs)
	}

	series.lastSeenMs = nowMs
	series.seen++
	sh.seenSpans++

	if nowMs-series.windowStartMs >= s.windowMs {
		series.rollWindow(nowMs, s.windowMs, s.budget)
	}

	if series.pos == 0 {
		// Always reset: a pick left over from a larger previous block would sit
		// past the end of this one, and the series would then keep nothing at
		// all rather than one span per block.
		series.pick = 0
		if series.blockSize > 1 {
			series.pick = sh.rng.Uint64N(series.blockSize)
		}
	}

	multiplier := 0.0
	if series.pos == series.pick {
		multiplier = float64(series.blockSize)
		sh.keptSpans++
	}

	series.pos++
	if series.pos >= series.blockSize {
		series.pos = 0
		// nextBlockSize is what the previous window's rate implies, which is the
		// right size while the rate holds. When this window is running hotter --
		// a burst, or a series in its very first window with no estimate at all
		// -- size the block from what has already arrived instead. That keeps
		// the block, and so the count error it bounds, at seen/budget however
		// large the burst gets, while letting only about budget*ln(seen/budget)
		// spans through.
		blockSize := series.nextBlockSize
		if burst := series.seen / s.budget; burst > blockSize {
			blockSize = burst
		}
		series.blockSize = blockSize
	}

	return multiplier
}

// counts reports how many spans the sampler has seen and how many of those ran
// the full aggregation path. The two are read shard by shard rather than under
// one lock, so a concurrent push can land between shards; the ratio is meant
// for reporting how hard sampling is biting, not for exact accounting.
func (s *seriesSampler) counts() (seen, kept uint64) {
	for i := range s.shards {
		sh := &s.shards[i]
		sh.mtx.Lock()
		seen += sh.seenSpans
		kept += sh.keptSpans
		sh.mtx.Unlock()
	}
	return seen, kept
}

// rollWindow re-estimates the series' span rate and sizes the next block from
// it. The new size is staged in nextBlockSize rather than applied here so the
// block in flight still yields exactly one kept span.
func (s *samplerSeries) rollWindow(nowMs, windowMs int64, budget uint64) {
	elapsedMs := nowMs - s.windowStartMs
	next := uint64(1)
	if elapsedMs > 0 && s.seen > 0 {
		// Spans this series would produce in a full window at the rate just
		// observed. elapsedMs can be much longer than one window when the
		// series went quiet, which correctly scales the estimate down.
		perWindow := float64(s.seen) * float64(windowMs) / float64(elapsedMs)
		if size := uint64(perWindow / float64(budget)); size > 1 {
			next = size
		}
	}

	s.nextBlockSize = next
	s.seen = 0
	s.windowStartMs = nowMs
}

// maybeEvict drops series that have gone quiet once a shard grows past its cap.
// Called with the shard mutex held.
func (sh *samplerShard) maybeEvict(nowMs int64) {
	if len(sh.series) <= samplerMaxSeriesPerShard || nowMs < sh.evictAtMs {
		return
	}
	sh.evictAtMs = nowMs + samplerEvictAfter.Milliseconds()

	staleBefore := nowMs - samplerEvictAfter.Milliseconds()
	for key, series := range sh.series {
		if series.lastSeenMs < staleBefore {
			delete(sh.series, key)
		}
	}
}

// samplerKey accumulates the values that determine a span's metric series into
// a scratch buffer, hashed once to produce the sampling key. It deliberately
// works from raw attribute values rather than a built label set: building,
// sorting, sanitizing and hashing a label set is the cost the sampler exists to
// avoid, so the key has to be decidable before any of it happens.
//
// The key is therefore finer than the label set. Label sanitization and the
// per-label cardinality limiter can merge distinct raw values into one series
// after the fact, and those merged series get one budget per raw value instead
// of one in total. That over-samples them, which costs CPU rather than
// accuracy — notably for the high-cardinality span names that span-name
// sanitization exists to collapse, where the sampler will not help.
type samplerKey struct {
	buf []byte
}

func (k *samplerKey) reset() {
	k.buf = k.buf[:0]
}

// addString appends a value and a separator. A value that itself contains the
// separator byte can collide with a different split of the same bytes, which
// only shifts a sampling rate.
func (k *samplerKey) addString(v string) {
	k.buf = append(k.buf, v...)
	k.buf = append(k.buf, 0)
}

func (k *samplerKey) addByte(v byte) {
	k.buf = append(k.buf, v, 0)
}

func (k *samplerKey) sum() uint64 {
	return xxhash.Sum64(k.buf)
}
