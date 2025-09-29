package registry

import (
	"sync"
	"time"

	hll "github.com/axiomhq/hyperloglog"
)

// HLLCounter: sliding distincts using 1-minute buckets.
// - Touch(hash): insert into current minute bucket (cheap).
// - MinuteTick(now): called by an external 1-minute ticker; flips buckets and recomputes cached union.
// - Estimate(): 1 merge (cached union + current) → fast, suitable for ~15s cadence.
type HLLCounter struct {
	mu          sync.RWMutex
	windowMin   int           // number of 1-minute buckets in the window (>=1)
	cur         int           // current bucket index
	buckets     []*hll.Sketch // ring of minute buckets
	cachedUnion *hll.Sketch   // union of all NON-current buckets (rebuilt on tick)
	lastFlipMs  int64         // last time we flipped (unix ms), used to catch up if ticks were delayed
}

func NewHLLCounter(staleTime time.Duration) *HLLCounter {
	if staleTime <= 0 {
		staleTime = time.Minute
	}
	windowMin := int((staleTime + time.Minute - 1) / time.Minute) // ceil
	if windowMin < 1 {
		windowMin = 1
	}

	buckets := make([]*hll.Sketch, windowMin)
	for i := range buckets {
		buckets[i] = hll.New() // default p=14 (~16 KB dense), sparse enabled
	}
	u := hll.New() // initial union of non-current buckets (all empty)
	for i := 1; i < windowMin; i++ {
		_ = u.Merge(buckets[i])
	}

	return &HLLCounter{
		windowMin:   windowMin,
		cur:         0,
		buckets:     buckets,
		cachedUnion: u,
		lastFlipMs:  time.Now().UnixMilli(),
	}
}

// Touch inserts a pre-hashed 64-bit key into the current minute bucket.
func (c *HLLCounter) Touch(hash uint64) {
	c.mu.Lock()
	c.buckets[c.cur].InsertHash(hash)
	c.mu.Unlock()
}

// Estimate returns the estimated distincts over the last staleTime window.
// Cheap: merge cachedUnion (static during the minute) with a Clone of the current bucket.
func (c *HLLCounter) Estimate() uint64 {
	c.mu.RLock()
	u := c.cachedUnion // stable pointer between ticks
	curClone := c.buckets[c.cur].Clone()
	c.mu.RUnlock()

	acc := u.Clone() // no lock needed beyond this point
	_ = acc.Merge(curClone)
	return acc.Estimate()
}

// MinuteTick should be called by an external 1-minute ticker.
// Not guaranteed to align to wall-clock minutes; we catch up for missed minutes.
func (c *HLLCounter) MinuteTick() {
	const minuteMs = int64(time.Minute / time.Millisecond)
	nowMs := time.Now().UnixMilli()

	c.mu.Lock()
	steps := int((nowMs - c.lastFlipMs) / minuteMs)
	if steps < 1 {
		steps = 1 // at least one flip per scheduled tick
	}
	// Cap steps so we don't loop excessively after very long pauses.
	if steps > c.windowMin {
		steps = c.windowMin
	}

	for i := 0; i < steps; i++ {
		// advance ring
		c.cur++
		if c.cur >= c.windowMin {
			c.cur = 0
		}
		// reset new current bucket
		c.buckets[c.cur] = hll.New()
		c.lastFlipMs += minuteMs
	}

	// Recompute cached union of all non-current buckets (O(windowMin) merges).
	u := hll.New()
	for i := 0; i < c.windowMin; i++ {
		if i == c.cur {
			continue
		}
		_ = u.Merge(c.buckets[i])
	}
	c.cachedUnion = u
	c.mu.Unlock()
}
