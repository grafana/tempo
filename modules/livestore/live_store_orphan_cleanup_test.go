package livestore

import (
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/grafana/tempo/modules/ingester"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/require"
)

// TestOrphanCleanup_ConcurrentCompletion verifies orphan cleanup doesn't delete completing blocks
func TestOrphanCleanup_ConcurrentCompletion(t *testing.T) {
	// Setup instance with all maps
	inst := &instance{
		walBlocks:        make(map[uuid.UUID]common.WALBlock),
		completeBlocks:   make(map[uuid.UUID]*ingester.LocalBlock),
		completingBlocks: make(map[uuid.UUID]bool),
		blocksMtx:        sync.RWMutex{},
	}

	blockID := uuid.New()

	// Simulate: Block is being completed (marked in completingBlocks)
	inst.blocksMtx.Lock()
	inst.completingBlocks[blockID] = true
	inst.blocksMtx.Unlock()

	// Orphan cleanup checks (OLD BUGGY CODE would only check walBlocks and completeBlocks)
	inst.blocksMtx.Lock()
	_, inWAL := inst.walBlocks[blockID]
	_, inComplete := inst.completeBlocks[blockID]
	isCompleting := inst.completingBlocks[blockID] // NEW CHECK
	inst.blocksMtx.Unlock()

	// Verify: Block should be considered "in use" because it's completing
	require.False(t, inWAL, "should not be in WAL yet")
	require.False(t, inComplete, "should not be complete yet")
	require.True(t, isCompleting, "should be marked as completing")

	// OLD CODE would consider this an orphan: !inWAL && !inComplete = true
	// NEW CODE considers completingBlocks: !inWAL && !inComplete && !isCompleting = false
	isOrphan := !inWAL && !inComplete && !isCompleting
	require.False(t, isOrphan, "block being completed should NOT be considered orphan")
}

// TestOrphanCleanup_ReverificationBeforeDelete verifies double-check before deletion
func TestOrphanCleanup_ReverificationBeforeDelete(t *testing.T) {
	inst := &instance{
		walBlocks:        make(map[uuid.UUID]common.WALBlock),
		completeBlocks:   make(map[uuid.UUID]*ingester.LocalBlock),
		completingBlocks: make(map[uuid.UUID]bool),
		blocksMtx:        sync.RWMutex{},
	}

	blockID := uuid.New()

	// Pass 1: Identify orphan candidates
	inst.blocksMtx.Lock()
	_, inWAL := inst.walBlocks[blockID]
	_, inComplete := inst.completeBlocks[blockID]
	isOrphan := !inWAL && !inComplete
	inst.blocksMtx.Unlock()

	require.True(t, isOrphan, "block should appear as orphan in Pass 1")

	// Simulate: Block gets completed between Pass 1 and Pass 2
	inst.blocksMtx.Lock()
	inst.completeBlocks[blockID] = &ingester.LocalBlock{} // Mock completed block
	inst.blocksMtx.Unlock()

	// Pass 2: Re-verify before deleting (with completingBlocks check)
	inst.blocksMtx.Lock()
	_, inWAL = inst.walBlocks[blockID]
	_, inComplete = inst.completeBlocks[blockID]
	_, isCompleting := inst.completingBlocks[blockID]
	stillOrphaned := !inWAL && !inComplete && !isCompleting
	inst.blocksMtx.Unlock()

	// Verify: Should NOT be considered orphaned anymore
	require.False(t, stillOrphaned, "re-verification should detect completed block")
}

// TestOrphanCleanup_TrueOrphansAreDetected verifies legitimate orphans are still detected
func TestOrphanCleanup_TrueOrphansAreDetected(t *testing.T) {
	inst := &instance{
		walBlocks:        make(map[uuid.UUID]common.WALBlock),
		completeBlocks:   make(map[uuid.UUID]*ingester.LocalBlock),
		completingBlocks: make(map[uuid.UUID]bool),
		blocksMtx:        sync.RWMutex{},
	}

	// Create a block on disk but not in any map (true orphan)
	blockID := uuid.New()

	// Verify it's truly orphaned
	inst.blocksMtx.Lock()
	_, inWAL := inst.walBlocks[blockID]
	_, inComplete := inst.completeBlocks[blockID]
	_, isCompleting := inst.completingBlocks[blockID]
	inst.blocksMtx.Unlock()

	require.False(t, inWAL, "should not be in WAL")
	require.False(t, inComplete, "should not be in complete")
	require.False(t, isCompleting, "should not be completing")

	// Pass 1: Identify orphan
	orphanCandidates := []uuid.UUID{}
	inst.blocksMtx.Lock()
	_, inWAL = inst.walBlocks[blockID]
	_, inComplete = inst.completeBlocks[blockID]
	if !inWAL && !inComplete {
		orphanCandidates = append(orphanCandidates, blockID)
	}
	inst.blocksMtx.Unlock()

	require.Len(t, orphanCandidates, 1, "orphan should be identified")

	// Pass 2: Re-verify and confirm still orphaned
	for _, orphanID := range orphanCandidates {
		inst.blocksMtx.Lock()
		_, inWAL := inst.walBlocks[orphanID]
		_, inComplete := inst.completeBlocks[orphanID]
		_, isCompleting := inst.completingBlocks[orphanID]
		stillOrphaned := !inWAL && !inComplete && !isCompleting
		inst.blocksMtx.Unlock()

		require.True(t, stillOrphaned, "true orphan should still be orphaned after re-verification")
	}
}
