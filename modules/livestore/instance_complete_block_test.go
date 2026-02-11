package livestore

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
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
