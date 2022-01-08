package processor

import (
	"context"

	"github.com/prometheus/prometheus/storage"

	"github.com/grafana/tempo/pkg/tempopb"
)

// TODO review this interface
//  we probably need something to update the configuration as well

type Processor interface {
	Name() string
	PushSpans(ctx context.Context, req *tempopb.PushSpansRequest) error
	CollectMetrics(ctx context.Context, appender storage.Appender) error
	Shutdown(ctx context.Context) error
}
