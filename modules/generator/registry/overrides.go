package registry

import (
	"time"

	"github.com/grafana/tempo/modules/overrides"
)

type Overrides interface {
	MetricsGeneratorMaxActiveSeries(userID string) uint32
	MetricsGeneratorCollectionInterval(userID string) time.Duration
}

var _ Overrides = (*overrides.Overrides)(nil)
