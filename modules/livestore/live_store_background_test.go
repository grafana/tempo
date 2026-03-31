package livestore

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestProcessCompleteOpAbandonOnCancelledContext verifies that processCompleteOp
// abandons an in-flight WAL block completion when the service context is cancelled
// (i.e. during shutdown), rather than scheduling a retry. The WAL block must remain
// on disk so that reloadBlocks() can re-enqueue it on the next startup.
func TestProcessCompleteOpAbandonOnCancelledContext(t *testing.T) {
	tmpDir := t.TempDir()

	liveStore, err := defaultLiveStore(t, tmpDir)
	require.NoError(t, err)

	inst, err := liveStore.getOrCreateInstance(testTenantID)
	require.NoError(t, err)

	// Push a trace, flush live traces to head block, then cut to WAL.
	pushTracesToInstance(t, inst, 1)
	err = inst.cutIdleTraces(t.Context(), true)
	require.NoError(t, err)
	walID, err := inst.cutBlocks(t.Context(), true)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, walID)
	requireInstanceState(t, inst, instanceState{liveTraces: 0, walBlocks: 1, completeBlocks: 0})

	// Simulate shutdown by cancelling the service context.
	liveStore.cancel()

	op := &completeOp{
		tenantID:   testTenantID,
		blockID:    walID,
		at:         time.Now(),
		bo:         liveStore.cfg.initialBackoff,
		maxBackoff: liveStore.cfg.maxBackoff,
		attempts:   1,
	}

	// processCompleteOp must return nil (not exit the worker loop) and must NOT
	// schedule a retry — the WAL block stays on disk for reloadBlocks() on restart.
	err = liveStore.processCompleteOp(op)
	require.NoError(t, err)

	// WAL block is still present: abandoned, not completed or removed.
	requireInstanceState(t, inst, instanceState{liveTraces: 0, walBlocks: 1, completeBlocks: 0})

	// No retry was enqueued.
	require.True(t, liveStore.completeQueues.IsEmpty())
}

