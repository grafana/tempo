package livestore

import (
	"testing"

	"github.com/grafana/dskit/services"
	"github.com/stretchr/testify/require"
)

// TestCompleteBlock_NoOrphanAccumulation verifies that the orphan check
// before CreateBlock doesn't break normal block completion flow.
// This test ensures CRIT-3 fix (orphan cleanup) doesn't introduce regressions.
func TestCompleteBlock_NoOrphanAccumulation(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup LiveStore
	liveStore, err := defaultLiveStore(t, tmpDir)
	require.NoError(t, err)
	require.NotNil(t, liveStore)

	// Get an instance
	inst, err := liveStore.getOrCreateInstance(testTenantID)
	require.NoError(t, err)

	// Push trace to create a live trace
	expectedID, expectedTrace := pushToLiveStore(t, liveStore)

	// Cut trace to head block
	err = inst.cutIdleTraces(true)
	require.NoError(t, err)

	// Cut head block to create a WAL block
	walUUID, err := inst.cutBlocks(true)
	require.NoError(t, err)
	require.NotNil(t, walUUID)

	// Verify WAL block exists
	inst.blocksMtx.Lock()
	_, hasWAL := inst.walBlocks[walUUID]
	inst.blocksMtx.Unlock()
	require.True(t, hasWAL, "WAL block should exist")

	// Complete the block (this should trigger orphan check but find nothing)
	err = inst.completeBlock(t.Context(), walUUID)
	require.NoError(t, err, "completeBlock should succeed with orphan check")

	// Verify complete block exists
	inst.blocksMtx.Lock()
	completeBlock, hasComplete := inst.completeBlocks[walUUID]
	_, stillHasWAL := inst.walBlocks[walUUID]
	inst.blocksMtx.Unlock()

	require.True(t, hasComplete, "complete block should exist")
	require.False(t, stillHasWAL, "WAL block should be removed after completion")
	require.NotNil(t, completeBlock)

	// Verify trace is still accessible in complete block
	requireTraceInBlock(t, completeBlock, expectedID, expectedTrace)

	// Clean shutdown
	err = services.StopAndAwaitTerminated(t.Context(), liveStore)
	require.NoError(t, err)
}

// TestReloadBlocks_OrphanCleanup verifies that orphan cleanup on startup
// doesn't break normal block reloading.
func TestReloadBlocks_OrphanCleanup(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup first LiveStore instance
	liveStore1, err := defaultLiveStore(t, tmpDir)
	require.NoError(t, err)

	// Push trace and create complete block
	inst1, err := liveStore1.getOrCreateInstance(testTenantID)
	require.NoError(t, err)

	expectedID, expectedTrace := pushToLiveStore(t, liveStore1)
	err = inst1.cutIdleTraces(true)
	require.NoError(t, err)

	walUUID, err := inst1.cutBlocks(true)
	require.NoError(t, err)

	err = inst1.completeBlock(t.Context(), walUUID)
	require.NoError(t, err)

	// Stop first instance
	err = services.StopAndAwaitTerminated(t.Context(), liveStore1)
	require.NoError(t, err)

	// Restart with second instance (should run orphan cleanup in reloadBlocks)
	liveStore2, err := defaultLiveStore(t, tmpDir)
	require.NoError(t, err)

	inst2, err := liveStore2.getOrCreateInstance(testTenantID)
	require.NoError(t, err)

	// Verify complete block was reloaded correctly
	inst2.blocksMtx.Lock()
	completeBlock, hasComplete := inst2.completeBlocks[walUUID]
	inst2.blocksMtx.Unlock()

	require.True(t, hasComplete, "complete block should be reloaded after restart")
	require.NotNil(t, completeBlock)

	// Verify trace is still accessible
	requireTraceInBlock(t, completeBlock, expectedID, expectedTrace)

	// Clean shutdown
	err = services.StopAndAwaitTerminated(t.Context(), liveStore2)
	require.NoError(t, err)
}
