package registry

import (
	"sync"
	"time"

	hll "github.com/axiomhq/hyperloglog"
)

// Cardinality implements a sliding-window of HyperLogLog sketches.
// Example (staleTime = 15m, sketchDuration = 5m): sketchesLength = ceil(15/5) = 4 sketches
//
//		          cachedMerge = S0 + S1 + S2
//		                   ┌─────────────┐
//		                   ▼             │
//		┌─────┬─────┬─────┬─────┐
//		│ S0  │ S1  │ S2  │ S3  │  ring of HyperLogLog sketches
//		└─▲───┴─▲───┴─▲───┴─▲───┘
//		  │     │     │     │
//		 oldest              current
//
//	 Stats per sketch depending on precision:
//	 p (precision)   m=2**p (registers)   memory (KB)   standard error (%)
//	             4                   16          0.02                26.00
//	             5                   32          0.03                18.38
//	             6                   64          0.06                13.00
//	             7                  128          0.13                 9.19
//	             8                  256          0.26                 6.50
//	             9                  512          0.51                 4.60
//	            10                1,024          1.02                 3.25
//	            11                2,048          2.05                 2.30
//	            12                4,096          4.10                 1.62
//	            13                8,192          8.19                 1.15
//	            14               16,384         16.38                 0.81
//	            15               32,768         32.77                 0.57
//	            16               65,536         65.54                 0.41
//	            17              131,072        131.07                 0.29
//	            18              262,144        262.14                 0.20
//
// Note that we use sparse mode, so actual memory usage is lower for small cardinalities.
type Cardinality struct {
	mu             sync.RWMutex
	sketches       []*hll.Sketch
	sketchDuration time.Duration
	sketchesLength int
	precision      uint8
	current        int
	cachedMerge    *hll.Sketch
	lastAdvance    time.Time
}

// NewCardinality creates a new Cardinality estimate with a 3.25 standard error.
func NewCardinality(staleTime, sketchDuration time.Duration) *Cardinality {
	return newCardinality(10, staleTime, sketchDuration)
}

func newCardinality(precision uint8, staleTime, sketchDuration time.Duration) *Cardinality {
	// If parameters are out of bounds, set defaults.
	if precision < 4 || precision > 18 {
		precision = 14
	}
	if staleTime <= 1*time.Minute || sketchDuration < 1*time.Minute || staleTime < sketchDuration {
		staleTime = 15 * time.Minute
		sketchDuration = 5 * time.Minute
	}

	sketchesLength := int((staleTime + sketchDuration) / sketchDuration) // ceil
	if sketchesLength < 2 {
		sketchesLength = 2
	}

	sketches := make([]*hll.Sketch, sketchesLength)
	for i := range sketches {
		sketches[i], _ = hll.NewSketch(precision, true)
	}

	cachedMerge, _ := hll.NewSketch(precision, true)
	return &Cardinality{
		sketchDuration: sketchDuration,
		sketchesLength: sketchesLength,
		current:        0,
		sketches:       sketches,
		cachedMerge:    cachedMerge,
		lastAdvance:    time.Now(),
		precision:      precision,
	}
}

func (c *Cardinality) Insert(hash uint64) {
	c.mu.Lock()
	c.sketches[c.current].InsertHash(hash)
	c.mu.Unlock()
}

// Estimate returns the estimated cardinality over the last staleTime window.
func (c *Cardinality) Estimate() uint64 {
	c.mu.RLock()
	window := c.cachedMerge.Clone()
	current := c.sketches[c.current].Clone()
	c.mu.RUnlock()
	_ = window.Merge(current)
	return window.Estimate()
}

// Advance should be called by an external ticker every sketchDuration to advance the ring.
// It advances at least 1 step per call.
func (c *Cardinality) Advance() {
	c.mu.Lock()
	delta := time.Since(c.lastAdvance)
	steps := 0
	if delta < c.sketchDuration {
		steps = 1
	} else {
		steps = int(delta / c.sketchDuration)
	}
	// Cap steps so we don't loop excessively after very long pauses.
	if steps > c.sketchesLength {
		steps = c.sketchesLength
	}

	for i := 0; i < steps; i++ {
		// advance ring
		c.current++
		if c.current >= c.sketchesLength {
			c.current = 0
		}
		// reset new current sketch
		c.sketches[c.current], _ = hll.NewSketch(c.precision, true)
		c.lastAdvance = c.lastAdvance.Add(c.sketchDuration)
	}

	// Recompute cached merge of all non-current sketches
	cachedMerge, _ := hll.NewSketch(c.precision, true)
	for i := 0; i < c.sketchesLength; i++ {
		if i == c.current {
			continue
		}
		_ = cachedMerge.Merge(c.sketches[i])
	}
	c.cachedMerge = cachedMerge
	c.mu.Unlock()
}
