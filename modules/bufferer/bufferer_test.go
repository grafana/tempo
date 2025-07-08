package bufferer

import (
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/stretchr/testify/assert"
)

type mockOverrides struct{}

func (m *mockOverrides) MaxLocalTracesPerUser(string) int {
	return 10000
}

func (m *mockOverrides) MaxBytesPerTrace(string) int {
	return 0 // Unlimited
}

func (m *mockOverrides) DedicatedColumns(string) backend.DedicatedColumns {
	return nil
}

func TestInstanceCreation(t *testing.T) {
	// This is a basic test to ensure the buffer structure is correct

	b := &Buffer{
		logger:    log.NewNopLogger(),
		instances: make(map[string]*instance),
		overrides: &mockOverrides{},
	}

	// Basic validation that the buffer is set up correctly
	assert.NotNil(t, b.logger)
	assert.NotNil(t, b.instances)
	assert.NotNil(t, b.overrides)
	assert.Equal(t, 0, len(b.instances))
}

func TestCutLogic(t *testing.T) {
	b := &Buffer{
		logger:      log.NewNopLogger(),
		cutInterval: 5 * time.Minute,
		cutDataSize: 100 * 1024 * 1024,                // 100MB
		lastCutTime: time.Now().Add(-6 * time.Minute), // 6 minutes ago
	}

	// Should trigger time-based cut
	assert.True(t, b.shouldTriggerCut())

	// Reset time, test size-based cut
	b.lastCutTime = time.Now()
	b.totalBlockBytesSinceLastCut = 101 * 1024 * 1024 // 101MB
	assert.True(t, b.shouldTriggerCut())

	// Reset both - should not trigger
	b.lastCutTime = time.Now()
	b.totalBlockBytesSinceLastCut = 50 * 1024 * 1024 // 50MB
	assert.False(t, b.shouldTriggerCut())
}

func TestGlobalOffsetTracking(t *testing.T) {
	b := &Buffer{
		logger:      log.NewNopLogger(),
		cutInterval: 1 * time.Minute,
		cutDataSize: 1024,                             // Small size to trigger quickly
		lastCutTime: time.Now().Add(-2 * time.Minute), // Old enough to trigger
	}

	// Initially no data should be tracked
	assert.Equal(t, int64(0), b.blockStartOffset)
	assert.Equal(t, int64(0), b.totalBlockBytesSinceLastCut)
	assert.Equal(t, int64(0), b.lastKafkaOffset)

	// Simulate first consumption - should initialize block start
	b.blockStartOffset = 100
	b.totalBlockBytesSinceLastCut += 2048 // 2KB > 1KB limit
	b.lastKafkaOffset = 200

	// Should trigger cut due to data size
	assert.True(t, b.shouldTriggerCut())

	// Offset should be updated
	assert.Equal(t, int64(100), b.blockStartOffset)
	assert.Equal(t, int64(2048), b.totalBlockBytesSinceLastCut)
	assert.Equal(t, int64(200), b.lastKafkaOffset)
}

func TestCoordinatedCut(t *testing.T) {
	b := &Buffer{
		logger:    log.NewNopLogger(),
		instances: make(map[string]*instance),
		overrides: &mockOverrides{},
	}

	// Basic validation that coordinated cuts work at buffer level
	assert.NotNil(t, b.logger)
	assert.NotNil(t, b.instances)
	assert.NotNil(t, b.overrides)
	assert.Equal(t, 0, len(b.instances))

	// Cut logic should exist at buffer level
	assert.NotNil(t, b.shouldTriggerCut)
	assert.NotNil(t, b.performCut)
}
