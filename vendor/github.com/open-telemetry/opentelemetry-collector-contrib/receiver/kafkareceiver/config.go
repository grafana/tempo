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

	// Profiles holds configuration about how profiles should be consumed.
	Profiles TopicEncodingConfig `mapstructure:"profiles"`

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

	// Telemetry controls optional telemetry configuration.
	Telemetry TelemetryConfig `mapstructure:"telemetry"`
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
		if zeroConfig.Profiles.Topic == "" {
			c.Profiles.Topic = c.Topic
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
		if zeroConfig.Profiles.Encoding == "" {
			c.Profiles.Encoding = c.Encoding
		}
	}

	// Set OnPermanentError default value to inherit from OnError for backward compatibility
	// Only if OnPermanentError was not explicitly set in the config
	rawConf := conf.Get("message_marking")
	if rawConf != nil {
		if messageMarkingConf, ok := rawConf.(map[string]any); ok {
			if _, hasOnPermanentError := messageMarkingConf["on_permanent_error"]; !hasOnPermanentError {
				c.MessageMarking.OnPermanentError = c.MessageMarking.OnError
			}
		}
	} else {
		// If message_marking section doesn't exist, set defaults
		c.MessageMarking.OnPermanentError = c.MessageMarking.OnError
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
	//  - "otlp_profiles" for profiles
	Topic string `mapstructure:"topic"`

	// Encoding holds the expected encoding of messages for the signal type
	//
	// Defaults to "otlp_proto".
	Encoding string `mapstructure:"encoding"`
}

type MessageMarking struct {
	// If true, the messages are marked after the pipeline execution
	After bool `mapstructure:"after"`

	// If false, only the successfully processed messages are marked. This only applies
	// to non-permanent errors. It has no impact if After is set to false.
	// Note: this can block the entire partition in case a message processing returns
	// a non-permanent error.
	OnError bool `mapstructure:"on_error"`

	// If false, only the successfully processed messages are marked. This only applies
	// to permanent errors. It has no impact if After is set to false.
	// Default value inherits from OnError for backward compatibility.
	// Note: this can block the entire partition in case a message processing returns
	// a permanent error.
	OnPermanentError bool `mapstructure:"on_permanent_error"`
}

type HeaderExtraction struct {
	ExtractHeaders bool     `mapstructure:"extract_headers"`
	Headers        []string `mapstructure:"headers"`
}

type TelemetryConfig struct {
	Metrics MetricsConfig `mapstructure:"metrics"`
	_       struct{}      // avoids unkeyed_literal_initialization
}

// MetricsConfig provides config for optional receiver metrics.
type MetricsConfig struct {
	// KafkaReceiverRecordsDelay controls whether the metric kafka_receiver_records_delay
	// that measures the time in seconds between producing and receiving a batch of records
	// will be reported or not. This metric is not reported by default because
	// it may slow down high-volume consuming.
	KafkaReceiverRecordsDelay MetricConfig `mapstructure:"kafka_receiver_records_delay"`
	_                         struct{}     // avoids unkeyed_literal_initialization
}

// MetricConfig provides common config for a particular metric.
type MetricConfig struct {
	Enabled bool     `mapstructure:"enabled"`
	_       struct{} // avoids unkeyed_literal_initialization
}
