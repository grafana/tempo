// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/confmap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/kafka/configkafka"
)

var _ component.Config = (*Config)(nil)

// Config defines configuration for Kafka receiver.
type Config struct {
	configkafka.ClientConfig   `mapstructure:",squash"`
	configkafka.ConsumerConfig `mapstructure:",squash"`

	// Logs holds configuration about how logs should be consumed.
	Logs TopicEncodingConfig `mapstructure:"logs"`

	// Metrics holds configuration about how metrics should be consumed.
	Metrics TopicEncodingConfig `mapstructure:"metrics"`

	// Traces holds configuration about how traces should be consumed.
	Traces TopicEncodingConfig `mapstructure:"traces"`

	// Topic holds the name of the Kafka topic from which to consume data.
	//
	// Topic has no default. If explicitly specified, it will take precedence
	// over the default values of Logs.Topic, Traces.Topic, and Metrics.Topic.
	//
	// Deprecated [v0.124.0]: Use Logs.Topic, Traces.Topic, and Metrics.Topic.
	Topic string `mapstructure:"topic"`

	// Encoding holds the expected encoding of messages (default "otlp_proto")
	//
	// Encoding has no default. If explicitly specified, it will take precedence
	// over the default values of Logs.Encoding, Traces.Encoding, and
	// Metrics.Encoding.
	//
	// Deprecated [v0.124.0]: Use Logs.Encoding, Traces.Encoding, and
	// Metrics.Encoding.
	Encoding string `mapstructure:"encoding"`

	// MessageMarking controls the way the messages are marked as consumed.
	MessageMarking MessageMarking `mapstructure:"message_marking"`

	// HeaderExtraction controls extraction of headers from Kafka records.
	HeaderExtraction HeaderExtraction `mapstructure:"header_extraction"`

	// ErrorBackoff controls backoff/retry behavior when the next consumer
	// returns an error.
	ErrorBackOff configretry.BackOffConfig `mapstructure:"error_backoff"`
}

func (c *Config) Unmarshal(conf *confmap.Conf) error {
	if err := conf.Unmarshal(c); err != nil {
		return err
	}
	// Check if deprecated fields have been explicitly set,
	// in which case they should be used instead of signal-
	// specific defaults.
	var zeroConfig Config
	if err := conf.Unmarshal(&zeroConfig); err != nil {
		return err
	}
	if c.Topic != "" {
		if zeroConfig.Logs.Topic == "" {
			c.Logs.Topic = c.Topic
		}
		if zeroConfig.Metrics.Topic == "" {
			c.Metrics.Topic = c.Topic
		}
		if zeroConfig.Traces.Topic == "" {
			c.Traces.Topic = c.Topic
		}
	}
	if c.Encoding != "" {
		if zeroConfig.Logs.Encoding == "" {
			c.Logs.Encoding = c.Encoding
		}
		if zeroConfig.Metrics.Encoding == "" {
			c.Metrics.Encoding = c.Encoding
		}
		if zeroConfig.Traces.Encoding == "" {
			c.Traces.Encoding = c.Encoding
		}
	}
	return conf.Unmarshal(c)
}

// TopicEncodingConfig holds signal-specific topic and encoding configuration.
type TopicEncodingConfig struct {
	// Topic holds the name of the Kafka topic from which messages of the
	// signal type should be consumed.
	//
	// The default depends on the signal type:
	//  - "otlp_spans" for traces
	//  - "otlp_metrics" for metrics
	//  - "otlp_logs" for logs
	Topic string `mapstructure:"topic"`

	// Encoding holds the expected encoding of messages for the signal type
	//
	// Defaults to "otlp_proto".
	Encoding string `mapstructure:"encoding"`
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
