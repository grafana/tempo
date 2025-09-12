// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/xreceiver"

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

	defaultProfilesTopic    = "otlp_profiles"
	defaultProfilesEncoding = "otlp_proto"
)

// NewFactory creates Kafka receiver factory.
func NewFactory() receiver.Factory {
	return xreceiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		xreceiver.WithTraces(createTracesReceiver, metadata.TracesStability),
		xreceiver.WithMetrics(createMetricsReceiver, metadata.MetricsStability),
		xreceiver.WithLogs(createLogsReceiver, metadata.LogsStability),
		xreceiver.WithProfiles(createProfilesReceiver, metadata.ProfilesStability),
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
		Profiles: TopicEncodingConfig{
			Topic:    defaultProfilesTopic,
			Encoding: defaultProfilesEncoding,
		},
		MessageMarking: MessageMarking{
			After:            false,
			OnError:          false,
			OnPermanentError: false,
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

func createProfilesReceiver(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer xconsumer.Profiles,
) (xreceiver.Profiles, error) {
	return newProfilesReceiver(cfg.(*Config), set, nextConsumer)
}
