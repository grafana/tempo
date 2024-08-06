package registry

import (
	"time"

	"github.com/grafana/tempo/v2/modules/overrides"
)

type Overrides interface {
	MetricsGeneratorMaxActiveSeries(userID string) uint32
	MetricsGeneratorCollectionInterval(userID string) time.Duration
	MetricsGeneratorDisableCollection(userID string) bool
	MetricsGeneratorGenerateNativeHistograms(userID string) string
	MetricsGenerationTraceIDLabelName(userID string) string
}

var _ Overrides = (overrides.Interface)(nil)
