package livestore

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	uberatomic "go.uber.org/atomic"
)

// TestCheckReady_NilPointerPanic tests the nil pointer dereference bug
func TestCheckReady_NilPointerPanic(t *testing.T) {
	s := &LiveStore{}

	// Simulate the bug: Store nil pointer
	s.readyErr.Store(nil)

	// This should NOT panic
	err := s.CheckReady(context.Background())
	require.NoError(t, err, "CheckReady should return nil when ready")
}

// TestCheckReady_StateTransitions tests all state transitions
func TestCheckReady_StateTransitions(t *testing.T) {
	s := &LiveStore{}

	// State 1: Starting (error stored)
	startErr := errors.New("live-store is starting")
	s.readyErr.Store(startErr) // With atomic.Error, no pointer needed

	err := s.CheckReady(context.Background())
	require.Error(t, err)
	require.Equal(t, "live-store is starting", err.Error())

	// State 2: Ready (nil stored) - WITH FIX, this is safe
	s.readyErr.Store(nil)

	// Should NOT panic here
	err = s.CheckReady(context.Background())
	require.NoError(t, err, "CheckReady should return nil when ready")

	// State 3: Error again (stopping)
	stopErr := errors.New("live-store is stopping")
	s.readyErr.Store(stopErr) // With atomic.Error, no pointer needed

	err = s.CheckReady(context.Background())
	require.Error(t, err)
	require.Equal(t, "live-store is stopping", err.Error())

	// State 4: Ready again
	s.readyErr.Store(nil)

	err = s.CheckReady(context.Background())
	require.NoError(t, err)
}

// TestCheckReady_ConcurrentAccess tests concurrent CheckReady calls during state changes
func TestCheckReady_ConcurrentAccess(t *testing.T) {
	s := &LiveStore{}

	// Start in error state
	startErr := errors.New("starting")
	s.readyErr.Store(startErr) // With atomic.Error, no pointer needed

	var wg sync.WaitGroup
	errCount := atomic.Int32{}
	okCount := atomic.Int32{}
	panicCount := atomic.Int32{}

	// Spawn 100 goroutines calling CheckReady concurrently
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panicCount.Add(1)
				}
			}()

			err := s.CheckReady(context.Background())
			if err != nil {
				errCount.Add(1)
			} else {
				okCount.Add(1)
			}
		}()
	}

	// Transition to ready state during concurrent checks
	s.readyErr.Store(nil)

	// Wait for all goroutines
	wg.Wait()

	// Verify no panics occurred
	require.Equal(t, int32(0), panicCount.Load(), "CheckReady should never panic")
	require.Equal(t, int32(100), errCount.Load()+okCount.Load(), "all calls should complete")
}

// TestCheckReady_WithAtomicError verifies atomic.Error type works correctly
func TestCheckReady_WithAtomicError(t *testing.T) {
	// This test demonstrates the FIXED behavior using uber atomic.Error
	var readyErr uberatomic.Error

	// Simulate starting state
	startErr := errors.New("starting")
	readyErr.Store(startErr)

	loaded := readyErr.Load()
	require.Equal(t, "starting", loaded.Error())

	// Simulate ready state - store nil
	readyErr.Store(nil)

	loaded = readyErr.Load()
	// With atomic.Error, Load() returns error or nil (not pointer)
	// No dereference needed
	require.Nil(t, loaded)
}
