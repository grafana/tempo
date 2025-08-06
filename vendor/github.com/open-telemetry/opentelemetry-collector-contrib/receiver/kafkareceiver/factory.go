// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/kafka/configkafka"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver/internal/metadata"
)

const (
	defaultLogsTopic    = "otlp_logs"
	defaultLogsEncoding = "otlp_proto"

	defaultMetricsTopic    = "otlp_metrics"
	defaultMetricsEncoding = "otlp_proto"

	defaultTracesTopic    = "otlp_spans"
	defaultTracesEncoding = "otlp_proto"
)

// NewFactory creates Kafka receiver factory.
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithTraces(createTracesReceiver, metadata.TracesStability),
		receiver.WithMetrics(createMetricsReceiver, metadata.MetricsStability),
		receiver.WithLogs(createLogsReceiver, metadata.LogsStability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		ClientConfig:   configkafka.NewDefaultClientConfig(),
		ConsumerConfig: configkafka.NewDefaultConsumerConfig(),
		Logs: TopicEncodingConfig{
			Topic:    defaultLogsTopic,
			Encoding: defaultLogsEncoding,
		},
		Metrics: TopicEncodingConfig{
			Topic:    defaultMetricsTopic,
			Encoding: defaultMetricsEncoding,
		},
		Traces: TopicEncodingConfig{
			Topic:    defaultTracesTopic,
			Encoding: defaultTracesEncoding,
		},
		MessageMarking: MessageMarking{
			After:   false,
			OnError: false,
		},
		HeaderExtraction: HeaderExtraction{
			ExtractHeaders: false,
		},
	}
}

func createTracesReceiver(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (receiver.Traces, error) {
	return newTracesReceiver(cfg.(*Config), set, nextConsumer)
}

func createMetricsReceiver(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (receiver.Metrics, error) {
	return newMetricsReceiver(cfg.(*Config), set, nextConsumer)
}

func createLogsReceiver(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (receiver.Logs, error) {
	return newLogsReceiver(cfg.(*Config), set, nextConsumer)
}
