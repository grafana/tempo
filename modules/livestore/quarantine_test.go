package livestore

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuarantine_ReclaimBeforeDeadlineKeepsEntry(t *testing.T) {
	q := newQuarantine(time.Hour)

	called := false
	q.add(uuid.New(), "tenant", "wal", func() error {
		called = true
		return nil
	})

	results := q.reclaim()
	assert.Nil(t, results, "no results before deadline")
	assert.False(t, called, "reclaim fn must not be called before deadline")
	assert.Len(t, q.entries, 1, "entry must remain in quarantine")
}

func TestQuarantine_ReclaimAfterDeadlineRunsFnAndRemovesEntry(t *testing.T) {
	// Negative grace ⇒ deadline is already in the past.
	q := newQuarantine(-time.Second)

	id := uuid.New()
	called := 0
	q.add(id, "tenant-a", "wal", func() error {
		called++
		return nil
	})

	results := q.reclaim()
	require.Len(t, results, 1)
	assert.Equal(t, id, results[0].BlockID)
	assert.Equal(t, "tenant-a", results[0].Tenant)
	assert.Equal(t, "wal", results[0].BlockType)
	assert.NoError(t, results[0].Err)
	assert.Equal(t, 1, called)
	assert.Empty(t, q.entries, "entry must be dropped after reclaim")

	// Second reclaim is a no-op — entries are not retried.
	results = q.reclaim()
	assert.Nil(t, results)
	assert.Equal(t, 1, called, "fn must not be retried on subsequent reclaim calls")
}

func TestQuarantine_ReclaimDropsEntryEvenOnError(t *testing.T) {
	// Documents the explicit comment in quarantine.go: "a stuck deletion is
	// not retried" — startup sweep is responsible for cleanup if reclaim
	// errors.
	q := newQuarantine(-time.Second)

	wantErr := errors.New("disk full")
	q.add(uuid.New(), "tenant", "complete", func() error { return wantErr })

	results := q.reclaim()
	require.Len(t, results, 1)
	assert.ErrorIs(t, results[0].Err, wantErr)
	assert.Empty(t, q.entries, "errored entry must still be dropped")
}

func TestQuarantine_ReclaimMixedDeadlines(t *testing.T) {
	q := newQuarantine(time.Hour) // entries default to a future deadline

	dueID := uuid.New()
	notDueID := uuid.New()

	dueCalled := false
	notDueCalled := false

	// Add a "due" entry by overriding the deadline directly.
	q.entries = append(q.entries, quarantineEntry{
		blockID:   dueID,
		tenant:    "tenant",
		blockType: "wal",
		deadline:  time.Now().Add(-time.Second),
		reclaim:   func() error { dueCalled = true; return nil },
	})
	q.add(notDueID, "tenant", "wal", func() error { notDueCalled = true; return nil })

	results := q.reclaim()
	require.Len(t, results, 1)
	assert.Equal(t, dueID, results[0].BlockID)
	assert.True(t, dueCalled)
	assert.False(t, notDueCalled, "future-deadline entry must remain")

	require.Len(t, q.entries, 1)
	assert.Equal(t, notDueID, q.entries[0].blockID)
}

// TestQuarantine_ConcurrentAddAndReclaim verifies that reclaim does not hold
// the mutex while invoking fn — otherwise concurrent adds would deadlock.
// Run with -race.
func TestQuarantine_ConcurrentAddAndReclaim(t *testing.T) {
	q := newQuarantine(-time.Second) // everything is immediately due

	const adds = 200

	// Block reclaim's fn long enough that an add() must succeed concurrently
	// while reclaim is still running.
	started := make(chan struct{})
	gate := make(chan struct{})
	q.add(uuid.New(), "blocker", "wal", func() error {
		close(started)
		<-gate
		return nil
	})

	var wg sync.WaitGroup
	wg.Add(2)

	// Reclaim goroutine: will block inside fn until gate closes.
	var reclaimResults []ReclaimResult
	go func() {
		defer wg.Done()
		reclaimResults = q.reclaim()
	}()

	// Adder goroutine: must not deadlock while reclaim is mid-fn.
	go func() {
		defer wg.Done()
		<-started
		for range adds {
			q.add(uuid.New(), "tenant", "wal", func() error { return nil })
		}
		close(gate) // unblock reclaim
	}()

	wg.Wait()

	require.Len(t, reclaimResults, 1)
	assert.Equal(t, "blocker", reclaimResults[0].Tenant)
	assert.Len(t, q.entries, adds, "all concurrent adds must be present")
}
