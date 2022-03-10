package generator

import (
	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/modules/overrides"
)

type metricsGeneratorOverrides interface {
	registry.Overrides

	MetricsGeneratorProcessors(userID string) map[string]struct{}
}

var _ metricsGeneratorOverrides = (*overrides.Overrides)(nil)
