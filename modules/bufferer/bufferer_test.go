package bufferer

import (
	"testing"
	"time"

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
		logger:           log.NewNopLogger(),
		instances:        make(map[string]*instance),
		overrides:        &mockOverrides{},
		tenantWatermarks: make(map[string]int64),
	}

	// Basic validation that the buffer is set up correctly
	assert.NotNil(t, b.logger)
	assert.NotNil(t, b.instances)
	assert.NotNil(t, b.overrides)
	assert.NotNil(t, b.tenantWatermarks)
	assert.Equal(t, 0, len(b.instances))
	assert.Equal(t, 0, len(b.tenantWatermarks))
}

func TestMaxWatermarkCalculation(t *testing.T) {
	b := &Bufferer{
		logger:           log.NewNopLogger(),
		instances:        make(map[string]*instance),
		overrides:        &mockOverrides{},
		tenantWatermarks: make(map[string]int64),
	}

	// Start with empty watermarks
	b.updateTenantWatermark("tenant1", 100)
	assert.Equal(t, int64(100), b.lastCommittedOffset)

	// Add more tenants with different watermarks
	b.updateTenantWatermark("tenant2", 200) // Higher
	b.updateTenantWatermark("tenant3", 150) // Middle
	b.updateTenantWatermark("tenant4", 50)  // Lower

	// Should use maximum watermark (200)
	assert.Equal(t, int64(200), b.lastCommittedOffset)
	assert.Equal(t, int64(100), b.tenantWatermarks["tenant1"])
	assert.Equal(t, int64(200), b.tenantWatermarks["tenant2"])
	assert.Equal(t, int64(150), b.tenantWatermarks["tenant3"])
	assert.Equal(t, int64(50), b.tenantWatermarks["tenant4"])

	// Update one tenant to an even higher value
	b.updateTenantWatermark("tenant1", 300)
	assert.Equal(t, int64(300), b.lastCommittedOffset)
}

func TestPartitionReaderWatermarkManagement(t *testing.T) {
	// Test the PartitionReader's watermark update logic directly

	pr := &PartitionReader{
		logger:         log.NewNopLogger(),
		watermarkChan:  make(chan int64, 10),
		commitInterval: 100 * time.Millisecond, // Fast for testing
	}

	// Test watermark updates
	pr.updateWatermark(100)
	assert.Equal(t, int64(100), pr.highWatermark)

	// Test watermark doesn't go backwards
	pr.updateWatermark(50)
	assert.Equal(t, int64(100), pr.highWatermark) // Should not change

	// Test watermark can advance
	pr.updateWatermark(200)
	assert.Equal(t, int64(200), pr.highWatermark)

	// Test channel receives updates
	select {
	case offset := <-pr.watermarkChan:
		assert.Equal(t, int64(100), offset)
	case <-time.After(time.Second):
		t.Fatal("Should have received watermark update")
	}

	select {
	case offset := <-pr.watermarkChan:
		assert.Equal(t, int64(200), offset)
	case <-time.After(time.Second):
		t.Fatal("Should have received second watermark update")
	}
}
