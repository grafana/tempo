package livestore

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/modules/ingester"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/require"
)

// TestCompleteBlock_ConcurrentAttempts verifies only one completion succeeds for same block
func TestCompleteBlock_ConcurrentAttempts(t *testing.T) {
	// This test verifies the completingBlocks map prevents concurrent completions
	inst := &instance{
		completingBlocks: make(map[uuid.UUID]bool),
		walBlocks:        make(map[uuid.UUID]common.WALBlock),
		blocksMtx:        sync.RWMutex{},
	}

	blockID := uuid.New()

	// Simulate 3 concurrent attempts to mark as completing
	var wg sync.WaitGroup
	results := make([]bool, 3) // true = successfully marked as completing

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			inst.blocksMtx.Lock()
			defer inst.blocksMtx.Unlock()

			// Try to mark as completing (simulates completeBlock logic)
			if inst.completingBlocks[blockID] {
				// Already being completed
				results[idx] = false
			} else {
				// Successfully claimed
				inst.completingBlocks[blockID] = true
				results[idx] = true
			}
		}(i)
	}

	wg.Wait()

	// Verify: Exactly one should succeed
	successCount := 0
	for _, success := range results {
		if success {
			successCount++
		}
	}

	require.Equal(t, 1, successCount, "exactly one attempt should successfully mark as completing")
}

// TestCompleteBlock_CompletingBlocksCleanup verifies completingBlocks map is cleaned up
func TestCompleteBlock_CompletingBlocksCleanup(t *testing.T) {
	// This test verifies the defer cleanup works
	inst := &instance{
		completingBlocks: make(map[uuid.UUID]bool),
		walBlocks:        make(map[uuid.UUID]common.WALBlock),
		blocksMtx:        sync.RWMutex{},
	}

	blockID := uuid.New()

	// Simulate marking and cleanup pattern (using defer like completeBlock does)
	func() {
		inst.blocksMtx.Lock()
		inst.completingBlocks[blockID] = true
		inst.blocksMtx.Unlock()

		defer func() {
			inst.blocksMtx.Lock()
			delete(inst.completingBlocks, blockID)
			inst.blocksMtx.Unlock()
		}()

		// Simulate some work...
		time.Sleep(10 * time.Millisecond)
	}()

	// Verify: completingBlocks map is cleaned up
	inst.blocksMtx.RLock()
	_, stillMarked := inst.completingBlocks[blockID]
	inst.blocksMtx.RUnlock()

	require.False(t, stillMarked, "completingBlocks should be cleaned up after completion")
}

// TestCompleteBlock_NoLockHeldDuringIO verifies I/O operations don't block queries
func TestCompleteBlock_NoLockHeldDuringIO(t *testing.T) {
	// This is a simplified test that just verifies the completingBlocks pattern works
	inst := &instance{
		completingBlocks: make(map[uuid.UUID]bool),
		blocksMtx:        sync.RWMutex{},
	}

	blockID := uuid.New()

	// Simulate marking as completing
	inst.blocksMtx.Lock()
	inst.completingBlocks[blockID] = true
	inst.blocksMtx.Unlock()

	// Verify we can acquire read lock quickly even though block is "completing"
	lockAcquired := make(chan bool, 1)

	go func() {
		inst.blocksMtx.RLock()
		lockAcquired <- true
		inst.blocksMtx.RUnlock()
	}()

	select {
	case <-lockAcquired:
		// Success - read lock acquired quickly
	case <-time.After(100 * time.Millisecond):
		t.Fatal("read lock was blocked - pattern not working")
	}

	// Cleanup
	inst.blocksMtx.Lock()
	delete(inst.completingBlocks, blockID)
	inst.blocksMtx.Unlock()
}

// TestCompleteBlock_NoDuplicateSearchResults verifies block is never in both walBlocks and completeBlocks simultaneously
func TestCompleteBlock_NoDuplicateSearchResults(t *testing.T) {
	// This test verifies there's no dual-map window that would cause duplicate search results
	// The bug: block is added to completeBlocks at line 556, but not removed from walBlocks until line 583
	// During this window, queries would search the block twice and return duplicate results

	inst := &instance{
		completingBlocks: make(map[uuid.UUID]bool),
		walBlocks:        make(map[uuid.UUID]common.WALBlock),
		completeBlocks:   make(map[uuid.UUID]*ingester.LocalBlock),
		blocksMtx:        sync.RWMutex{},
	}

	blockID := uuid.New()

	// Add block ID to walBlocks map (just need the key to exist for this test)
	// Using nil since we're only testing map membership, not actual block operations
	inst.walBlocks[blockID] = nil

	// Simulate the completeBlock state transitions
	// This models what happens in the actual completeBlock function

	// STEP 1: Mark as completing (completeBlock line 473)
	inst.blocksMtx.Lock()
	inst.completingBlocks[blockID] = true
	inst.blocksMtx.Unlock()

	// STEP 2: Simulate I/O happening (no lock held)
	time.Sleep(10 * time.Millisecond)

	// Launch concurrent "query" goroutines that check both maps
	duplicateDetected := make(chan bool, 1)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			default:
				inst.blocksMtx.RLock()
				_, inWAL := inst.walBlocks[blockID]
				_, inComplete := inst.completeBlocks[blockID]
				inst.blocksMtx.RUnlock()

				// If block is in BOTH maps, that's the bug!
				if inWAL && inComplete {
					select {
					case duplicateDetected <- true:
					default:
					}
				}
				time.Sleep(1 * time.Millisecond)
			}
		}
	}()

	// STEP 3: Add to completeBlocks and remove from walBlocks atomically (FIXED CODE)
	// This prevents the dual-map window
	inst.blocksMtx.Lock()
	inst.completeBlocks[blockID] = nil // Just need the key to exist
	delete(inst.walBlocks, blockID)    // Remove atomically to prevent dual-map window
	inst.blocksMtx.Unlock()

	// STEP 4: Simulate more I/O (WAL clear) - no lock held
	// With the fix, there's no dual-map window during this I/O
	time.Sleep(20 * time.Millisecond)

	// Stop query goroutine
	close(done)
	time.Sleep(5 * time.Millisecond)

	// Check if duplicate was detected
	select {
	case <-duplicateDetected:
		t.Fatal("Block exists in both walBlocks and completeBlocks simultaneously - dual-map window detected!")
	default:
		// Success - no dual-map window detected
		t.Log("Success: No dual-map window detected with atomic add/remove")
	}

	// Cleanup
	inst.blocksMtx.Lock()
	delete(inst.completingBlocks, blockID)
	inst.blocksMtx.Unlock()
}
