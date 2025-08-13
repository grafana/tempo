package livestore

import (
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/modules/generator/processor/localblocks"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/prometheus/client_golang/prometheus"
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

func (m *mockOverrides) UnsafeQueryHints(string) bool {
	return false
}

func (m *mockOverrides) LiveStoreCompleteBlockTimeout(string) time.Duration {
	return 5 * time.Minute
}

func (m *mockOverrides) LiveStoreMetricsTimeOverlapCutoff(string) float64 {
	return 0.5
}

func (m *mockOverrides) LiveStoreMetricsConcurrentBlocks(string) uint {
	return 10
}

func TestBufferCreation(t *testing.T) {
	// This is a basic test to ensure the buffer structure is correct

	b := &LiveStore{
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

func TestLiveStore(t *testing.T) {
	b := &LiveStore{
		logger:    log.NewNopLogger(),
		instances: make(map[string]*instance),
		overrides: &mockOverrides{},
	}

	assert.NotNil(t, b)
	assert.NotNil(t, b.logger)
	assert.Equal(t, 0, len(b.instances))
}

func TestMergedOverrides(t *testing.T) {
	// Create a mock overrides service
	limits, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.DefaultRegisterer)
	assert.NoError(t, err)

	// Create merged overrides
	merged := NewMergedOverrides(limits)

	// Test that it implements both interfaces
	var _ Overrides = merged
	var _ localblocks.ProcessorOverrides = merged

	// Test that all interface methods work without panicking
	// and return reasonable values
	maxLocalTraces := merged.MaxLocalTracesPerUser("test-tenant")
	assert.True(t, maxLocalTraces >= 0)

	unsafeHints := merged.UnsafeQueryHints("test-tenant")
	_ = unsafeHints // Just verify it doesn't panic

	timeout := merged.LiveStoreCompleteBlockTimeout("test-tenant")
	assert.True(t, timeout >= 0)

	// Test livestore-specific overrides that have hardcoded defaults
	assert.Equal(t, 0.5, merged.LiveStoreMetricsTimeOverlapCutoff("test-tenant"))
	assert.Equal(t, uint(10), merged.LiveStoreMetricsConcurrentBlocks("test-tenant"))
}
