package livestore

import (
	"time"

	"github.com/grafana/tempo/modules/generator/processor/localblocks"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/tempodb/backend"
)

// MergedOverrides implements both the livestore Overrides interface
// and the generator ProcessorOverrides interface by delegating to
// the main overrides service.
//
// Usage example:
//
//	// Create the main overrides service
//	limits, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.DefaultRegisterer)
//	if err != nil {
//		return err
//	}
//
//	// Create merged overrides for livestore
//	merged := NewMergedOverrides(limits)
//
//	// Use with livestore
//	liveStore, err := New(cfg, merged, logger, reg, singlePartition)
//
//	// Or use the convenience constructor
//	liveStore, err := NewWithService(cfg, limits, logger, reg, singlePartition)
type MergedOverrides struct {
	service overrides.Interface
}

// NewMergedOverrides creates a new MergedOverrides that wraps the main overrides service
func NewMergedOverrides(service overrides.Interface) *MergedOverrides {
	return &MergedOverrides{
		service: service,
	}
}

// Livestore Overrides interface implementation
func (m *MergedOverrides) MaxLocalTracesPerUser(userID string) int {
	return m.service.MaxLocalTracesPerUser(userID)
}

func (m *MergedOverrides) MaxBytesPerTrace(userID string) int {
	return m.service.MaxBytesPerTrace(userID)
}

func (m *MergedOverrides) DedicatedColumns(userID string) backend.DedicatedColumns {
	return m.service.DedicatedColumns(userID)
}

func (m *MergedOverrides) UnsafeQueryHints(userID string) bool {
	return m.service.UnsafeQueryHints(userID)
}

func (m *MergedOverrides) LiveStoreCompleteBlockTimeout(userID string) time.Duration {
	// Map to the processor local blocks complete block timeout
	return m.service.MetricsGeneratorProcessorLocalBlocksCompleteBlockTimeout(userID)
}

func (m *MergedOverrides) LiveStoreMetricsTimeOverlapCutoff(userID string) float64 {
	// This is a livestore-specific setting, provide a reasonable default
	// since it's not available in the main overrides service
	return 0.5
}

func (m *MergedOverrides) LiveStoreMetricsConcurrentBlocks(userID string) uint {
	// This is a livestore-specific setting, provide a reasonable default
	// since it's not available in the main overrides service
	return 10
}

// Verify that MergedOverrides implements both interfaces
var (
	_ Overrides                      = (*MergedOverrides)(nil)
	_ localblocks.ProcessorOverrides = (*MergedOverrides)(nil)
)

// ProcessorOverrides interface from modules/generator/processor/localblocks/processor.go
// is automatically implemented by the above methods:
// - DedicatedColumns(string) backend.DedicatedColumns
// - MaxLocalTracesPerUser(userID string) int
// - MaxBytesPerTrace(string) int
// - UnsafeQueryHints(string) bool
