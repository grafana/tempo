package processor

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/grafana/tempo/pkg/tempopb"
)

type Processor interface {
	// Name returns the name of the processor
	Name() string

	// RegisterMetrics registers metrics that are emitted by this processor.
	RegisterMetrics(reg prometheus.Registerer) error

	// PushSpans processes a batch of spans and updates the metrics register in RegisterMetrics.
	PushSpans(ctx context.Context, req *tempopb.PushSpansRequest) error

	// Shutdown releases any resources allocated by the processor and unregisters metrics registered
	// by RegisterMetrics.
	Shutdown(ctx context.Context, reg prometheus.Registerer) error
}
