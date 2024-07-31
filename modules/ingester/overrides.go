package ingester

import (
	"github.com/grafana/tempo/v2/modules/generator/registry"
	"github.com/grafana/tempo/v2/modules/overrides"
	"github.com/grafana/tempo/v2/tempodb/backend"
)

type ingesterOverrides interface {
	registry.Overrides

	DedicatedColumns(userID string) backend.DedicatedColumns
}

var _ ingesterOverrides = (overrides.Interface)(nil)
