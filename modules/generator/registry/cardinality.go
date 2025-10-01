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
	precision      uint8         // HLL precision (number of registers = 2^precision)
}

//  Per Sketch:
//  p (precision)   m=2**p (registers)   memory (KB)   standard error (%)
//              4                   16          0.02                26.00
//              5                   32          0.03                18.38
//              6                   64          0.06                13.00
//              7                  128          0.13                 9.19
//              8                  256          0.26                 6.50
//              9                  512          0.51                 4.60
//             10                1,024          1.02                 3.25
//             11                2,048          2.05                 2.30
//             12                4,096          4.10                 1.62
//             13                8,192          8.19                 1.15
//             14               16,384         16.38                 0.81
//             15               32,768         32.77                 0.57
//             16               65,536         65.54                 0.41
//             17              131,072        131.07                 0.29
//             18              262,144        262.14                 0.20
// size: 16KB * 5 Sketches = 80KB per HLLCounter (constant with inserts)
// note that we use sparse mode, so actual memory usage is lower for small cardinalities.

// NewHLLCounter creates a sliding-window HLL counter with default precision p=14.
// Use NewHLLCounterWithPrecision to specify a custom precision.
func NewHLLCounter(staleTime, bucketDuration time.Duration) *HLLCounter {
	return NewHLLCounterWithPrecision(14, staleTime, bucketDuration)
}

// NewHLLCounterWithPrecision creates a sliding-window HLL counter with the given precision.
// Precision must be in [4,18]. Values outside this range are clamped to 14.
func NewHLLCounterWithPrecision(precision uint8, staleTime, bucketDuration time.Duration) *HLLCounter {
	if precision < 4 || precision > 18 {
		precision = 14
	}
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
		buckets[i] = newHLLSketch(precision)
	}
	u := newHLLSketch(precision) // initial union of non-current buckets (all empty)
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
		precision:      precision,
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
		c.buckets[c.cur] = newHLLSketch(c.precision)
		c.lastFlipMs += bucketMs
	}

	// Recompute cached union of all non-current buckets (O(windowMin) merges).
	u := newHLLSketch(c.precision)
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

// newHLLSketch returns a new HLL sketch with the given precision and sparse mode enabled.
func newHLLSketch(p uint8) *hll.Sketch {
	if p < 4 || p > 18 {
		p = 14
	}
	sk, err := hll.NewSketch(p, true)
	if err != nil {
		// Fallback to default if precision is invalid for any reason
		return hll.New()
	}
	return sk
}
