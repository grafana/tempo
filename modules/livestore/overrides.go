package livestore

import (
	"time"

	"github.com/grafana/tempo/modules/generator/processor/localblocks"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/tempodb/backend"
)

var (
	_ localblocks.ProcessorOverrides = (*Overrides)(nil)
)

type Overrides struct {
	cfg     Config
	service overrides.Interface
}

// NewMergedOverrides creates a new MergedOverrides that wraps the main overrides service
func NewOverrides(service overrides.Interface) *Overrides {
	return &Overrides{
		service: service,
	}
}

// Livestore Overrides interface implementation
func (m *Overrides) MaxLocalTracesPerUser(userID string) int {
	return m.service.MaxLocalTracesPerUser(userID)
}

func (m *Overrides) MaxBytesPerTrace(userID string) int {
	return m.service.MaxBytesPerTrace(userID)
}

func (m *Overrides) DedicatedColumns(userID string) backend.DedicatedColumns {
	return m.service.DedicatedColumns(userID)
}

func (m *Overrides) UnsafeQueryHints(userID string) bool {
	return m.service.UnsafeQueryHints(userID)
}

func (m *Overrides) LiveStoreCompleteBlockTimeout(userID string) time.Duration {
	return m.cfg.CompleteBlockTimeout
}

func (m *Overrides) LiveStoreMetricsConcurrentBlocks(_ string) uint {
	return 10
}
