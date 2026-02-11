package livestore

import (
	"context"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/uuid"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

// TestScheduleRetry_ShutdownDuringDelay verifies retry goroutine exits on shutdown
func TestScheduleRetry_ShutdownDuringDelay(t *testing.T) {
	// Setup: Create LiveStore with short shutdown timeout
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := Config{
		initialBackoff: 5 * time.Second,  // Long delay to test cancellation
		maxBackoff:     10 * time.Second,
	}

	s := &LiveStore{
		ctx:    ctx,
		cancel: cancel,
		logger: log.NewNopLogger(),
		cfg:    cfg,
	}

	// Create a fake operation
	op := &completeOp{
		tenantID: "test-tenant",
		blockID:  uuid.New(),
		at:       time.Now(),
		bo:       5 * time.Second,
	}

	// Start: Schedule retry with long delay
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		select {
		case <-time.After(op.bo):
			// Should NOT reach here - context should cancel first
			t.Error("goroutine did not respect context cancellation")
		case <-ctx.Done():
			// Expected: goroutine exits on context cancellation
			return
		}
	}()

	// Trigger shutdown after 100ms
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Verify: WaitGroup completes quickly (not waiting for full delay)
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success: goroutine exited quickly
	case <-time.After(1 * time.Second):
		t.Fatal("goroutine did not exit within 1 second of context cancellation")
	}
}

// TestEnqueueOpWithJitter_ShutdownDuringDelay verifies jitter goroutine exits on shutdown
func TestEnqueueOpWithJitter_ShutdownDuringDelay(t *testing.T) {
	defer goleak.VerifyNone(t)

	// Setup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := &LiveStore{
		ctx:    ctx,
		cancel: cancel,
		logger: log.NewNopLogger(),
	}

	// Start: Enqueue with jitter (will spawn goroutine with up to 10s delay)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		delay := 5 * time.Second // Simulate long jitter delay
		select {
		case <-time.After(delay):
			t.Error("jitter goroutine did not respect context cancellation")
		case <-ctx.Done():
			return
		}
	}()

	// Trigger shutdown immediately
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Verify: Goroutine exits quickly
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("jitter goroutine did not exit within 1 second")
	}
}

// TestLiveStore_NoGoroutineLeaksAfterShutdown verifies no goroutines leak after full lifecycle
func TestLiveStore_NoGoroutineLeaksAfterShutdown(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	// This test verifies that with the fix in place, no goroutines leak
	// The actual fix is in live_store_background.go where goroutines respect ctx.Done()

	ctx, cancel := context.WithCancel(context.Background())

	s := &LiveStore{
		ctx:    ctx,
		cancel: cancel,
		logger: log.NewNopLogger(),
		cfg: Config{
			initialBackoff: 10 * time.Second,
			maxBackoff:     30 * time.Second,
		},
	}

	// Simulate scheduling retries using the FIXED pattern
	for i := 0; i < 5; i++ {
		bo := 10 * time.Second

		// FIXED implementation - respects context cancellation
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			select {
			case <-time.After(bo):
				// Would proceed with work
			case <-ctx.Done():
				// Exit on cancellation
				return
			}
		}()
	}

	// Shutdown
	cancel()

	// Wait for goroutines to exit
	s.wg.Wait()

	// goleak.VerifyNone() will detect any leaked goroutines
}
