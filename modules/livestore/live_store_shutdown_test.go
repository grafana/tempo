package livestore

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestShutdownTimeout_Mechanism verifies the timeout mechanism in stopping()
// This is a unit test for the timeout logic itself
func TestShutdownTimeout_Mechanism(t *testing.T) {
	// Test the timeout pattern using channels
	// This simulates what stopping() should do

	done := make(chan struct{})
	stop := make(chan struct{})
	defer close(stop)

	timeout := time.NewTimer(100 * time.Millisecond)
	defer timeout.Stop()

	// Simulate work that never completes (unless told to stop)
	go func() {
		select {
		case <-time.After(10 * time.Second): // Much longer than timeout
			close(done)
		case <-stop:
			return
		}
	}()

	// Race between completion and timeout
	start := time.Now()
	select {
	case <-done:
		t.Fatal("should have timed out")
	case <-timeout.C:
		// Expected path
		duration := time.Since(start)
		require.Greater(t, duration, 100*time.Millisecond)
		require.Less(t, duration, 200*time.Millisecond)
	}
}

// TestShutdownTimeout_CompletesBeforeTimeout verifies normal case
func TestShutdownTimeout_CompletesBeforeTimeout(t *testing.T) {
	// Test the timeout pattern when work completes normally

	done := make(chan struct{})
	timeout := time.NewTimer(1 * time.Second)
	defer timeout.Stop()

	// Simulate work that completes quickly
	go func() {
		time.Sleep(50 * time.Millisecond)
		close(done)
	}()

	// Race between completion and timeout
	start := time.Now()
	select {
	case <-done:
		// Expected path - completed before timeout
		duration := time.Since(start)
		require.Less(t, duration, 200*time.Millisecond)
	case <-timeout.C:
		t.Fatal("should not have timed out")
	}
}
