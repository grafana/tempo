package livestore

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// reclaimFn deletes the on-disk files for a quarantined block.
type reclaimFn func() error

type quarantineEntry struct {
	blockID   uuid.UUID
	tenant    string
	blockType string // "wal" or "complete"
	deadline  time.Time
	reclaim   reclaimFn
}

// ReclaimResult is the outcome of one reclaim attempt, used for logs/metrics.
type ReclaimResult struct {
	BlockID   uuid.UUID
	Tenant    string
	BlockType string
	Err       error
}

// quarantine holds blocks pending file reclamation.
type quarantine struct {
	mtx     sync.Mutex
	entries []quarantineEntry
	grace   time.Duration
}

func newQuarantine(grace time.Duration) *quarantine {
	return &quarantine{grace: grace}
}

func (q *quarantine) add(blockID uuid.UUID, tenant, blockType string, fn reclaimFn) {
	q.mtx.Lock()
	defer q.mtx.Unlock()
	q.entries = append(q.entries, quarantineEntry{
		blockID:   blockID,
		tenant:    tenant,
		blockType: blockType,
		deadline:  time.Now().Add(q.grace),
		reclaim:   fn,
	})
}

// reclaim runs fn for entries whose deadline has passed. Entries are dropped
// after attempt — a stuck deletion is not retried.
func (q *quarantine) reclaim() []ReclaimResult {
	q.mtx.Lock()
	now := time.Now()
	// Allocate a fresh keep slice (rather than reslicing q.entries[:0]) so
	// dropped entries don't sit in the backing array's hidden capacity and
	// keep their captured WAL/complete-block objects from GC.
	var keep []quarantineEntry
	var due []quarantineEntry
	for _, e := range q.entries {
		if !now.Before(e.deadline) {
			due = append(due, e)
		} else {
			keep = append(keep, e)
		}
	}
	q.entries = keep
	q.mtx.Unlock()

	if len(due) == 0 {
		return nil
	}
	results := make([]ReclaimResult, 0, len(due))
	for _, e := range due {
		results = append(results, ReclaimResult{
			BlockID:   e.blockID,
			Tenant:    e.tenant,
			BlockType: e.blockType,
			Err:       e.reclaim(),
		})
	}
	return results
}
