package registry

import (
	"sync"
	"time"

	"github.com/go-kit/log/level"

	hll "github.com/axiomhq/hyperloglog"
	tempo_log "github.com/grafana/tempo/pkg/util/log"
)

// HLLCounter: sliding HyperLogLog counter with fixed-width time window.
// - Insert(hash): insert into current bucket (cheap).
// - Advance(): called by an external ticker; flips buckets and recomputes cached union.
// - Estimate(): 1 merge (cached union + current) → fast, suitable for ~15s cadence.
//
// Example (staleTime = 15m, bucketDuration = 5m): windowMin = ceil(15/5) + 1 = 4 buckets
//
//	          cachedUnion = B0 + B1 + B2
//	                   ┌─────────────┐
//	                   ▼             │
//	┌─────┬─────┬─────┬─────┐
//	│ B0  │ B1  │ B2  │ B3  │  ring buffer of HyperLogLog sketches
//	└─▲───┴─▲───┴─▲───┴─▲───┘
//	  │     │     │     │
//	 oldest              current (cur)
//
// On Advance(): move cur → next slot (wrapping), reset that sketch, then rebuild cachedUnion
// from the remaining windows. The extra bucket guarantees data survive for at least staleTime.
type HLLCounter struct {
	mu             sync.RWMutex
	bucketDuration time.Duration
	windowMin      int           // ring length (>=2) including one extra bucket to avoid early drops
	cur            int           // current bucket index
	buckets        []*hll.Sketch // ring of bucket sketches
	cachedUnion    *hll.Sketch   // union of all NON-current buckets (rebuilt on tick)
	lastFlipMs     int64         // last time we flipped (unix ms), used to catch up if ticks were delayed
}

func NewHLLCounter(staleTime, bucketDuration time.Duration) *HLLCounter {
	if bucketDuration <= 0 {
		bucketDuration = staleTime
	}
	if staleTime <= 0 {
		staleTime = bucketDuration
	}
	windowMin := int((staleTime + bucketDuration - 1) / bucketDuration) // ceil
	if windowMin < 1 {
		windowMin = 1
	}
	// Overprovision the ring with one extra bucket so items stay resident for at
	// least staleTime before being recycled.
	windowMin++

	buckets := make([]*hll.Sketch, windowMin)
	for i := range buckets {
		buckets[i] = hll.New() // default p=14 (~16 KB dense), sparse enabled
	}
	u := hll.New() // initial union of non-current buckets (all empty)
	for i := 1; i < windowMin; i++ {
		_ = u.Merge(buckets[i])
	}

	return &HLLCounter{
		bucketDuration: bucketDuration,
		windowMin:      windowMin,
		cur:            0,
		buckets:        buckets,
		cachedUnion:    u,
		lastFlipMs:     time.Now().UnixMilli(),
	}
}

func (c *HLLCounter) Insert(hash uint64) {
	c.mu.Lock()
	c.buckets[c.cur].InsertHash(hash)
	c.mu.Unlock()
}

// Estimate returns the estimated distincts over the last staleTime window.
// Cheap: merge cachedUnion (static during the bucket interval) with a Clone of the current bucket.
func (c *HLLCounter) Estimate() uint64 {
	c.mu.RLock()
	u := c.cachedUnion // stable pointer between ticks
	curClone := c.buckets[c.cur].Clone()
	c.mu.RUnlock()

	acc := u.Clone() // no lock needed beyond this point
	_ = acc.Merge(curClone)
	return acc.Estimate()
}

// Advance should be called by an external ticker.
// Not guaranteed to align to wall-clock minutes; we catch up for missed minutes.
func (c *HLLCounter) Advance() {
	bucketMs := int64(c.bucketDuration / time.Millisecond)
	nowMs := time.Now().UnixMilli()

	c.mu.Lock()
	delta := nowMs - c.lastFlipMs
	if delta < bucketMs {
		c.mu.Unlock()
		return
	}

	steps := int(delta / bucketMs)
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
		c.lastFlipMs += bucketMs
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

	level.Debug(tempo_log.Logger).Log("msg", "hll counter advanced", "steps", steps)
}
