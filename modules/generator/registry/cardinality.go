package registry

import (
	"encoding/binary"
	"sync"
	"time"

	hll "github.com/axiomhq/hyperloglog"
)

// HLLCounter maintains a single 15-minute tumbling HLL.
type HLLCounter struct {
	mu        sync.RWMutex
	lastReset int64       // start of current window (aligned)
	sk        *hll.Sketch // current (open) sketch
	prevEst   uint64      // last *completed* window's estimate
}

func NewHLLCounter() *HLLCounter {
	return &HLLCounter{
		lastReset: time.Now().UnixMilli(),
		sk:        hll.New(),
	}
}

// Touch inserts a key observed at time t into the current window.
func (c *HLLCounter) Touch(hash uint64) {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], hash)
	c.mu.Lock()
	c.sk.Insert(buf[:])
	c.mu.Unlock()
}

// LastComplete returns the estimate for the last *completed* window.
func (c *HLLCounter) Estimate() uint64 {
	c.mu.RLock()
	est := c.prevEst
	c.mu.RUnlock()
	return est
}

func (c *HLLCounter) Advance() {
	c.prevEst = c.sk.Estimate()
	c.sk = hll.New() // fresh empty sketch
	c.lastReset = time.Now().UnixMilli()
}
