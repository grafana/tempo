package registry

import (
	"time"

	"github.com/grafana/tempo/modules/overrides"
)

type Overrides interface {
	MetricsGeneratorMaxActiveSeries(userID string) uint32
	MetricsGeneratorCollectionInterval(userID string) time.Duration
	MetricsGeneratorDisableCollection(userID string) bool
	MetricsGenerationTraceIDLabelName(userID string) string
}

var _ Overrides = (overrides.Interface)(nil)
