package ingester

import (
	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/tempodb/backend"
)

type ingesterOverrides interface {
	registry.Overrides

	DedicatedColumns(userID string) []backend.DedicatedColumn
}

var _ ingesterOverrides = (overrides.Interface)(nil)
