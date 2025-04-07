// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/kafka/configkafka"
)

var _ component.Config = (*Config)(nil)

// Config defines configuration for Kafka receiver.
type Config struct {
	configkafka.ClientConfig   `mapstructure:",squash"`
	configkafka.ConsumerConfig `mapstructure:",squash"`

	// The name of the kafka topic to consume from (default "otlp_spans" for traces, "otlp_metrics" for metrics, "otlp_logs" for logs)
	Topic string `mapstructure:"topic"`

	// Encoding of the messages (default "otlp_proto")
	Encoding string `mapstructure:"encoding"`

	// Controls the way the messages are marked as consumed
	MessageMarking MessageMarking `mapstructure:"message_marking"`

	// Extract headers from kafka records
	HeaderExtraction HeaderExtraction `mapstructure:"header_extraction"`

	// In case of some errors returned by the next consumer, the receiver will wait and retry the failed message
	ErrorBackOff configretry.BackOffConfig `mapstructure:"error_backoff"`
}

type MessageMarking struct {
	// If true, the messages are marked after the pipeline execution
	After bool `mapstructure:"after"`

	// If false, only the successfully processed messages are marked, it has no impact if
	// After is set to false.
	// Note: this can block the entire partition in case a message processing returns
	// a permanent error.
	OnError bool `mapstructure:"on_error"`
}

type HeaderExtraction struct {
	ExtractHeaders bool     `mapstructure:"extract_headers"`
	Headers        []string `mapstructure:"headers"`
}
