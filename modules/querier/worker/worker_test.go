package worker

import (
	"context"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type mockProcessor struct {
}

func (m *mockProcessor) processQueriesOnSingleStream(ctx context.Context, conn *grpc.ClientConn, address string) {
}

func (m *mockProcessor) notifyShutdown(ctx context.Context, conn *grpc.ClientConn, address string) {
}

func TestAddressAdded_NoDeadlockOnShutdown(t *testing.T) {
	// Initialize the worker with no address so it doesn't start a DNSWatcher.
	worker, err := newQuerierWorkerWithProcessor(Config{}, log.NewNopLogger(), &mockProcessor{}, "", nil)
	require.NoError(t, err)

	ctx := context.Background()
	err = worker.StartAsync(ctx)
	require.NoError(t, err)
	defer func() {
		worker.StopAsync()
	}()

	err = worker.AwaitRunning(ctx)
	require.NoError(t, err)

	// Use a non-routable IP address to force the dial to block/hang.
	// RFC 5737 specifies 192.0.2.0/24 for documentation, which is non-routable.
	const nonRoutableAddr = "192.0.2.1:1234"

	done := make(chan struct{})
	go func() {
		worker.AddressAdded(nonRoutableAddr)
		close(done)
	}()

	// Wait a moment to ensure AddressAdded has entered the w.connect dial step.
	time.Sleep(100 * time.Millisecond)

	// Verify that the mutex w.mu is not held during the dial.
	// We should be able to acquire and release the lock immediately.
	lockAcquired := make(chan struct{})
	go func() {
		worker.mu.Lock()
		_ = len(worker.managers)
		worker.mu.Unlock()
		close(lockAcquired)
	}()

	select {
	case <-lockAcquired:
		// Success: lock was acquired.
	case <-time.After(1 * time.Second):
		t.Fatal("deadlock: worker mutex is held during connection dial")
	}

	// Verify that stopping the worker succeeds and cancels the dial context,
	// allowing AddressAdded to return without hanging.
	worker.StopAsync()

	stopCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	err = worker.AwaitTerminated(stopCtx)
	require.NoError(t, err)

	select {
	case <-done:
		// Success: AddressAdded returned after shutdown context was cancelled.
	case <-time.After(1 * time.Second):
		t.Fatal("hang: AddressAdded did not return after worker shutdown")
	}
}
