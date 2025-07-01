package receiver

import (
	"context"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var _ propagation.TextMapCarrier = (*clientMetadataCarrier)(nil)

// clientMetadataCarrier is a propagation.TextMapCarrier for client.Metadata
type clientMetadataCarrier struct {
	metadata client.Metadata
}

func (c *clientMetadataCarrier) Get(key string) string {
	values := c.metadata.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (c *clientMetadataCarrier) Set(string, string) {} // Not implemented as we only need extraction

func (c *clientMetadataCarrier) Keys() []string {
	var keys []string
	for key := range c.metadata.Keys() {
		keys = append(keys, key)
	}
	return keys
}

var _ propagation.TextMapPropagator = (*onlySampledTraces)(nil)

type onlySampledTraces struct {
	propagation.TextMapPropagator
}

func (o onlySampledTraces) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsSampled() {
		return
	}
	o.TextMapPropagator.Inject(ctx, carrier)
}

func extractTracingContextFromMetadata(ctx context.Context, md client.Metadata) context.Context {
	carrier := &clientMetadataCarrier{metadata: md}
	propagator := onlySampledTraces{otel.GetTextMapPropagator()}
	return propagator.Extract(ctx, carrier)
}
