package processor

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/grafana/tempo/pkg/tempopb"
)

// TODO review this interface
//  we probably need something to update the configuration as well

type Processor interface {
	Name() string
	PushSpans(ctx context.Context, req *tempopb.PushSpansRequest) error
	Shutdown(ctx context.Context) error

	RegisterMetrics(reg prometheus.Registerer) error
	// TODO can't we just unregister metrics during Shutdown?
	UnregisterMetrics(reg prometheus.Registerer)
}
