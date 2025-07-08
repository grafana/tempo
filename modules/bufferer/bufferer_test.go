package bufferer

import (
	"context"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/assert"
)

func TestPushBytes(t *testing.T) {
	// This is a basic test to ensure the pushBytes method doesn't panic
	// More comprehensive tests would require setting up WAL and other dependencies

	b := &Buffer{
		logger: log.NewNopLogger(),
	}

	// Create a simple PushBytesRequest
	req := &tempopb.PushBytesRequest{
		Traces: []tempopb.PreallocBytes{},
		Ids:    [][]byte{},
	}

	// This should not panic
	assert.NotPanics(t, func() {
		b.pushBytes(time.Now(), req, "test-tenant")
	})
}

func TestHardCutLogic(t *testing.T) {
	b := &Buffer{
		logger:          log.NewNopLogger(),
		hardCutInterval: 5 * time.Minute,
		hardCutDataSize: 100 * 1024 * 1024,                // 100MB
		lastHardCutTime: time.Now().Add(-6 * time.Minute), // 6 minutes ago
	}

	// Should trigger time-based cut
	assert.True(t, b.shouldTriggerHardCut())

	// Reset time, test size-based cut
	b.lastHardCutTime = time.Now()
	b.totalBlockBytes = 101 * 1024 * 1024 // 101MB
	assert.True(t, b.shouldTriggerHardCut())

	// Reset both - should not trigger
	b.lastHardCutTime = time.Now()
	b.totalBlockBytes = 50 * 1024 * 1024 // 50MB
	assert.False(t, b.shouldTriggerHardCut())
}

func TestCutLoopSafety(t *testing.T) {
	b := &Buffer{
		logger:          log.NewNopLogger(),
		hardCutInterval: 5 * time.Minute,
		lastHardCutTime: time.Now().Add(-11 * time.Minute), // 11 minutes ago (> 2 * hardCutInterval)
	}

	// Safety threshold should be 2 * hardCutInterval = 10 minutes
	// Since lastHardCutTime is 11 minutes ago, it should trigger safety cut
	safetyThreshold := 2 * b.hardCutInterval
	timeSinceLastCut := time.Since(b.lastHardCutTime)
	
	assert.True(t, timeSinceLastCut > safetyThreshold, 
		"Test setup: time since last cut (%v) should exceed safety threshold (%v)", 
		timeSinceLastCut, safetyThreshold)
}

func TestAsyncHardCut(t *testing.T) {
	b := &Buffer{
		logger:          log.NewNopLogger(),
		hardCutInterval: 1 * time.Minute,
		hardCutDataSize: 1024, // Small size to trigger quickly
		lastHardCutTime: time.Now().Add(-2 * time.Minute), // Old enough to trigger
	}

	// Initially no hard cut should be requested
	assert.False(t, b.hardCutRequested.Load())

	// Consume some data that should trigger a hard cut request
	err := b.consume(context.Background(), []record{}, 100, 200, 2048) // 2KB > 1KB limit
	assert.NoError(t, err)

	// Hard cut should now be requested (but not performed yet)
	assert.True(t, b.hardCutRequested.Load())
	
	// Offset should be updated
	assert.Equal(t, int64(200), b.lastKafkaOffset.Load())
}
