package livestore

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// reclaimFn deletes the on-disk files for a block. Invoked by the quarantine
// after the grace window has elapsed so in-flight readers don't see ENOENT
// on page files.
type reclaimFn func() error

type quarantineEntry struct {
	blockID   uuid.UUID
	tenant    string
	blockType string // "wal" or "complete"
	deadline  time.Time
	reclaim   reclaimFn
}

// ReclaimResult describes the outcome of reclaiming a single quarantined
// block. Caller uses this for logging + metrics.
type ReclaimResult struct {
	BlockID   uuid.UUID
	Tenant    string
	BlockType string
	Err       error
}

// quarantine holds blocks pending file reclamation. Per-instance; serialized
// on its own mutex.
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

// reclaim runs fn for every entry whose deadline has passed and returns one
// ReclaimResult per entry attempted. Entries are dropped after attempt
// regardless — a stuck deletion doesn't get retried forever.
func (q *quarantine) reclaim() []ReclaimResult {
	q.mtx.Lock()
	now := time.Now()
	keep := q.entries[:0]
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

// drain runs fn for every entry regardless of deadline. Used at shutdown.
func (q *quarantine) drain() []ReclaimResult {
	q.mtx.Lock()
	due := q.entries
	q.entries = nil
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

func (q *quarantine) pendingCount() int {
	q.mtx.Lock()
	defer q.mtx.Unlock()
	return len(q.entries)
}
