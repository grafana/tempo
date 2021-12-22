package processor

import (
	"context"

	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/model/pdata"

	"github.com/grafana/tempo/modules/generator/processor/servicegraphprocessor"
)

type serviceGraphProcessor struct {
	processor component.TracesProcessor
}

var _ Processor = (*serviceGraphProcessor)(nil)

func NewServiceGraphProcessor(reg prometheus.Registerer) (Processor, error) {
	cfg := &servicegraphprocessor.Config{}
	processor := servicegraphprocessor.NewProcessor(&noopTracesProcessor{}, cfg, reg)

	// Start it immediately
	err := processor.Start(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	return &serviceGraphProcessor{processor: processor}, nil
}

func (p *serviceGraphProcessor) Name() string {
	return "serviceGraphProcessor"
}

func (p *serviceGraphProcessor) ConsumeTraces(ctx context.Context, req pdata.Traces) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "serviceGraphProcessor.ConsumeTraces")
	defer span.Finish()

	return p.processor.ConsumeTraces(ctx, req)
}

func (p *serviceGraphProcessor) Shutdown(ctx context.Context) error {
	return p.processor.Shutdown(ctx)
}
