package registry

import (
	"encoding/binary"
	"sync"
	"time"

	hll "github.com/axiomhq/hyperloglog"
)

// HLLCounter maintains a sliding distinct counter using 1-minute buckets.
// - Touch() adds to the current minute bucket.
// - Advance() rotates buckets if time has moved forward (safe to call anytime).
// - Estimate() returns the union of the last 'staleTime' worth of minutes.
type HLLCounter struct {
	mu           sync.RWMutex
	lastReset    int64         // start of current bucket in unix millis (aligned to minute)
	sk           *hll.Sketch   // alias to buckets[cur] (current minute)
	prevEst      uint64        // kept for compatibility; unused in sliding logic
	minuteMillis int64         // constant = 60_000
	windowMin    int           // number of 1-minute buckets in the window (>=1)
	cur          int           // current bucket index in the ring
	buckets      []*hll.Sketch // ring of 1-minute HLL buckets
}

// NewHLLCounter creates a sliding counter with 1-minute buckets covering 'staleTime'.
// For example, staleTime=15m -> 15 buckets. Values <1m are rounded up to 1 bucket.
func NewHLLCounter(staleTime time.Duration) *HLLCounter {
	if staleTime <= 0 {
		staleTime = time.Minute
	}
	windowMin := int((staleTime + time.Minute - 1) / time.Minute) // ceil to whole minutes
	if windowMin < 1 {
		windowMin = 1
	}

	aligned := time.Now().Truncate(time.Minute)
	lastReset := aligned.UnixMilli()

	buckets := make([]*hll.Sketch, windowMin)
	for i := range buckets {
		buckets[i] = hll.New() // p=14 by default; sparse enabled
	}

	c := &HLLCounter{
		lastReset:    lastReset,
		sk:           buckets[0],
		minuteMillis: int64(time.Minute / time.Millisecond),
		windowMin:    windowMin,
		cur:          0,
		buckets:      buckets,
	}
	return c
}

// Touch inserts a key (64-bit hash) into the current minute bucket.
func (c *HLLCounter) Touch(hash uint64) {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], hash)

	nowMs := time.Now().UnixMilli()
	c.mu.Lock()
	c.advanceTo(nowMs)
	c.sk.Insert(buf[:]) // axiomhq/hyperloglog hashes internally for []byte
	c.mu.Unlock()
}

// Estimate returns the estimated distinct count over the last 'staleTime'.
// It unions the last 'windowMin' 1-minute buckets.
func (c *HLLCounter) Estimate() uint64 {
	nowMs := time.Now().UnixMilli()

	c.mu.Lock()
	c.advanceTo(nowMs)

	// Accumulate into a fresh sketch to avoid mutating buckets.
	acc := hll.New()
	for i := 0; i < c.windowMin; i++ {
		idx := c.cur - i
		if idx < 0 {
			idx += c.windowMin
		}
		_ = acc.Merge(c.buckets[idx])
	}
	c.mu.Unlock()

	return acc.Estimate()
}

// Advance rotates minute buckets if time has moved forward.
// You can call this from a 1-minute ticker, but Touch/Estimate also call it.
func (c *HLLCounter) Advance() {
	nowMs := time.Now().UnixMilli()
	c.mu.Lock()
	c.advanceTo(nowMs)
	c.mu.Unlock()
}

// --- internals ---

// advanceTo rotates one-minute buckets for each full minute elapsed since lastReset.
// For each step: move to the next slot and replace it with a fresh empty sketch.
func (c *HLLCounter) advanceTo(nowMs int64) {
	for nowMs >= c.lastReset+c.minuteMillis {
		// Move to next bucket
		c.cur++
		if c.cur >= c.windowMin {
			c.cur = 0
		}
		// Reset the new current bucket
		c.buckets[c.cur] = hll.New()
		c.sk = c.buckets[c.cur]

		// Advance the aligned time by exactly one minute
		c.lastReset += c.minuteMillis
	}
}
