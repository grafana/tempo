package livestore

import (
	"github.com/grafana/tempo/modules/generator/processor/localblocks"
	"github.com/grafana/tempo/modules/overrides"
)

var _ localblocks.ProcessorOverrides = (*Overrides)(nil)

type Overrides struct {
	cfg Config
	overrides.Interface
}

func NewOverrides(service overrides.Interface) *Overrides {
	return &Overrides{
		Interface: service,
	}
}
