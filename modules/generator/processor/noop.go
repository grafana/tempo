package processor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/model/pdata"
)

// noopTracesProcessor implements component.TracesProcessor
type noopTracesProcessor struct {
}

var _ component.TracesProcessor = (*noopTracesProcessor)(nil)

func (n *noopTracesProcessor) Start(ctx context.Context, host component.Host) error {
	return nil
}

func (n *noopTracesProcessor) Shutdown(ctx context.Context) error {
	return nil
}

func (n *noopTracesProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{}
}

func (n *noopTracesProcessor) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	return nil
}
