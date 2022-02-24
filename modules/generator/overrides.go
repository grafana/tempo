package generator

import (
	"github.com/grafana/tempo/modules/overrides"
)

type metricsGeneratorOverrides interface {
	MetricsGeneratorProcessors(userID string) map[string]struct{}
}

var _ metricsGeneratorOverrides = (*overrides.Overrides)(nil)
