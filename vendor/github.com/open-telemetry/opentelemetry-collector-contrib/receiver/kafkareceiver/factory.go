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
		func() component.Config {
			return createDefaultConfig()
		},
		receiver.WithTraces(createTracesReceiver, metadata.TracesStability),
		receiver.WithMetrics(createMetricsReceiver, metadata.MetricsStability),
		receiver.WithLogs(createLogsReceiver, metadata.LogsStability),
	)
}

func createDefaultConfig() *Config {
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
	r, err := newTracesReceiver(cfg.(*Config), set, nextConsumer)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func createMetricsReceiver(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (receiver.Metrics, error) {
	r, err := newMetricsReceiver(cfg.(*Config), set, nextConsumer)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func createLogsReceiver(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (receiver.Logs, error) {
	r, err := newLogsReceiver(cfg.(*Config), set, nextConsumer)
	if err != nil {
		return nil, err
	}
	return r, nil
}
