package processor

import (
	"context"

	"github.com/grafana/tempo/pkg/tempopb"
)

// TODO review this interface
//  we probably need something to update the configuration as well

type Processor interface {
	Name() string
	PushSpans(ctx context.Context, req *tempopb.PushSpansRequest) error
	Shutdown(ctx context.Context) error
}
