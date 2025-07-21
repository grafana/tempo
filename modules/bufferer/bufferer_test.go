package bufferer

import (
	"testing"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/stretchr/testify/assert"
)

type mockOverrides struct{}

// mockPartitionReader implements WatermarkUpdater for testing

type mockPartitionReader struct {
	updateWatermarkFunc func(offset int64)
}

func (m *mockPartitionReader) updateWatermark(offset int64) {
	if m.updateWatermarkFunc != nil {
		m.updateWatermarkFunc(offset)
	}
}

func (m *mockOverrides) MaxLocalTracesPerUser(string) int {
	return 10000
}

func (m *mockOverrides) MaxBytesPerTrace(string) int {
	return 0 // Unlimited
}

func (m *mockOverrides) DedicatedColumns(string) backend.DedicatedColumns {
	return nil
}

func TestBufferCreation(t *testing.T) {
	// This is a basic test to ensure the buffer structure is correct

	b := &Bufferer{
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

func TestBufferer(t *testing.T) {
	b := &Bufferer{
		logger:    log.NewNopLogger(),
		instances: make(map[string]*instance),
		overrides: &mockOverrides{},
	}
	
	assert.NotNil(t, b)
	assert.NotNil(t, b.logger)
	assert.Equal(t, 0, len(b.instances))
}
