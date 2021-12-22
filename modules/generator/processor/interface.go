package processor

import (
	"context"

	"go.opentelemetry.io/collector/model/pdata"
)

// TODO review this interface
//  we probably need something to update the configuration as well
//  should we use pdata.Traces?

type Processor interface {
	Name() string
	ConsumeTraces(ctx context.Context, td pdata.Traces) error
	Shutdown(ctx context.Context) error
}
