package worker

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/grpcclient"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type mockProcessor struct {
	mu            sync.Mutex
	notifiedAddrs []string
}

func (m *mockProcessor) processQueriesOnSingleStream(ctx context.Context, _ *grpc.ClientConn, _ string) {
	<-ctx.Done()
}

func (m *mockProcessor) notifyShutdown(_ context.Context, _ *grpc.ClientConn, address string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifiedAddrs = append(m.notifiedAddrs, address)
}

func TestQuerierWorker_AddressLifecycle(t *testing.T) {
	cfg := Config{
		GRPCClientConfig: grpcclient.Config{},
	}
	proc := &mockProcessor{}
	w, err := newQuerierWorkerWithProcessor(cfg, log.NewNopLogger(), proc, "", nil)
	require.NoError(t, err)

	err = w.StartAsync(context.Background())
	require.NoError(t, err)
	err = w.AwaitRunning(context.Background())
	require.NoError(t, err)

	address := "127.0.0.1:9999"
	w.AddressAdded(address)

	w.mu.Lock()
	manager := w.managers[address]
	w.mu.Unlock()
	require.NotNil(t, manager)

	// Verify duplicate add doesn't cause issues
	w.AddressAdded(address)

	// Remove address
	w.AddressRemoved(address)

	w.mu.Lock()
	manager = w.managers[address]
	w.mu.Unlock()
	require.Nil(t, manager)

	// Stop service
	w.StopAsync()
	err = w.AwaitTerminated(context.Background())
	require.NoError(t, err)

	// AwaitTerminated guarantees notifyShutdown has run; lock to satisfy the race detector.
	proc.mu.Lock()
	addrs := make([]string, len(proc.notifiedAddrs))
	copy(addrs, proc.notifiedAddrs)
	proc.mu.Unlock()
	require.Contains(t, addrs, address)
}

// TestQuerierWorker_ShutdownDuringHangingDial is the regression test for #7505.
// It injects a blocking dialer so AddressAdded hangs inside connect while the
// mutex is NOT held (the fix). Shutdown must complete without deadlocking and
// no manager must be installed.
func TestQuerierWorker_ShutdownDuringHangingDial(t *testing.T) {
	proc := &mockProcessor{}
	w, err := newQuerierWorkerWithProcessor(
		Config{GRPCClientConfig: grpcclient.Config{}},
		log.NewNopLogger(), proc, "", nil,
	)
	require.NoError(t, err)

	dialStarted := make(chan struct{})
	w.connectFunc = func(ctx context.Context, _ string) (*grpc.ClientConn, error) {
		close(dialStarted) // signal that we are inside the blocking dial
		<-ctx.Done()       // block until the service context is cancelled
		return nil, ctx.Err()
	}

	require.NoError(t, w.StartAsync(context.Background()))
	require.NoError(t, w.AwaitRunning(context.Background()))

	go w.AddressAdded("127.0.0.1:9999")

	select {
	case <-dialStarted:
		// OK
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for dial to start")
	}

	// Without the fix, StopAsync would block forever because the mutex was held
	// during the dial and stopping() could never acquire it.
	w.StopAsync()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, w.AwaitTerminated(shutdownCtx))

	// The dial was cancelled; no manager should have been installed.
	w.mu.Lock()
	mgr := w.managers["127.0.0.1:9999"]
	w.mu.Unlock()
	require.Nil(t, mgr)
}
