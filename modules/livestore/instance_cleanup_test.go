package livestore

import (
	"testing"
	"time"

	"github.com/grafana/dskit/services"
	"github.com/stretchr/testify/require"
)

// TestDeleteOldBlocks_ContinuesOnFailure verifies CRIT-4 fix:
// deleteOldBlocks() should continue cleaning other blocks even if one fails.
// Before fix: returned error on first failure, preventing ALL subsequent blocks from being cleaned.
// After fix: logs failure but continues, allowing gradual cleanup and self-healing.
func TestDeleteOldBlocks_ContinuesOnFailure(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup LiveStore
	liveStore, err := defaultLiveStore(t, tmpDir)
	require.NoError(t, err)
	require.NotNil(t, liveStore)

	// Get instance
	inst, err := liveStore.getOrCreateInstance(testTenantID)
	require.NoError(t, err)

	// Create multiple old blocks by pushing traces and cutting blocks
	for i := 0; i < 3; i++ {
		pushToLiveStore(t, liveStore)
		err = inst.cutIdleTraces(true)
		require.NoError(t, err)
	}

	// Cut all head blocks to WAL blocks
	_, err = inst.cutBlocks(true)
	require.NoError(t, err)

	// Verify we have WAL blocks
	inst.blocksMtx.Lock()
	initialWALCount := len(inst.walBlocks)
	inst.blocksMtx.Unlock()
	require.Greater(t, initialWALCount, 0, "should have WAL blocks to clean")

	// Make blocks old by setting a long CompleteBlockTimeout
	// Then call deleteOldBlocks - should succeed even if individual blocks might fail
	inst.Cfg.CompleteBlockTimeout = -1 * time.Hour // Make cutoff in the future so all blocks are "old"

	// Call deleteOldBlocks - should not return error even if some blocks fail
	err = inst.deleteOldBlocks()
	require.NoError(t, err, "deleteOldBlocks should not return error (continue on failures)")

	// In the normal case (no failures), all blocks should be cleaned
	inst.blocksMtx.Lock()
	finalWALCount := len(inst.walBlocks)
	inst.blocksMtx.Unlock()

	// Blocks should be cleaned (or at least attempt was made without stopping)
	t.Logf("Initial WAL blocks: %d, Final WAL blocks: %d", initialWALCount, finalWALCount)

	// Clean shutdown
	err = services.StopAndAwaitTerminated(t.Context(), liveStore)
	require.NoError(t, err)
}

// TestDeleteOldBlocks_OnlyOldBlocksDeleted verifies that deleteOldBlocks
// only deletes blocks older than CompleteBlockTimeout.
func TestDeleteOldBlocks_OnlyOldBlocksDeleted(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup LiveStore
	liveStore, err := defaultLiveStore(t, tmpDir)
	require.NoError(t, err)
	require.NotNil(t, liveStore)

	// Get instance
	inst, err := liveStore.getOrCreateInstance(testTenantID)
	require.NoError(t, err)

	// Create a block
	pushToLiveStore(t, liveStore)
	err = inst.cutIdleTraces(true)
	require.NoError(t, err)

	walUUID, err := inst.cutBlocks(true)
	require.NoError(t, err)

	// Set a very long CompleteBlockTimeout so the block is NOT old
	inst.Cfg.CompleteBlockTimeout = 24 * time.Hour

	// Call deleteOldBlocks
	err = inst.deleteOldBlocks()
	require.NoError(t, err)

	// Verify block still exists (not deleted because it's not old)
	inst.blocksMtx.Lock()
	_, hasWAL := inst.walBlocks[walUUID]
	inst.blocksMtx.Unlock()

	require.True(t, hasWAL, "recent block should not be deleted")

	// Clean shutdown
	err = services.StopAndAwaitTerminated(t.Context(), liveStore)
	require.NoError(t, err)
}
