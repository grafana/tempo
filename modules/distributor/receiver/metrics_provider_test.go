package receiver

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func BenchmarkMetricsProvider(b *testing.B) {
	meterProvider := NewMeterProvider()
	meter := meterProvider.Meter("test")
	acceptedSpans, _ := meter.Int64Counter("receiver_accepted_spans")
	c := context.Background()

	otelAttrs := []attribute.KeyValue{
		attribute.String("receiver", "test_receiver"),
		attribute.String("transport", "http"),
	}

	for i := 0; i < b.N; i++ {
		acceptedSpans.Add(c, 2, metric.WithAttributes(otelAttrs...))
	}
}
