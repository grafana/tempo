package livestore

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/dskit/services"
	"github.com/stretchr/testify/require"
)

func TestCompleteOpBackoffProgression(t *testing.T) {
	const (
		initialBackoff = 30 * time.Second
		maxBackoff     = 100 * time.Second
	)

	op := &completeOp{
		bo:         initialBackoff,
		maxBackoff: maxBackoff,
	}

	require.Equal(t, initialBackoff, op.backoff(), "first retry should wait initialBackoff")
	require.Equal(t, 2*initialBackoff, op.backoff(), "second retry should be 2*initialBackoff")
	require.Equal(t, maxBackoff, op.backoff(), "third retry should cap at maxBackoff")
	require.Equal(t, maxBackoff, op.backoff(), "subsequent retries remain at maxBackoff")
}

// TestProcessCompleteOpAbandonOnCancelledContext verifies that processCompleteOp
// skips WAL block completion when the service context is already cancelled
// (i.e. during shutdown), rather than attempting the work and scheduling a retry.
// The WAL block must remain on disk so that reloadBlocks() can re-enqueue it on
// the next startup.
func TestProcessCompleteOpAbandonOnCancelledContext(t *testing.T) {
	tmpDir := t.TempDir()

	liveStore, err := defaultLiveStore(t, tmpDir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = services.StopAndAwaitTerminated(context.Background(), liveStore) })

	inst, err := liveStore.getOrCreateInstance(testTenantID)
	require.NoError(t, err)

	// Push a trace, flush live traces to head block, then cut to WAL.
	pushTracesToInstance(t, inst, 1)
	drained, err := inst.cutIdleTraces(t.Context(), true)
	require.NoError(t, err)
	require.True(t, drained, "should drain live traces in one iteration")
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
